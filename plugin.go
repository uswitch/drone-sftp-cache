package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/uswitch/drone-cache/cache"
	"github.com/uswitch/drone-cache/cache/s3"
	"github.com/uswitch/drone-cache/cache/sftp"
)

// Plugin for caching directories to an SFTP server.
type Plugin struct {
	Rebuild bool
	Restore bool
	Mount   []string
	Repo    string
	Branch  string
	Default string // default master branch
	Path    string

	SFTP string
	S3   string
}

func oneAndOnlyOne(options []string) (int, string, error) {
	foundIndex := -1
	var foundValue string

	for index, value := range options {
		if value != "" {
			if foundIndex != -1 {
				return 0, "", errors.New("You can only configure one cache backend.")
			} else {
				foundIndex = index
				foundValue = value
			}
		}
	}

	if foundIndex == -1 {
		return 0, "", errors.New("No configuration for cache backend found.")
	} else {
		return foundIndex, foundValue, nil
	}
}

func (p *Plugin) Exec() error {

	cacheIndex, cacheJSON, err := oneAndOnlyOne([]string{p.SFTP, p.S3})

	if err != nil {
		return err
	}

	var cache cache.Cache
	switch cacheIndex {
	case 0:
		cache, err = sftp.FromJSON(
			cacheJSON,
		)
	case 1:
		cache, err = s3.FromJSON(
			cacheJSON,
		)
	}

	if err != nil {
		return err
	}

	defer cache.(io.Closer).Close()

	if p.Rebuild {
		now := time.Now()
		err = p.ProcessRebuild(cache)
		logrus.Printf("cache built in %v", time.Since(now))
	}

	if p.Restore {
		now := time.Now()
		err = p.ProcessRestore(cache)
		logrus.Printf("cache restored in %v", time.Since(now))
	}

	if err != nil {
		logrus.Println(err)
	}

	return nil
}

// Rebuild the remote cache from the local environment.
func (p Plugin) ProcessRebuild(c cache.Cache) error {
	for _, mount := range p.Mount {
		hash := hasher(mount, p.Branch)
		path := filepath.Join(p.Path, p.Repo, hash)

		log.Printf("archiving directory <%s> to remote cache <%s>", mount, path)

		err := cache.RebuildCmd(c, mount, path)
		if err != nil {
			return err
		}
	}
	return nil
}

// Restore the local environment from the remote cache.
func (p Plugin) ProcessRestore(c cache.Cache) error {
	for _, mount := range p.Mount {
		hash := hasher(mount, p.Branch)
		path := filepath.Join(p.Path, p.Repo, hash)

		log.Printf("restoring directory <%s> from remote cache <%s>", mount, path)

		err := cache.RestoreCmd(c, path, mount)
		if err != nil {

			// this is fallback code to restore from the projects default branch.
			// hash = hasher(mount, "master")
			// path = filepath.Join(p.Path, p.Repo, hash)
			// log.Printf("restoring directory %s from remote cache, using default branch", mount)
			// if xerr := cache.Restore(c, path, mount); xerr != nil {
			return err
			// }
		}
	}
	return nil
}

// helper function to hash a file name based on path and branch.
func hasher(mount, branch string) string {
	parts := []string{mount, branch}

	// calculate the hash using the branch
	h := md5.New()
	for _, part := range parts {
		io.WriteString(h, part)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

FROM alpine:3.4

RUN apk update && \
  apk add \
    ca-certificates && \
  rm -rf /var/cache/apk/*

ADD drone-cache /bin/
ENTRYPOINT ["/bin/drone-cache"]

FROM alpine:3.11 AS builder

RUN apk update && apk add go git

ADD . /src

WORKDIR /src

RUN ./build.sh release -a -ldflags "-linkmode external -extldflags '-static' -s -w"

FROM busybox:1.31.1-musl

COPY --from=builder /src/lightmeter /lightmeter

VOLUME /logs

USER 1000

ENTRYPOINT ["/lightmeter"]

FROM golang:1.14.2-alpine3.11 AS builder

RUN apk update && apk add gcc libc-dev git

ADD . /src

WORKDIR /src

RUN ./build.sh release -a -ldflags "-linkmode external -extldflags '-static' -s -w"

FROM scratch

COPY --from=builder /src/lightmeter /lightmeter

VOLUME /logs

USER 1000

ENTRYPOINT ["/lightmeter"]

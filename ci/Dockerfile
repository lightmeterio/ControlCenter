FROM golang:1.14.2-alpine3.11 AS builder

RUN apk update && apk add gcc libc-dev git make

ADD . /src

WORKDIR /src

RUN make static_release

FROM scratch

COPY --from=builder /src/lightmeter /lightmeter

VOLUME /logs

USER 1000

ENTRYPOINT ["/lightmeter"]

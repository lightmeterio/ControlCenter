FROM alpine:3.11

RUN apk update && apk add git go

VOLUME /src

VOLUME /dst

WORKDIR /src

USER 1000

ENV HOME /tmp

CMD ./build.sh release -a -ldflags "-linkmode external -extldflags '-static' -s -w" && mv lightmeter /dst

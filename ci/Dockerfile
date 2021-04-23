FROM registry.gitlab.com/lightmeter/golang-builder-docker-image:latest AS builder

RUN apk update && apk add ca-certificates

ADD . /src

WORKDIR /src

RUN make static_release

FROM scratch

ARG LIGHTMETER_VERSION
ARG LIGHTMETER_COMMIT
ARG IMAGE_TAG

# List of interesting labels: http://label-schema.org/rc1/
LABEL org.label-schema.name="Lightmeter Control Center"
LABEL org.label-schema.vcs-url="https://gitlab.com/lightmeter/controlcenter"
LABEL org.label-schema.url="https://lightmeter.io"
LABEL org.label-schema.description="Mail server delivery monitoring"
LABEL org.label-schema.usage="https://gitlab.com/lightmeter/controlcenter/-/raw/release/$IMAGE_TAG/README.md"
LABEL org.label-schema.vendor="Lightmeter Project"
LABEL org.label-schema.version="$LIGHTMETER_VERSION"
LABEL org.label-schema.vcs-ref="$LIGHTMETER_COMMIT"
LABEL org.label-schema.schema-version="1.0"
LABEL maintainer="Leandro Santiago <leandro@lightmeter.io>"

COPY --from=builder /src/lightmeter /lightmeter
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

ENV SSL_CERT_DIR /usr/share/ca-certificates/mozilla

VOLUME /logs
VOLUME /workspace
EXPOSE 8080

ENTRYPOINT ["/lightmeter"]

# ---------------------------------------------------------#
#                   Get certificates                       #
# ---------------------------------------------------------#
### Pull CA Certs
FROM --platform=$BUILDPLATFORM alpine:3.18 as cert-image

RUN apk --update add ca-certificates

# ---------------------------------------------------------#
#                   Build Harness image                    #
# ---------------------------------------------------------#
FROM --platform=$BUILDPLATFORM golang:1.22-alpine3.18 as builder
RUN apk update && apk add make build-base git

WORKDIR /app
ARG TARGETOS
ARG TARGETARCH

# Get dependencies - will also be cached if we won't change mod/sum
COPY go.mod .
COPY go.sum .

COPY Makefile .
# If dependencies did not change, this would be cached
RUN make dep

# COPY the source code as the last step as it'll most likely be different
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o release/runner

# ---------------------------------------------------------#
#                   Create final image                     #
# ---------------------------------------------------------#
FROM --platform=$TARGETPLATFORM alpine:3.6 as final
RUN apk add -U --no-cache curl
EXPOSE 3000

WORKDIR /app
VOLUME /data
ENV XDG_CACHE_HOME /data
ENV GODEBUG netdns=go
ENV DRONE_PLATFORM_OS linux
ENV DRONE_PLATFORM_ARCH amd64

COPY --from=cert-image /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/release/runner /bin/runner

LABEL com.centurylinklabs.watchtower.stop-signal="SIGINT"

ENTRYPOINT ["/bin/runner", "server"]

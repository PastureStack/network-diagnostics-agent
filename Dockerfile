FROM golang:1.27.0-bookworm@sha256:484ef6066fa69acb059fdfeda7ba2b8f7391f2ef6abc6f9b8411e669ebd56466 AS build

ARG VERSION=dev
ARG SOURCE_DATE_EPOCH=0
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go test ./... \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/network-diagnostics-agent \
      ./cmd/network-diagnostics-agent \
    && mkdir -p /rootfs/etc/ssl/certs \
    && cp /etc/ssl/certs/ca-certificates.crt /rootfs/etc/ssl/certs/ca-certificates.crt \
    && cp /out/network-diagnostics-agent /rootfs/network-diagnostics-agent \
    && chmod 0755 /rootfs /rootfs/etc /rootfs/etc/ssl /rootfs/etc/ssl/certs /rootfs/network-diagnostics-agent \
    && chmod 0644 /rootfs/etc/ssl/certs/ca-certificates.crt \
    && find /rootfs -exec touch --no-dereference --date="@${SOURCE_DATE_EPOCH}" {} +

FROM scratch

ARG VERSION=dev
ARG SOURCE_DATE_EPOCH=0
LABEL org.opencontainers.image.title="PastureStack Network Diagnostics Agent" \
      org.opencontainers.image.description="Bounded, pseudonymous host network diagnostics collector" \
      org.opencontainers.image.source="https://github.com/PastureStack/network-diagnostics-agent" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="Apache-2.0"
COPY --from=build --chown=65532:65532 /rootfs/ /
USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/network-diagnostics-agent", "run"]

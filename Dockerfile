# Build stage runs on the builder's native platform and cross-compiles to
# TARGETARCH (linux/arm64 for the Oracle A1 VM). modernc.org/sqlite is pure
# Go, so CGO stays off and no QEMU emulation is needed.
FROM --platform=$BUILDPLATFORM golang:1.26 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/gtd-server ./cmd/gtd-server

# distroless/static: no shell, no package manager; runs as uid 65532.
# The host directory mounted at /data must be owned by 65532.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gtd-server /gtd-server
EXPOSE 8080
VOLUME /data
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/gtd-server", "-healthcheck"]
ENTRYPOINT ["/gtd-server"]
CMD ["-addr", ":8080", "-db", "/data/gtd.db"]

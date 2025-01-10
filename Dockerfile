FROM --platform=$BUILDPLATFORM registry-cn-hangzhou.ack.aliyuncs.com/dev/golang:1.23.4 as build

WORKDIR /src
ARG TARGETARCH
ARG TARGETOS
ARG VERSION="unknown"
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    GOARCH=$TARGETARCH GOOS=$TARGETOS CGO_ENABLED=0 \
    go build -trimpath -o /out/csi-resizer -ldflags "-X main.version=$VERSION" ./cmd/csi-resizer

FROM --platform=$TARGETPLATFORM registry-cn-hangzhou.ack.aliyuncs.com/dev/ack-base/distroless/static-debian12:latest
LABEL maintainers="Kubernetes Authors"
LABEL description="CSI External Resizer"

COPY --from=build --link /out/csi-resizer /csi-resizer
ENTRYPOINT ["/csi-resizer"]

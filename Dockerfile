FROM gcr.io/distroless/static:latest
LABEL maintainers="Kubernetes Authors"
LABEL description="CSI External Attacher"
ARG binary=./bin/csi-attacher

COPY ${binary} csi-attacher
ENTRYPOINT ["/csi-attacher"]

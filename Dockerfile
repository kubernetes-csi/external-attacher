FROM gcr.io/distroless/static:latest
LABEL maintainers="Kubernetes Authors"
LABEL description="CSI External Attacher"

COPY ./bin/csi-attacher csi-attacher
ENTRYPOINT ["/csi-attacher"]

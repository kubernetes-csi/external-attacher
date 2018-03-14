FROM alpine
LABEL maintainers="Kubernetes Authors"
LABEL description="CSI External Attacher"

COPY csi-attacher csi-attacher
ENTRYPOINT ["/csi-attacher"]

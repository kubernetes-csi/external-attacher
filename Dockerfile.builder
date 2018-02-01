FROM golang:alpine
LABEL maintainers="Kubernetes Authors"
LABEL description="CSI External Attacher"

WORKDIR /go/src/github.com/kubernetes-csi/external-attacher
COPY . .
RUN cd cmd/csi-attacher && \
    go install

FROM ghcr.io/oracle/oraclelinux:8-slim

COPY rpms /tmp/

RUN microdnf update && \
    microdnf clean all && \
    rpm -i /tmp/*.rpm && \
    rm /tmp/*.rpm

ENTRYPOINT ["/csi-attacher"]

FROM golang:alpine as builder

WORKDIR /opt/ebs-exporter

COPY . .

RUN apk add --no-cache make ca-certificates git
RUN make dist

FROM golang:alpine

COPY --from=builder /opt/ebs-exporter/ebs-exporter          /opt/bin/ebs-exporter
COPY --from=builder /opt/ebs-exporter/config.toml.example   /etc/ebs-exporter/config.toml
COPY --from=builder /etc/ssl/certs/ca-certificates.crt      /etc/ssl/certs/

RUN adduser \
    --disabled-password \
    --home /home/go \
    go

RUN chown go /opt/bin/ebs-exporter /etc/ebs-exporter/config.toml

USER go

ENTRYPOINT ["/opt/bin/ebs-exporter"]

CMD ["--config=/etc/ebs-exporter/config.toml"]

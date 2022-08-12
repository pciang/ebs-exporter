FROM golang:alpine as builder

WORKDIR /opt/misc-exporter

COPY . .

RUN apk add --no-cache make ca-certificates git
RUN make dist

FROM golang:alpine

COPY --from=builder /opt/misc-exporter/misc-exporter          /opt/bin/misc-exporter
COPY --from=builder /opt/misc-exporter/config.toml.example   /etc/misc-exporter/config.toml
COPY --from=builder /etc/ssl/certs/ca-certificates.crt      /etc/ssl/certs/

RUN adduser \
    --disabled-password \
    --home /home/go \
    go

RUN chown go /opt/bin/misc-exporter /etc/misc-exporter/config.toml

USER go

ENTRYPOINT ["/opt/bin/misc-exporter"]

CMD ["--config=/etc/misc-exporter/config.toml"]

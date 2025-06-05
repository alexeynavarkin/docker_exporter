FROM golang:1.21.1-alpine3.18 AS builder

WORKDIR /build
COPY . .
RUN go build -o docker-exporter ./cmd/main.go

FROM alpine:3.18
WORKDIR /docker-exporter
COPY --from=builder /build/docker-exporter .
CMD [ "./docker-exporter" ]
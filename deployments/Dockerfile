FROM --platform=$BUILDPLATFORM golang:1.21.1-alpine3.18 AS builder

WORKDIR /build
COPY . .
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build cmd/main.go

FROM alpine:3.18
WORKDIR /docker-exporter
COPY --from=builder /build/main .
CMD [ "./main" ]
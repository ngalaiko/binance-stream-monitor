ARG GO_VERSION=1.16.0
ARG ALPINE_VERSION=3.13

FROM golang:$GO_VERSION-alpine$ALPINE_VERSION as go-builder

COPY src /src/

WORKDIR /src

RUN go build -o monitor ./cmd/monitor/main.go


FROM alpine:$ALPINE_VERSION

COPY --from=go-builder /src/monitor /app/monitor

ENTRYPOINT ["/app/monitor"]
CMD ["/app/monitor"]

# syntax=docker/dockerfile:1
FROM golang:1.17 as builder
WORKDIR /build
ADD . /build/
RUN --mount=type=cache,target=/root/.cache/go-build make build-for-docker

FROM scratch
WORKDIR /app
COPY --from=builder /build/reorg-monitor /app/reorg-monitor
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV LISTEN_ADDR=":8080"
EXPOSE 8080
CMD ["/app/reorg-monitor"]

FROM golang:1.19-alpine AS builder
ENV GO111MODULE=on
RUN apk --update upgrade \
    && apk --no-cache --no-progress add git ca-certificates libcap \
    && update-ca-certificates
WORKDIR /src
ADD . .
RUN go mod download
RUN go mod verify
RUN mkdir -p /app
ADD www /app/www
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/echohttp ./cmd
RUN addgroup -S -g 10101 appuser
RUN adduser -S -D -u 10101 -s /sbin/nologin -h /appuser -G appuser appuser
RUN chown -R appuser:appuser /app/echohttp

FROM scratch
EXPOSE 8080
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/group /etc/passwd /etc/
COPY --from=builder /app /
USER appuser
ENTRYPOINT ["/echohttp"]

# cap_net_bind_service çalışması için app klasör içinde olmalı, klasör kopyalanmalı.
# https://medium.com/elbstack/docker-go-and-privileged-ports-d6354db472c3

# docker build -t netyazilim/echo-http:1.1.0 -f Dockerfile .




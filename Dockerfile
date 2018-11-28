FROM golang:alpine as builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -mod vendor -a -installsuffix cgo -ldflags="-w -s" -o chat-service

FROM scratch

WORKDIR /app
COPY --from=builder /src/chat-service .
COPY logger/logger-config.json logger/logger-config.json
COPY migrations migrations

VOLUME ["/var/log/dmstudio", "/app/static"]

EXPOSE 8083
CMD ["./chat-service"]

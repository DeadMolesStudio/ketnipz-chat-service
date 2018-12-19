FROM golang:alpine as builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -mod vendor -a -installsuffix cgo -ldflags="-w -s" -o chat-service

FROM alpine

WORKDIR /app
COPY --from=builder /src/chat-service .
COPY logger/logger-config.json logger/logger-config.json
COPY migrations migrations

VOLUME ["/var/log/dmstudio", "/app/static"]

ENV db_connstr ${db_connstr}
ENV db_name ${db_name}
ENV auth_connstr ${auth_connstr}

EXPOSE 8083
CMD ["sh", "-c", "./chat-service -db_connstr ${db_connstr} -db_name ${db_name} -auth_connstr ${auth_connstr}"]

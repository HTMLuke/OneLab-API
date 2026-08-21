FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /onelab-api .

FROM alpine:latest
RUN apk add --no-cache ca-certificates docker-cli \
	&& update-ca-certificates
WORKDIR /
COPY --from=builder /onelab-api /onelab-api
COPY config/config.json /config/config.json
EXPOSE 8080
CMD ["/onelab-api"]
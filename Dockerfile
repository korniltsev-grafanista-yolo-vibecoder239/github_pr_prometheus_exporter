FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /github-pr-exporter .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /github-pr-exporter /usr/local/bin/github-pr-exporter
ENTRYPOINT ["github-pr-exporter"]

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/devmatch-back ./cmd/app

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /build/devmatch-back ./devmatch-back
COPY migrations ./migrations

CMD ["./devmatch-back"]

FROM golang:alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/dbsentinel ./main.go

FROM alpine:latest

RUN apk update && apk add --no-cache ca-certificates

USER nobody

WORKDIR /app

COPY --from=builder --chown=nobody:nobody /app/dbsentinel .

EXPOSE 3425

CMD ["./dbsentinel"]
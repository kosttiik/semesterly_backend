FROM golang:1.24.0-alpine3.21 AS builder

WORKDIR /usr/src/semesterly

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY .env .env

RUN go build -o main ./cmd/semesterly

FROM alpine:3.21

WORKDIR /usr/src/semesterly

COPY --from=builder /usr/src/semesterly/main .

EXPOSE 8080

CMD ["./main"]

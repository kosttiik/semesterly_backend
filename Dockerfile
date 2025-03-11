FROM golang:1.24.0-alpine3.21 AS builder

WORKDIR /usr/src/semesterly

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/semesterly/ ./cmd/semesterly/
COPY internal/ ./internal/
COPY docs/ ./docs/

RUN go build -tags musl -o main ./cmd/semesterly

FROM alpine:3.21

RUN apk add --no-cache tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /usr/src/semesterly

COPY --from=builder /usr/src/semesterly/main .

EXPOSE 8080

CMD ["./main"]

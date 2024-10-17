FROM golang:1.23.2-alpine3.20

WORKDIR /usr/src/semesterly

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/semesterly

EXPOSE 8080

CMD ["./main"]

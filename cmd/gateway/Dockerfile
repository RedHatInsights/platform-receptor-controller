FROM golang:latest

WORKDIR /go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o gateway ./cmd/gateway/main.go

EXPOSE 8080 9090

CMD ["./gateway"]

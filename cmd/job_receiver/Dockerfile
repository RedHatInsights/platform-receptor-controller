FROM golang:latest

WORKDIR /go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o job-receiver ./cmd/job_receiver/main.go

EXPOSE 8081

CMD ["./job-receiver"]

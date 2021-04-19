FROM registry.redhat.io/ubi8/go-toolset

WORKDIR /go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

USER root

RUN go build -o gateway ./cmd/gateway/main.go

RUN go build -o connection-cleaner ./cmd/connection_cleaner/main.go

RUN go build -o job-receiver ./cmd/job_receiver/main.go

RUN go build -o connection-util ./cmd/connection_util/main.go

RUN go build -o response-consumer ./cmd/response_consumer/main.go

RUN REMOVE_PKGS="binutils kernel-headers nodejs nodejs-full-i18n npm" && \
    yum remove -y $REMOVE_PKGS && \
    yum clean all 

USER 1001

EXPOSE 8000 10000 

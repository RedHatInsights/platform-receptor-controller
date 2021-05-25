# Use go-toolset as the builder image
# Once built, copy to a smaller image and run from there
FROM registry.redhat.io/ubi8/go-toolset as builder

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

# Using ubi8-minimal due to its smaller footprint
FROM registry.redhat.io/ubi8/ubi-minimal

WORKDIR /

# Copy executable files from the builder image
COPY --from=builder /go/src/app/gateway /gateway
COPY --from=builder /go/src/app/connection-cleaner /connection-cleaner
COPY --from=builder /go/src/app/job-receiver /job-receiver
COPY --from=builder /go/src/app/connection-util /connection-util
COPY --from=builder /go/src/app/response-consumer /response-consumer

USER 1001

EXPOSE 8000 10000 

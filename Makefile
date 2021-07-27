GATEWAY_BINARY=gateway
JOB_RECEIVER_BINARY=job-receiver

DOCKER_COMPOSE_CFG=docker-compose.yml

COVERAGE_OUTPUT=coverage.out
COVERAGE_HTML=coverage.html

.PHONY: test clean deps coverage $(GATEWAY_BINARY) $(JOB_RECEIVER_BINARY)

build:
	go build -o $(GATEWAY_BINARY) cmd/gateway/main.go
	go build -o $(JOB_RECEIVER_BINARY) cmd/job_receiver/main.go
	go build -o response_consumer cmd/response_consumer/main.go
	go build -o connection_util cmd/connection_util/main.go
	go build -o connection_cleaner cmd/connection_cleaner/main.go

deps:
	go get -u golang.org/x/lint/golint

test:
	# Use the following command to run specific tests (not the entire suite)
	# TEST_ARGS="-run TestReadMessage -v" make test
	go test $(TEST_ARGS) ./...

coverage:
	go test -v -coverprofile=$(COVERAGE_OUTPUT) ./...
	go tool cover -html=$(COVERAGE_OUTPUT) -o $(COVERAGE_HTML)
	@echo "file://$(PWD)/$(COVERAGE_HTML)"

start-test-env:
	podman-compose -f $(DOCKER_COMPOSE_CFG) up

stop-test-env:
	podman-compose -f $(DOCKER_COMPOSE_CFG) down

fmt:
	go fmt ./...

lint:
	$(GOPATH)/bin/golint ./...

clean:
	go clean
	rm -f $(GATEWAY_BINARY) $(JOB_RECEIVER_BINARY)
	rm -f $(COVERAGE_OUTPUT) $(COVERAGE_HTML)

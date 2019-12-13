GATEWAY_BINARY=receptor-controller-gateway

DOCKER_COMPOSE_CFG=docker-compose.yml

.PHONY: test clean deps coverage $(GATEWAY_BINARY)

run: 
	go build -o gateway cmd/gateway/main.go
	go build -o job-receiver cmd/job_receiver/main.go

deps:
	go get -u golang.org/x/lint/golint

test:
	# Use the following command to run specific tests (not the entire suite)
	# TEST_ARGS="-run TestReadMessage -v" make test
	go test $(TEST_ARGS) ./...

coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "file://$(PWD)/coverage.html"

start-test-env:
	docker-compose -f $(DOCKER_COMPOSE_CFG) up

stop-test-env:
	docker-compose -f $(DOCKER_COMPOSE_CFG) down

fmt:
	go fmt ./...

lint:
	$(GOPATH)/bin/golint ./...

clean:
	go clean
	rm -f $(GATEWAY_BINARY)

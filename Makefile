GATEWAY_BINARY=receptor-controller-gateway

.PHONY: test clean deps coverage $(GATEWAY_BINARY)


run: $(GATEWAY_BINARY)
	./$<

$(GATEWAY_BINARY):
	go build -o $@

deps:
	go get -u golang.org/x/lint/golint
	go get -u github.com/google/uuid
	go get -u github.com/spf13/viper
	go get -u github.com/segmentio/kafka-go

test:
	# Use the following command to run specific tests (not the entire suite)
	# TEST_ARGS="-run TestReadMessage -v" make test
	go test $(TEST_ARGS) ./...

coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "file://$(PWD)/coverage.html"

fmt:
	go fmt ./...

lint:
	$(GOPATH)/bin/golint ./...

clean:
	go clean
	rm -f $(GATEWAY_BINARY)

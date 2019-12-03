.PHONY: test clean deps coverage

GATEWAY_BINARY=receptor-controller-gateway


run: $(GATEWAY_BINARY)
	./$<

$(GATEWAY_BINARY):
	go build -o $@

deps:
	go get -u golang.org/x/lint/golint
	go get -u github.com/google/uuid

test:
	#go test $(TEST_ARGS) ./...
	#go test -v -run InvalidFrameType github.com/RedHatInsights/platform-receptor-controller/receptor/protocol
	go test -v -run TestReadMessage github.com/RedHatInsights/platform-receptor-controller/receptor/protocol

coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

lint:
	$(GOPATH)/bin/golint ./...

clean:
	go clean
	rm -f $(GATEWAY_BINARY)

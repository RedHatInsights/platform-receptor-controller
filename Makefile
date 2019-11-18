.PHONY: test clean

GATEWAY_BINARY=receptor-controller-gateway

run: $(GATEWAY_BINARY)
	./$<

$(GATEWAY_BINARY): main.go ws_controller.go management.go job_receiver.go
	go build -o $@

test:
	go test -v ./...

clean:
	rm -f $(GATEWAY_BINARY)

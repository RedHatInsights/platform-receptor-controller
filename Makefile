run: receptor-controller
	./receptor-controller

receptor-controller: main.go ws_controller.go management.go job_receiver.go
	go build -o receptor-controller

test:
	go test -v receptor/mesh_router/router_test.go receptor/mesh_router/heap_test.go

clean:
	rm -f receptor-controller

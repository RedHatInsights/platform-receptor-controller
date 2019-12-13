package mesh_router

import (
	"testing"
)

func TestRouter(t *testing.T) {
	router := NewMeshRouter()
	router.AddEdge("A", "B", 4)
	router.AddEdge("A", "C", 10)
	router.AddEdge("B", "C", 1)
}

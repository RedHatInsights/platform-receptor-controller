package mesh_router

import (
	"github.com/RedHatInsights/platform-receptor-controller/receptor/mesh_router"
	"testing"
)

func TestRouter(t *testing.T) {
	router := mesh_router.NewMeshRouter()
	router.AddEdge("A", "B", 4)
	router.AddEdge("A", "C", 10)
	router.AddEdge("B", "C", 1)
}

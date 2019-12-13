package mesh_router

import "fmt"

type Edge struct {
	Left  *string
	Right *string
	/*
	   Left *node
	   Right *node
	*/
	Cost int
}

// From here:  https://blog.golang.org/go-maps-in-action
type EdgeKey struct {
	Left, Right string
}

/*
type node struct {
  Name string
}
*/

type MeshRouter struct {
	edges map[EdgeKey]int
	nodes map[string]string
}

func NewMeshRouter() *MeshRouter {
	return &MeshRouter{
		edges: make(map[EdgeKey]int),
		nodes: make(map[string]string),
	}
}

func (mr *MeshRouter) GetRouteToNode(from_node string, to_node string) []*string {
	return nil
	/*
	   visited := make(map[string]struct{}{})
	   for len(visited) < len(mr.nodes) {
	   }
	*/
}

func (mr *MeshRouter) AddEdge(left string, right string, cost int) {
	fmt.Println("** MeshRouter.AddEdge()")
	edge_key := EdgeKey{Left: left, Right: right}
	existing_cost, exists := mr.edges[edge_key]
	if exists == false {
		fmt.Println("Adding a new edge...")
		mr.edges[edge_key] = cost
	} else if exists && cost < existing_cost {
		fmt.Println("New cost is less than the existing cost...updating cost...")
		mr.edges[edge_key] = cost
	} else {
		fmt.Println("Existing edge...no-op...")
	}
}

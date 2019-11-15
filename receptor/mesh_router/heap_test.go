package mesh_router

import (
	"container/heap"
	"testing"
)

func TestHeap(t *testing.T) {

	expected_minimum := 1

	p1 := Path{Cost: expected_minimum, Nodes: []string{"A"}}
	p2 := Path{Cost: 2, Nodes: []string{"A", "B"}}
	p3 := Path{Cost: 3, Nodes: []string{"A", "B", "C"}}
	h := &PathHeap{p1, p2, p3}
	heap.Init(h)
	heap.Push(h, Path{Cost: 4})

	minimum := (*h)[0]
	if minimum.Cost != expected_minimum {
		t.Errorf("Minimum was incorrect, got: %d, want: %d.", minimum.Cost, expected_minimum)
	}

	t.Logf("minimum: %d\n", (*h)[0].Cost)
	for h.Len() > 0 {
		t.Logf("%d ", heap.Pop(h))
	}
}

package mesh_router

type Path struct {
	Cost  int
	Nodes []string
}

// An PathHeap is a min-heap of ints.
type PathHeap []Path

func (h PathHeap) Len() int           { return len(h) }
func (h PathHeap) Less(i, j int) bool { return h[i].Cost < h[j].Cost }
func (h PathHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *PathHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(Path))
}

func (h *PathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

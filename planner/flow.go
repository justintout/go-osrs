package planner

import (
	"container/heap"
	"math"
	"sort"
)

// candidatesPerSegment bounds how many capped methods each segment
// considers. Past the cheapest few, a capped method could only matter if
// all of them ran out, and the cheapest uncapped method still covers it.
const candidatesPerSegment = 8

// assign gives each segment's XP to methods at minimum total cost.
//
// Uncapped methods need no search: each segment falls back to its
// cheapest one. Capped methods compete through a minimum-cost flow
// (source to capped method, capped method to segment, segment to sink,
// plus a fallback edge from source to segment at the uncapped cost), so a
// capped method's limited XP goes to the segments where it saves the most.
// It returns assign[m][s], the XP of method m in segment s, and the XP no
// method could cover.
func assign(methods []Method, segs []segment, cost [][]float64) ([][]float64, float64) {
	nm, ns := len(methods), len(segs)
	out := make([][]float64, nm)
	for m := range out {
		out[m] = make([]float64, ns)
	}
	capped := func(m int) bool { return methods[m].MaxActions > 0 && methods[m].XP > 0 }

	// fallback[s] is the cheapest uncapped method in segment s, or -1.
	fallback := make([]int, ns)
	for s := range segs {
		fallback[s] = -1
		for m := range methods {
			c := cost[m][s]
			if !capped(m) && !math.IsNaN(c) && (fallback[s] < 0 || c < cost[fallback[s]][s]) {
				fallback[s] = m
			}
		}
	}

	// Capped methods worth considering in each segment.
	type cand struct {
		m int
		c float64
	}
	cands := make([][]cand, ns)
	used := map[int]bool{}
	for s := range segs {
		for m := range methods {
			c := cost[m][s]
			if !capped(m) || math.IsNaN(c) || (fallback[s] >= 0 && c >= cost[fallback[s]][s]) {
				continue
			}
			cands[s] = append(cands[s], cand{m, c})
		}
		sort.Slice(cands[s], func(i, j int) bool { return cands[s][i].c < cands[s][j].c })
		if len(cands[s]) > candidatesPerSegment {
			cands[s] = cands[s][:candidatesPerSegment]
		}
		for _, c := range cands[s] {
			used[c.m] = true
		}
	}

	// Nodes: 0 source, 1..k capped methods, then segments, then sink.
	var cm []int
	for m := range methods {
		if used[m] {
			cm = append(cm, m)
		}
	}
	idx := map[int]int{}
	for i, m := range cm {
		idx[m] = 1 + i
	}
	segNode := func(s int) int { return 1 + len(cm) + s }
	src, sink := 0, 1+len(cm)+ns
	g := newGraph(sink + 1)
	for _, m := range cm {
		g.add(src, idx[m], float64(methods[m].MaxActions)*methods[m].XP, 0)
	}
	need := 0.0
	fallbackEdge := make([]int, ns)
	for s, sg := range segs {
		for _, c := range cands[s] {
			g.add(idx[c.m], segNode(s), math.Inf(1), c.c)
		}
		fallbackEdge[s] = -1
		if fallback[s] >= 0 {
			fallbackEdge[s] = len(g.adj[src])
			g.add(src, segNode(s), math.Inf(1), cost[fallback[s]][s])
		}
		length := float64(sg.to - sg.from)
		g.add(segNode(s), sink, length, 0)
		need += length
	}
	unmet := g.minCostFlow(src, sink, need)

	for s := range segs {
		if e := fallbackEdge[s]; e >= 0 {
			out[fallback[s]][s] += g.flow(src, e)
		}
	}
	for _, m := range cm {
		for i, e := range g.adj[idx[m]] {
			if s := e.to - 1 - len(cm); s >= 0 && s < ns {
				out[m][s] += g.flow(idx[m], i)
			}
		}
	}
	return out, unmet
}

type edge struct {
	to, rev int
	cap, c  float64
	orig    float64
}

type graph struct{ adj [][]edge }

func newGraph(n int) *graph { return &graph{adj: make([][]edge, n)} }

func (g *graph) add(u, v int, cap, c float64) {
	g.adj[u] = append(g.adj[u], edge{v, len(g.adj[v]), cap, c, cap})
	g.adj[v] = append(g.adj[v], edge{u, len(g.adj[u]) - 1, 0, -c, 0})
}

// flow is the amount sent along the i-th edge out of u.
func (g *graph) flow(u, i int) float64 {
	e := g.adj[u][i]
	if math.IsInf(e.orig, 1) {
		return g.adj[e.to][e.rev].cap
	}
	return e.orig - e.cap
}

// minCostFlow sends up to need from src to sink along successive shortest
// paths, using Dijkstra with potentials. It returns the amount it could
// not send. The graph starts acyclic, so the first potentials come from
// one pass in node order; edges only run from lower to higher nodes then.
func (g *graph) minCostFlow(src, sink int, need float64) float64 {
	n := len(g.adj)
	pot := make([]float64, n)
	for i := range pot {
		pot[i] = math.Inf(1)
	}
	pot[src] = 0
	for u := range n {
		if math.IsInf(pot[u], 1) {
			continue
		}
		for _, e := range g.adj[u] {
			if e.cap > 0 && e.to > u && pot[u]+e.c < pot[e.to] {
				pot[e.to] = pot[u] + e.c
			}
		}
	}
	for i := range pot {
		if math.IsInf(pot[i], 1) {
			pot[i] = 0
		}
	}

	const tol = 0.5 // XP; plans round to whole XP
	dist := make([]float64, n)
	prevNode, prevEdge := make([]int, n), make([]int, n)
	for need > tol {
		for i := range dist {
			dist[i], prevNode[i] = math.Inf(1), -1
		}
		dist[src] = 0
		pq := &nodeHeap{{src, 0}}
		for pq.Len() > 0 {
			cur := heap.Pop(pq).(item)
			if cur.d > dist[cur.n] {
				continue
			}
			for i, e := range g.adj[cur.n] {
				if e.cap <= 1e-9 {
					continue
				}
				// Reduced costs are nonnegative up to float error.
				nd := dist[cur.n] + math.Max(e.c+pot[cur.n]-pot[e.to], 0)
				if nd < dist[e.to] {
					dist[e.to], prevNode[e.to], prevEdge[e.to] = nd, cur.n, i
					heap.Push(pq, item{e.to, nd})
				}
			}
		}
		if math.IsInf(dist[sink], 1) {
			break
		}
		for i := range pot {
			if !math.IsInf(dist[i], 1) {
				pot[i] += dist[i]
			}
		}
		push := need
		for v := sink; v != src; v = prevNode[v] {
			push = math.Min(push, g.adj[prevNode[v]][prevEdge[v]].cap)
		}
		for v := sink; v != src; v = prevNode[v] {
			e := &g.adj[prevNode[v]][prevEdge[v]]
			e.cap -= push
			g.adj[v][e.rev].cap += push
		}
		need -= push
	}
	return math.Max(need, 0)
}

type item struct {
	n int
	d float64
}

type nodeHeap []item

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(i, j int) bool { return h[i].d < h[j].d }
func (h nodeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *nodeHeap) Push(x any)        { *h = append(*h, x.(item)) }
func (h *nodeHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}

// Copyright(c) 2016 Ethan Zhuang <zhuangwj@gmail.com>.

package goraph

import (
	"fmt"
	"github.com/starwander/GoFibonacciHeap"
	"math"
)

// Dijkstra gets the shortest path from one vertex to all other vertices in the graph.
// https://en.wikipedia.org/wiki/Dijkstra%27s_algorithm
func (graph *Graph) Dijkstra(source ID) (dist map[ID]float64, prev map[ID]ID, err error) {
	if _, exists := graph.vertices[source]; !exists {
		return nil, nil, fmt.Errorf("Vertex %v is not existed", source)
	}

	dist = make(map[ID]float64)
	prev = make(map[ID]ID)
	heap := fibHeap.NewFibHeap()

	for id := range graph.vertices {
		prev[id] = nil
		if id != source {
			dist[id] = math.Inf(1)
			heap.Insert(id, math.Inf(1))
		} else {
			dist[id] = 0
			heap.Insert(id, 0)
		}
	}

	for heap.Num() != 0 {
		min, _ := heap.ExtractMin()
		for to, i := range graph.egress[min] {
			for _, edge := range(i) {
				if edge.GetWeight() < 0 {
					return nil, nil, fmt.Errorf("Negative weight form vertex %v to vertex %v is not allowed", min, to)
				}

				if !edge.enable {
					continue
				}

				prevWeight := edge.GetWeight()
				if edge.GetWeight() > 0 {
					edge.SetWeight(edge.GetWeight() + 500000)
				}

				if dist[min]+edge.GetWeight() < dist[to] {
					heap.DecreaseKey(to, dist[min]+edge.GetWeight())
					prev[to] = min

					dist[to] = dist[min] + edge.GetWeight()
				}

				edge.SetWeight(prevWeight)
			}
		}
	}

	return
}

func getPath(prev map[ID]ID, lastNode ID) (path []ID) {
	prevNode := prev[lastNode]
	if prevNode == nil {
		return nil
	}

	reversePath := []ID{lastNode}
	for ; prevNode != nil; prevNode = prev[prevNode] {
		reversePath = append(reversePath, prevNode)
	}

	path = make([]ID, len(reversePath))
	for index, node := range reversePath {
		path[len(reversePath)-index-1] = node
	}

	return
}

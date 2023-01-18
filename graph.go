// Copyright(c) 2016 Ethan Zhuang <zhuangwj@gmail.com>.

package goraph

import (
	"fmt"
	"math"
)

var vIndex = 0

// ID uniquely identify a vertex.
type ID interface{}

// Vertex interface represents a vertex with edges connected to it.
type Vertex interface {
	// ID get the unique id of the vertex.
	ID() ID
	// Edges get all the edges connected to the vertex
	Edges() []Edge
}

// Edge interface represents an edge connecting two vertices.
//type Edge interface {
//	Get returns the edge's inbound vertex, outbound vertex and weight.
//Get() (from ID, to ID, weight float64)
//}

// Graph is made up of vertices and edges.
// Vertices in the graph must have an unique id.
// Each edges in the graph connects two vertices directed with a weight.
type Graph struct {
	vertices map[ID]*vertex
	egress   map[ID]map[ID][]*Edge
}

type vertex struct {
	enable bool
	Index  int
	ID     string
	Name   string
}

type Edge struct {
	tempWeight float64 // In case we temp set the weight to 0 (to force path re-use) we store the old value here, so we can put it back.
	weight     float64 // in our implementation used for the active bandwidth on the link, in megabits. So 10000 = 10 gigabit of traffic
	maxWeight  float64 // in our implementation used for the maximum allowed bandwidth on the link, in megabits.
	// IMPORTANT: any reservations will be subtracted here, so this does NOT equal the port speed.
	enable  bool
	changed bool

	TargetVerget *vertex
	SourceVertex *vertex

	LocalPort    string // LocalPort is the port on THIS switch (vertex) that is connected to the OTHER switch
	LocalPortID  string // uuid of the localPort in TFC API
	RemotePort   string // RemotePort is the port on the OTHER switch that is connected to this switch.
	RemotePortID string // uuid of the remotePort in TFC API
}

func (edge *Edge) GetWeight() float64 {
	return edge.weight
}

func (edge *Edge) SetTempWeight0() {
	edge.tempWeight = edge.weight
	edge.weight = 0
}

func (edge *Edge) ResetTempWeight0() {
	edge.weight = edge.tempWeight
	edge.tempWeight = 0
}

func (edge *Edge) SetWeight(weight float64) {
	edge.weight = weight
}

func (edge *Edge) GetMaxWeight() float64 {
	return edge.maxWeight
}

func (edge *Edge) SetMaxWeight(weight float64) {
	edge.maxWeight = weight
}

// NewGraph creates a new empty graph.
func NewGraph() *Graph {
	graph := new(Graph)
	graph.vertices = make(map[ID]*vertex)
	graph.egress = make(map[ID]map[ID][]*Edge)

	return graph
}

// GetVertex get a vertex by input id.
// Try to get a vertex not in the graph will get an error.
func (graph *Graph) HasVertex(id ID) bool {
	if _, exists := graph.vertices[id]; exists {
		return true
	}

	return false
}

func (graph *Graph) HasEdge(from ID, to ID, localPort, remotePort string) bool {
	if edges, exists := graph.egress[from][to]; exists {
		for _, edge := range edges {
			if edge.LocalPort == localPort && edge.RemotePort == remotePort {
				return true
			}
		}
	}

	return false
}

func (graph *Graph) GetEdge(from ID, to ID, localPort, remotePort string) (*Edge, error) {
	if edges, exists := graph.egress[from][to]; exists {
		for _, edge := range edges {
			if edge.LocalPort == localPort && edge.RemotePort == remotePort {
				return edge, nil
			}
		}
	}

	return nil, fmt.Errorf("Edge between %s (%s) and %s (%s) do not exist", from, to, localPort, remotePort)
}

// GetVertex get a vertex by input id.
// Try to get a vertex not in the graph will get an error.
func (graph *Graph) GetVertex(id ID) (vertex *vertex, err error) {
	if v, exists := graph.vertices[id]; exists {
		vertex = v
		return
	}

	err = fmt.Errorf("Vertex %v is not found", id)
	return
}

// GetEdge gets the edge between the two vertices by input ids.
// Try to get the edge from or to a vertex not in the graph will get an error.
// Try to get the edge between two disconnected vertices will get an error.
func (graph *Graph) GetEdges(from ID, to ID) ([]*Edge, error) {
	if _, exists := graph.vertices[from]; !exists {
		return nil, fmt.Errorf("Vertex(from) %v is not found", from)
	}

	if _, exists := graph.vertices[to]; !exists {
		return nil, fmt.Errorf("Vertex(to) %v is not found", to)
	}

	if edge, exists := graph.egress[from][to]; exists {
		var res []*Edge

		// Only add the enabled edges to the result.
		for _, e := range edge {
			if e.IsEnabled() {
				res = append(res, e)
			}
		}

		return res, nil
	}

	return nil, fmt.Errorf("Edge from %v to %v is not found", from, to)
}

// GetEdgeWeight gets the weight of the edge between the two vertices by input ids.
// Try to get the weight of the edge from or to a vertex not in the graph will get an error.
// Try to get the weight of the edge between two disconnected vertices will get +Inf.
func (graph *Graph) GetEdgeWeight(from ID, to ID) (float64, error) {
	if _, exists := graph.vertices[from]; !exists {
		return math.Inf(1), fmt.Errorf("Vertex(from) %v is not found", from)
	}

	if _, exists := graph.vertices[to]; !exists {
		return math.Inf(1), fmt.Errorf("Vertex(to) %v is not found", to)
	}

	// @TODO
	//if edge, exists := graph.egress[from][to]; exists {
	//	return edge.weight, nil
	//}

	return math.Inf(1), nil
}

func (graph *Graph) GetAllVertices() map[ID]*vertex {
	return graph.vertices
}

func (graph *Graph) GetAllEdges() map[ID]map[ID][]*Edge {
	return graph.egress
}

// AddVertex adds a new vertex into the graph.
// Try to add a duplicate vertex will get an error.
func (graph *Graph) AddVertex(id ID, name string) error {
	if _, exists := graph.vertices[id]; exists {
		return fmt.Errorf("Vertex %v is duplicate", id)
	}

	graph.vertices[id] = &vertex{true, vIndex, id.(string), name}
	graph.egress[id] = make(map[ID][]*Edge)

	vIndex = vIndex + 1

	return nil
}

// AddEdge adds a new edge between the vertices by the input ids.
// Try to add an edge with -Inf weight will get an error.
// Try to add an edge from or to a vertex not in the graph will get an error.
// Try to add a duplicate edge will get an error.
func (graph *Graph) AddEdge(from ID, to ID, localPort, localPortId, remotePort, remotePortId string, weight float64, maxWeight float64) error {
	if weight == math.Inf(-1) {
		return fmt.Errorf("-inf weight is reserved for internal usage")
	}

	if _, exists := graph.vertices[from]; !exists {
		return fmt.Errorf("Vertex(from) %v is not found", from)
	}

	if _, exists := graph.vertices[to]; !exists {
		return fmt.Errorf("Vertex(to) %v is not found", to)
	}

	if graph.HasEdge(from, to, localPort, remotePort) {
		return fmt.Errorf("Edge from %v (%s) to %v (%s) is duplicate", from, localPort, to, remotePort)
	}

	sourceVertex, err := graph.GetVertex(from)
	if err != nil {
		return fmt.Errorf("Vertex(from) %v is not found", from)
	}

	targetVertex, err := graph.GetVertex(to)
	if err != nil {
		return fmt.Errorf("Vertex(to) %v is not found", to)
	}

	// If we got this far, we can create the edge memory pointer and assign it.
	edge := &Edge{
		weight:       weight,
		maxWeight:    maxWeight,
		enable:       true,
		changed:      false,
		SourceVertex: sourceVertex,
		LocalPort:    localPort,
		LocalPortID:  localPortId,
		TargetVerget: targetVertex,
		RemotePort:   remotePort,
		RemotePortID: remotePortId,
	}

	graph.egress[from][to] = append(graph.egress[from][to], edge)
	//graph.ingress[to][from] = edge

	return nil
}

// UpdateEdgeWeight updates the weight of the edge between vertices by the input ids.
// Try to update an edge with -Inf weight will get an error.
// Try to update an edge from or to a vertex not in the graph will get an error.
// Try to update an edge between disconnected vertices will get an error.
/*func (graph *Graph) UpdateEdgeWeight(from ID, to ID, weight float64) error {
	if weight == math.Inf(-1) {
		return fmt.Errorf("-inf weight is reserved for internal usage")
	}

	if _, exists := graph.vertices[from]; !exists {
		return fmt.Errorf("Vertex(from) %v is not found", from)
	}

	if _, exists := graph.vertices[to]; !exists {
		return fmt.Errorf("Vertex(to) %v is not found", to)
	}

	if edge, exists := graph.egress[from][to]; exists {
		edge.weight = weight
		return nil
	}

	return fmt.Errorf("Edge from %v to %v is not found", from, to)
}*/

// DeleteVertex deletes a vertex from the graph and gets the value of the vertex.
// Try to delete a vertex not in the graph will get an nil.
func (graph *Graph) DeleteVertex(id ID) interface{} {
	if vertex, exists := graph.vertices[id]; exists {
		//for to := range graph.egress[id] {
		//	delete(graph.ingress[to], id)
		//}
		//for from := range graph.ingress[id] {
		//	delete(graph.egress[from], id)
		//}
		delete(graph.egress, id)
		//delete(graph.ingress, id)
		delete(graph.vertices, id)

		return vertex
	}

	return nil
}

// DeleteEdge deletes the edge between the vertices by the input id from the graph and gets the value of edge.
// Try to delete an edge from or to a vertex not in the graph will get an error.
// Try to delete an edge between disconnected vertices will get a nil.
func (graph *Graph) DeleteEdge(from ID, to ID) interface{} {
	if _, exists := graph.vertices[from]; !exists {
		return nil
	}

	if _, exists := graph.vertices[to]; !exists {
		return nil
	}

	if edge, exists := graph.egress[from][to]; exists {
		delete(graph.egress[from], to)
		//delete(graph.ingress[to], from)
		return edge
	}

	return nil
}

// DeleteEdge deletes the edge between the vertices by the input id from the graph and gets the value of edge.
// Try to delete an edge from or to a vertex not in the graph will get an error.
// Try to delete an edge between disconnected vertices will get a nil.
func (graph *Graph) DeleteEdgeByLocalPort(from ID, localPort string) *Edge {
	// First we need a list of all the vertices we have
	all := graph.GetAllVertices()

	// Now we need to find where this vertex is linked to any other vertex
	for to, _ := range all {
		if _, exists := graph.egress[from][to]; exists { // If the link exists
			for i, edge := range graph.egress[from][to] { // Range over all the edges
				if edge.LocalPort == localPort { // If the localport matches
					graph.egress[from][to] = append(graph.egress[from][to][:i], graph.egress[from][to][i+1:]...) //slick trick, to kick out this guy.

					return edge
				}
			}
		}
	}

	return nil
}

// AddVertexWithEdges adds a vertex value which implements Vertex interface.
// AddVertexWithEdges adds edges connected to the vertex at the same time, due to the Vertex interface can get the Edges.
/*func (graph *Graph) AddVertexWithEdges(v Vertex) error {
	if _, exists := graph.vertices[v.ID()]; exists {
		return fmt.Errorf("Vertex %v is duplicate", v.ID())
	}

	graph.vertices[v.ID()] = &vertex{true}
	graph.egress[v.ID()] = make(map[ID]*Edge)
	graph.ingress[v.ID()] = make(map[ID]*Edge)

	for _, eachEdge := range v.Edges() {
		from, to, weight := eachEdge.Get()
		if weight == math.Inf(-1) {
			return fmt.Errorf("-inf weight is reserved for internal usage")
		}
		if from != v.ID() && to != v.ID() {
			return fmt.Errorf("Edge from %v to %v is unrelated to the vertex %v", from, to, v.ID())
		}

		if _, exists := graph.egress[to]; !exists {
			graph.egress[to] = make(map[ID]*Edge)
		}
		if _, exists := graph.egress[from]; !exists {
			graph.egress[from] = make(map[ID]*Edge)
		}
		if _, exists := graph.ingress[from]; !exists {
			graph.ingress[from] = make(map[ID]*Edge)
		}
		if _, exists := graph.ingress[to]; !exists {
			graph.ingress[to] = make(map[ID]*Edge)
		}

		graph.egress[from][to] = &edge{weight, true, false}
		graph.ingress[to][from] = graph.egress[from][to]
	}

	return nil
}*/

// CheckIntegrity checks if any edge connects to or from unknown vertex.
// If the graph is integrate, nil is returned. Otherwise an error is returned.
/*func (graph *Graph) CheckIntegrity() error {
	for from, out := range graph.egress {
		if _, exists := graph.vertices[from]; !exists {
			return fmt.Errorf("Vertex %v is not found", from)
		}
		for to := range out {
			if _, exists := graph.vertices[to]; !exists {
				return fmt.Errorf("Vertex %v is not found", to)
			}
		}
	}

	for to, in := range graph.ingress {
		if _, exists := graph.vertices[to]; !exists {
			return fmt.Errorf("Vertex %v is not found", to)
		}
		for from := range in {
			if _, exists := graph.vertices[from]; !exists {
				return fmt.Errorf("Vertex %v is not found", from)
			}
		}
	}

	return nil
}*/

// GetPathWeight gets the total weight along the path by input ids.
// It will get -Inf if the input path is nil or empty.
// It will get -Inf if the path contains vertex not in the graph.
// It will get +Inf if the path contains vertices not connected.
func (graph *Graph) GetPathWeight(path []ID) (totalWeight float64) {
	if len(path) == 0 {
		return math.Inf(-1)
	}

	if _, exists := graph.vertices[path[0]]; !exists {
		return math.Inf(-1)
	}

	for i := 0; i < len(path)-1; i++ {
		if _, exists := graph.vertices[path[i+1]]; !exists {
			return math.Inf(-1)
		}
		if x, exists := graph.egress[path[i]][path[i+1]]; exists {
			for _, edge := range x {
				totalWeight += edge.GetWeight()
			}
		} else {
			return math.Inf(1)
		}
	}

	return totalWeight
}

// DisableEdge disables the edge for further calculation.
func (graph *Graph) DisableEdge(from, to ID) {
	// We only use this if we are finding multiple paths, which we don't
	// therefor left out, for now.

	//for _, edge := range graph.egress[from][to] {
	//	edge.enable = false
	//}
}

func (edge *Edge) IsDisabled() bool {
	return !edge.enable
}

func (edge *Edge) IsEnabled() bool {
	return edge.enable
}

func (edge *Edge) Disable() {
	edge.enable = false
}

func (edge *Edge) Enable() {
	edge.enable = true
}

// DisableVertex disables the vertex for further calculation.
func (graph *Graph) DisableVertex(vertex ID) {
	for _, edges := range graph.egress[vertex] {
		for _, edge := range edges {
			edge.enable = false
		}
	}
}

// EnableVertex disables the vertex for further calculation.
func (graph *Graph) EnableVertex(vertex ID) {
	for _, edges := range graph.egress[vertex] {
		for _, edge := range edges {
			edge.enable = true
		}
	}
}

// DisablePath disables all the vertices in the path for further calculation.
func (graph *Graph) DisablePath(path []ID) {
	for _, vertex := range path {
		graph.DisableVertex(vertex)
	}
}

// Reset enables all vertices and edges for further calculation.
func (graph *Graph) Reset() {
	//for _, out := range graph.egress {
	//	for _, edge := range out {
	//		edge.enable = true
	//	}
	//}
}

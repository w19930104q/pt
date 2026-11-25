package pt

import (
	"math"
	"math/rand"
	"testing"
)

// octahedronMesh returns a regular octahedron (8 triangles) centered at the
// origin with vertices on the unit axes.
func octahedronMesh() *Mesh {
	v := []Vector{
		{1, 0, 0}, {-1, 0, 0},
		{0, 1, 0}, {0, -1, 0},
		{0, 0, 1}, {0, 0, -1},
	}
	faces := [][3]int{
		{0, 2, 4}, {2, 1, 4}, {1, 3, 4}, {3, 0, 4},
		{2, 0, 5}, {1, 2, 5}, {3, 1, 5}, {0, 3, 5},
	}
	triangles := make([]*Triangle, 0, len(faces))
	for _, f := range faces {
		t := NewTriangle(v[f[0]], v[f[1]], v[f[2]], Vector{}, Vector{}, Vector{}, Material{})
		triangles = append(triangles, t)
	}
	return NewMesh(triangles)
}

// testRays generates a deterministic set of rays in general position:
// random origins on a sphere of radius 3 aimed at jittered near-center
// points, so most rays hit the octahedron and none graze edges exactly.
func testRays(n int) []Ray {
	rnd := rand.New(rand.NewSource(42))
	rays := make([]Ray, 0, n)
	for i := 0; i < n; i++ {
		u := rnd.Float64()*2 - 1
		theta := rnd.Float64() * 2 * math.Pi
		s := math.Sqrt(1 - u*u)
		origin := Vector{s * math.Cos(theta), u, s * math.Sin(theta)}.MulScalar(3)
		target := Vector{
			rnd.Float64()*0.4 - 0.2,
			rnd.Float64()*0.4 - 0.2,
			rnd.Float64()*0.4 - 0.2,
		}
		rays = append(rays, Ray{origin, target.Sub(origin).Normalize()})
	}
	return rays
}

func triangleIndex(m *Mesh, tri *Triangle) int {
	for i, t := range m.Triangles {
		if t == tri {
			return i
		}
	}
	return -1
}

type hitResult struct {
	ok    bool
	t     float64
	index int
}

func castAll(m *Mesh, rays []Ray) []hitResult {
	results := make([]hitResult, len(rays))
	for i, r := range rays {
		h := m.Intersect(r)
		if h.Ok() {
			results[i] = hitResult{true, h.T, triangleIndex(m, h.Shape.(*Triangle))}
		}
	}
	return results
}

func assertSameHits(t *testing.T, a, b []hitResult) {
	t.Helper()
	for i := range a {
		if a[i].ok != b[i].ok {
			t.Fatalf("ray %d: hit mismatch: %v vs %v", i, a[i], b[i])
		}
		if !a[i].ok {
			continue
		}
		if math.Abs(a[i].t-b[i].t) > 1e-9 {
			t.Fatalf("ray %d: t mismatch: %v vs %v", i, a[i].t, b[i].t)
		}
		if a[i].index != b[i].index {
			t.Fatalf("ray %d: hit triangle %d, rebuilt mesh hit triangle %d",
				i, a[i].index, b[i].index)
		}
	}
}

// After modifying a few triangle vertices and calling UpdateTriangles,
// intersection results must match a from-scratch rebuild of the mesh.
func TestUpdateTrianglesMatchesRebuild(t *testing.T) {
	rays := testRays(512)

	mesh := octahedronMesh()
	mesh.Compile()
	before := castAll(mesh, rays)
	hits := 0
	for _, h := range before {
		if h.ok {
			hits++
		}
	}
	if hits == 0 {
		t.Fatal("sanity check failed: no ray hits the octahedron")
	}

	// Remember the unmodified triangle pointers.
	original := make([]*Triangle, len(mesh.Triangles))
	copy(original, mesh.Triangles)

	// Locally modify vertices of triangles 0 and 5.
	mesh.Triangles[0].V1 = Vector{1.7, 0.3, -0.2}
	mesh.Triangles[0].V2 = Vector{0.1, 1.4, 0.5}
	mesh.Triangles[5].V3 = Vector{0.2, -0.1, -1.6}
	mesh.UpdateTriangles(0, 5)

	// Unmodified triangles keep their original indices and identities.
	if len(mesh.Triangles) != len(original) {
		t.Fatalf("triangle count changed: %d -> %d", len(original), len(mesh.Triangles))
	}
	for i, tri := range mesh.Triangles {
		if i == 0 || i == 5 {
			continue
		}
		if tri != original[i] {
			t.Fatalf("unmodified triangle %d was replaced", i)
		}
	}

	// Fully rebuilt reference mesh with the same modifications.
	rebuilt := octahedronMesh()
	rebuilt.Triangles[0].V1 = Vector{1.7, 0.3, -0.2}
	rebuilt.Triangles[0].V2 = Vector{0.1, 1.4, 0.5}
	rebuilt.Triangles[5].V3 = Vector{0.2, -0.1, -1.6}
	rebuilt.Compile()

	assertSameHits(t, castAll(mesh, rays), castAll(rebuilt, rays))
}

// A ray must never hit the stale pre-update position of a modified triangle.
func TestUpdateTrianglesNoStaleHits(t *testing.T) {
	mesh := octahedronMesh()
	mesh.Compile()

	// Ray straight at the centroid of triangle 0 = (1,0,0),(0,1,0),(0,0,1).
	centroid := Vector{1.0 / 3, 1.0 / 3, 1.0 / 3}
	origin := centroid.Add(Vector{5, 5, 5})
	ray := Ray{origin, centroid.Sub(origin).Normalize()}
	h := mesh.Intersect(ray)
	if !h.Ok() || triangleIndex(mesh, h.Shape.(*Triangle)) != 0 {
		t.Fatal("expected ray to hit triangle 0 before the update")
	}

	// Move triangle 0 far away; its old position must be empty now.
	mesh.Triangles[0].V1 = Vector{11, 10, 10}
	mesh.Triangles[0].V2 = Vector{10, 11, 10}
	mesh.Triangles[0].V3 = Vector{10, 10, 11}
	mesh.UpdateTriangles(0)

	h = mesh.Intersect(ray)
	if h.Ok() && triangleIndex(mesh, h.Shape.(*Triangle)) == 0 {
		t.Fatal("ray hit the deleted old position of triangle 0")
	}

	rebuilt := octahedronMesh()
	rebuilt.Triangles[0].V1 = Vector{11, 10, 10}
	rebuilt.Triangles[0].V2 = Vector{10, 11, 10}
	rebuilt.Triangles[0].V3 = Vector{10, 10, 11}
	rebuilt.Compile()
	assertSameHits(t, castAll(mesh, testRays(512)), castAll(rebuilt, testRays(512)))
}

// Degenerate triangles are explicitly ignored: excluded from the k-d tree
// and never hit, with results identical to a full rebuild.
func TestDegenerateTrianglesIgnored(t *testing.T) {
	mesh := octahedronMesh()
	mesh.Compile()

	// Collapse triangle 2 onto a line and triangle 6 onto a point.
	mesh.Triangles[2].V3 = mesh.Triangles[2].V1
	mesh.Triangles[6].V1 = Vector{0, 0, 0}
	mesh.Triangles[6].V2 = Vector{0, 0, 0}
	mesh.Triangles[6].V3 = Vector{0, 0, 0}
	if !mesh.Triangles[2].Degenerate() || !mesh.Triangles[6].Degenerate() {
		t.Fatal("expected triangles 2 and 6 to be degenerate")
	}
	if mesh.Triangles[0].Degenerate() {
		t.Fatal("triangle 0 should not be degenerate")
	}
	mesh.UpdateTriangles(2, 6)
	mesh.Compile()

	// The k-d tree must contain exactly the 6 non-degenerate triangles.
	if n := countTreeShapes(mesh.tree.Root); n != 6 {
		t.Fatalf("k-d tree holds %d shapes, want 6 (degenerate triangles ignored)", n)
	}

	// Indices are preserved: degenerate triangles still occupy slots 2 and 6.
	if len(mesh.Triangles) != 8 {
		t.Fatalf("triangle count changed: %d", len(mesh.Triangles))
	}

	// Intersection results match a full rebuild of the same geometry.
	rebuilt := octahedronMesh()
	rebuilt.Triangles[2].V3 = rebuilt.Triangles[2].V1
	rebuilt.Triangles[6].V1 = Vector{0, 0, 0}
	rebuilt.Triangles[6].V2 = Vector{0, 0, 0}
	rebuilt.Triangles[6].V3 = Vector{0, 0, 0}
	rebuilt.Compile()
	assertSameHits(t, castAll(mesh, testRays(512)), castAll(rebuilt, testRays(512)))
}

func countTreeShapes(node *Node) int {
	if node == nil {
		return 0
	}
	return len(node.Shapes) + countTreeShapes(node.Left) + countTreeShapes(node.Right)
}

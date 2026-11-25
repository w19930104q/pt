package pt

import (
	"math"
	"testing"
)

const commitTestEps = 1e-6

// tetrahedron builds a small triangle mesh centered at (cx, cy) with its
// apex at cz+r and its base in the z=cz plane.
func tetrahedron(cx, cy, cz, r float64, material Material) *Mesh {
	h := r * math.Sqrt(3) / 2
	apex := V(cx, cy, cz+r)
	b1 := V(cx+r, cy, cz)
	b2 := V(cx-r/2, cy+h, cz)
	b3 := V(cx-r/2, cy-h, cz)
	n := Vector{}
	triangles := []*Triangle{
		NewTriangle(b1, b3, b2, n, n, n, material),
		NewTriangle(b1, b2, apex, n, n, n, material),
		NewTriangle(b2, b3, apex, n, n, n, material),
		NewTriangle(b3, b1, apex, n, n, n, material),
	}
	return NewMesh(triangles)
}

func downRay(x, y float64) Ray {
	return Ray{Origin: V(x, y, 10), Direction: V(0, 0, -1)}
}

func assertHit(t *testing.T, scene *Scene, r Ray, wantT float64) {
	t.Helper()
	hit := scene.Intersect(r)
	if !hit.Ok() {
		t.Fatalf("expected hit at t=%v, got no hit (ray %+v)", wantT, r)
	}
	if math.Abs(hit.T-wantT) > commitTestEps {
		t.Fatalf("expected hit at t=%v, got t=%v (ray %+v)", wantT, hit.T, r)
	}
}

func assertMiss(t *testing.T, scene *Scene, r Ray) {
	t.Helper()
	if hit := scene.Intersect(r); hit.Ok() {
		t.Fatalf("expected miss, got hit at t=%v (ray %+v)", hit.T, r)
	}
}

// Three objects (a sphere, a cube and a triangle mesh) are grouped under
// one parent node. Staging a rotation of the parent must not change what
// rendering sees until the scene is committed; the commit must update the
// node transform, the mesh bounds and the ray index together.
func TestSceneCommitSnapshot(t *testing.T) {
	material := DiffuseMaterial(White)
	s3 := math.Sqrt(3) / 2
	group := NewGroup(
		NewSphere(V(2, 0, 0.5), 0.5, material),
		NewCube(V(-1.5, s3-0.5, 0), V(-0.5, s3+0.5, 0.5), material),
		tetrahedron(-1, -s3, 0, 0.5, material),
	)
	scene := &Scene{}
	scene.Add(group)
	scene.Commit()

	// initial snapshot: objects at their old positions
	assertHit(t, scene, downRay(2, 0), 9)      // sphere top at z=1
	assertHit(t, scene, downRay(-1, s3), 9.5)  // cube top at z=0.5
	assertHit(t, scene, downRay(-1, -s3), 9.5) // mesh apex at z=0.5
	assertMiss(t, scene, downRay(-2, 0))
	assertMiss(t, scene, downRay(1, -s3))
	assertMiss(t, scene, downRay(1, s3))

	if !group.BoundingBox().Contains(V(2, 0, 0.5)) {
		t.Fatal("group box should contain the old sphere position")
	}
	if group.BoundingBox().Contains(V(-2, 0, 0.5)) {
		t.Fatal("group box should not contain the rotated sphere position yet")
	}

	// stage one rotation of the parent node: 180 degrees about Z
	group.SetMatrix(Rotate(V(0, 0, 1), math.Pi))

	// before the commit, rendering still sees the old snapshot
	assertHit(t, scene, downRay(2, 0), 9)
	assertHit(t, scene, downRay(-1, s3), 9.5)
	assertHit(t, scene, downRay(-1, -s3), 9.5)
	assertMiss(t, scene, downRay(-2, 0))
	assertMiss(t, scene, downRay(1, -s3))
	assertMiss(t, scene, downRay(1, s3))

	scene.Commit()

	// after the commit, rendering sees the new snapshot
	assertMiss(t, scene, downRay(2, 0))
	assertMiss(t, scene, downRay(-1, s3))
	assertMiss(t, scene, downRay(-1, -s3))
	assertHit(t, scene, downRay(-2, 0), 9)
	assertHit(t, scene, downRay(1, -s3), 9.5)
	assertHit(t, scene, downRay(1, s3), 9.5)

	// node bounds and the published ray index cover the new positions
	if !group.BoundingBox().Contains(V(-2, 0, 0.5)) {
		t.Fatal("group box should contain the rotated sphere position")
	}
	if group.BoundingBox().Contains(V(2, 0, 0.5)) {
		t.Fatal("group box should no longer cover the old sphere position")
	}
	tree, _ := scene.tree.Load().(*Tree)
	if tree == nil {
		t.Fatal("scene should have a published snapshot")
	}
	if !tree.Box.Contains(V(-2, 0, 0.5)) {
		t.Fatal("scene ray index box should cover the rotated sphere position")
	}
	if tree.Box.Contains(V(2, 0, 0.5)) {
		t.Fatal("scene ray index box should no longer cover the old sphere position")
	}
}

// Transforming a triangle mesh and committing must rebuild the mesh bounds
// and the ray index in the same commit.
func TestMeshCommitBounds(t *testing.T) {
	material := DiffuseMaterial(White)
	mesh := tetrahedron(0, 0, 0, 0.5, material)
	scene := &Scene{}
	scene.Add(mesh)
	scene.Commit()

	assertHit(t, scene, downRay(0, 0), 9.5)
	assertMiss(t, scene, downRay(5, 0))

	mesh.Transform(Translate(V(5, 0, 0)))
	scene.Commit()

	assertMiss(t, scene, downRay(0, 0))
	assertHit(t, scene, downRay(5, 0), 9.5)

	box := mesh.BoundingBox()
	if box.Min.X < 4 || box.Max.X > 6 {
		t.Fatalf("mesh bounds should move with the mesh, got %+v", box)
	}
	tree, _ := scene.tree.Load().(*Tree)
	if !tree.Box.Contains(V(5, 0, 0.25)) {
		t.Fatal("scene ray index box should cover the moved mesh")
	}
	if tree.Box.Contains(V(0, 0, 0.25)) {
		t.Fatal("scene ray index box should no longer cover the old mesh position")
	}
}

// A staged transform on a TransformedShape takes effect only at commit.
func TestTransformedShapeCommit(t *testing.T) {
	material := DiffuseMaterial(White)
	shape := NewTransformedShape(NewSphere(V(0, 0, 0.5), 0.5, material), Identity())
	scene := &Scene{}
	scene.Add(shape)
	scene.Commit()

	assertHit(t, scene, downRay(0, 0), 9)
	assertMiss(t, scene, downRay(0, 5))

	ts := shape.(*TransformedShape)
	ts.SetMatrix(Translate(V(0, 5, 0)))

	// staged: old snapshot still in effect
	assertHit(t, scene, downRay(0, 0), 9)
	assertMiss(t, scene, downRay(0, 5))

	scene.Commit()

	assertMiss(t, scene, downRay(0, 0))
	assertHit(t, scene, downRay(0, 5), 9)
}

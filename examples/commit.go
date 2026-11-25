package main

// This example demonstrates atomic scene commits. Three objects (a sphere,
// a cube and a triangle mesh) are grouped under one parent node. Rotating
// the parent only stages the change: rendering before scene.Commit still
// sees the old snapshot, while rendering after the commit sees the node
// transform, the mesh bounds and the ray index updated together.
//
// It writes three images:
//   before.png - initial scene
//   staged.png - parent rotation staged but not committed (same as before)
//   after.png  - committed rotation, all three objects moved

import (
	"math"

	. "github.com/fogleman/pt/pt"
)

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

func render(scene *Scene, camera *Camera, path string) {
	sampler := NewSampler(4, 4)
	renderer := NewRenderer(scene, camera, sampler, 480, 360)
	renderer.SamplesPerPixel = 16
	image := renderer.Render()
	if err := SavePNG(path, image); err != nil {
		panic(err)
	}
}

func main() {
	scene := Scene{}
	scene.Color = HexColor(0xFFFFFF)

	// floor and light stay outside the group
	floor := NewPlane(V(0, 0, 0), V(0, 0, 1), DiffuseMaterial(HexColor(0xD8D8D8)))
	scene.Add(floor)
	light := NewSphere(V(0, 0, 6), 0.5, LightMaterial(White, 8))
	scene.Add(light)

	// three objects under one parent node
	s3 := math.Sqrt(3) / 2
	group := NewGroup(
		NewSphere(V(2, 0, 0.5), 0.5, GlossyMaterial(HexColor(0xC93C00), 1.3, Radians(20))),
		NewCube(V(-1.5, s3-0.5, 0), V(-0.5, s3+0.5, 0.5), DiffuseMaterial(HexColor(0x334D5C))),
		tetrahedron(-1, -s3, 0, 0.5, GlossyMaterial(HexColor(0xBFBB11), 1.3, Radians(20))),
	)
	scene.Add(group)

	camera := LookAt(V(4, -4, 3.5), V(0, 0, 0.25), V(0, 0, 1), 40)

	// initial snapshot (Render commits the scene on first use)
	render(&scene, &camera, "before.png")

	// stage one rotation of the parent node: 180 degrees about Z
	group.SetMatrix(Rotate(V(0, 0, 1), math.Pi))

	// not committed yet: rendering still sees the old snapshot
	render(&scene, &camera, "staged.png")

	// commit: node transform, mesh bounds and ray index update together
	scene.Commit()
	render(&scene, &camera, "after.png")
}

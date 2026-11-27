package main

import . "github.com/fogleman/pt/pt"

func main() {
	scene := Scene{}
	scene.Color = Black

	// Floor
	floor := NewCube(V(-10, -10, -1), V(10, 10, 0), DiffuseMaterial(HexColor(0x444444)))
	scene.Add(floor)

	// Three-layer slab stack along Z axis
	// Layer 1: transparent (glass) at z=0.2 .. z=0.5
	glass := ClearMaterial(1.5, 0)
	slab1 := NewCube(V(-1, -1, 0.2), V(1, 1, 0.5), glass)
	scene.Add(slab1)

	// Layer 2: opaque diffuse at z=0.6 .. z=0.9
	red := DiffuseMaterial(HexColor(0xCC3333))
	slab2 := NewCube(V(-1, -1, 0.6), V(1, 1, 0.9), red)
	scene.Add(slab2)

	// Layer 3: transparent (glass) at z=1.0 .. z=1.3
	slab3 := NewCube(V(-1, -1, 1.0), V(1, 1, 1.3), glass)
	scene.Add(slab3)

	// Light above
	scene.Add(NewSphere(V(0, 0, 5), 1, LightMaterial(White, 30)))

	camera := LookAt(V(0, 2, 1.5), V(0, 0, 0.75), V(0, 0, 1), 40)
	sampler := NewSampler(16, 8)
	sampler.SpecularMode = SpecularModeAll
	renderer := NewRenderer(&scene, &camera, sampler, 960, 540)
	renderer.IterativeRender("out%03d.png", 1000)
}

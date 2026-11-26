// Command deterministic renders a small sample scene with a configurable
// number of threads. Because each pixel's random sequence is derived from
// stable coordinates, the output image is identical for any thread count.
//
//	go run ./cmd/deterministic -threads 1 -out pt-64x64-1thread.png
//	go run ./cmd/deterministic -threads 2 -out pt-64x64-2threads.png
//	cmp pt-64x64-1thread.png pt-64x64-2threads.png  # identical
package main

import (
	"flag"

	. "github.com/fogleman/pt/pt"
)

func main() {
	threads := flag.Int("threads", 2, "number of threads (tiles) to render with")
	width := flag.Int("width", 64, "image width")
	height := flag.Int("height", 64, "image height")
	spp := flag.Int("spp", 16, "samples per pixel")
	out := flag.String("out", "out.png", "output path")
	flag.Parse()

	// Cornell-box style scene: colored walls, a glossy gold sphere and a
	// glass sphere, lit by a soft ceiling light.
	scene := Scene{}
	white := DiffuseMaterial(White)
	scene.Add(NewCube(V(-5, -1, -5), V(5, 0, 10), white))                                // floor
	scene.Add(NewCube(V(-5, 5, -5), V(5, 6, 10), white))                                 // ceiling
	scene.Add(NewCube(V(-5, -1, 9), V(5, 6, 10), white))                                 // back wall
	scene.Add(NewCube(V(-5, -1, -5), V(-4, 6, 10), DiffuseMaterial(HexColor(0xC80000)))) // left wall
	scene.Add(NewCube(V(4, -1, -5), V(5, 6, 10), DiffuseMaterial(HexColor(0x0000C8))))   // right wall
	scene.Add(NewSphere(V(-1.4, 0.75, 5), 0.75, GlossyMaterial(HexColor(0xEFC94C), 1.3, Radians(15))))
	scene.Add(NewSphere(V(1.2, 0.75, 3.2), 0.75, ClearMaterial(1.5, 0)))
	scene.Add(NewCube(V(-1, 4.99, 1.5), V(1, 5, 3.5), LightMaterial(White, 8)))

	camera := LookAt(V(0, 2.5, -4), V(0, 1.5, 5), V(0, 1, 0), 50)

	sampler := NewSampler(4, 8)
	renderer := NewRenderer(&scene, &camera, sampler, *width, *height)
	renderer.SamplesPerPixel = *spp
	renderer.NumCPU = *threads
	renderer.Verbose = false
	image := renderer.Render()
	if err := SavePNG(*out, image); err != nil {
		panic(err)
	}
}

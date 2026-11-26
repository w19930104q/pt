package pt

import (
	"testing"
)

// testScene builds a small scene that exercises the random number
// generator heavily: soft shadows over two lights (random light pick and
// random points on the light), glossy and refractive materials and a
// camera aperture.
func testScene() (*Scene, *Camera) {
	scene := Scene{}
	scene.Add(NewCube(V(-10, -1, -10), V(10, 0, 10), DiffuseMaterial(White)))
	scene.Add(NewCube(V(-10, -1, -10), V(-9, 10, 10), DiffuseMaterial(HexColor(0xCC0000))))
	scene.Add(NewCube(V(9, -1, -10), V(10, 10, 10), DiffuseMaterial(HexColor(0x0000CC))))
	scene.Add(NewSphere(V(-1.5, 1, 2), 1, GlossyMaterial(HexColor(0xEFC94C), 1.3, Radians(20))))
	scene.Add(NewSphere(V(1.5, 1, 0), 1, ClearMaterial(1.5, 0)))
	scene.Add(NewSphere(V(-2, 5, 2), 0.75, LightMaterial(White, 10)))
	scene.Add(NewSphere(V(2, 5, -2), 0.75, LightMaterial(White, 10)))
	camera := LookAt(V(0, 3, -8), V(0, 1, 2), V(0, 1, 0), 45)
	camera.SetFocus(V(0, 1, 1), 0.05)
	return &scene, &camera
}

func render(t *testing.T, w, h, spp, ncpu int) *Buffer {
	scene, camera := testScene()
	sampler := NewSampler(4, 4)
	renderer := NewRenderer(scene, camera, sampler, w, h)
	renderer.SamplesPerPixel = spp
	renderer.NumCPU = ncpu
	renderer.Verbose = false
	renderer.Render()
	return renderer.Buffer
}

func buffersEqual(a, b *Buffer) bool {
	if a.W != b.W || a.H != b.H {
		return false
	}
	for i := range a.Pixels {
		if a.Pixels[i] != b.Pixels[i] {
			return false
		}
	}
	return true
}

// The image must not depend on the number of threads (tile scheduling):
// every pixel consumes a random sequence derived from its own stable
// coordinates.
func TestDeterministicThreadCount(t *testing.T) {
	reference := render(t, 64, 64, 4, 1)
	for _, ncpu := range []int{2, 3, 4, 7, 64} {
		buf := render(t, 64, 64, 4, ncpu)
		if !buffersEqual(reference, buf) {
			t.Errorf("render with %d threads differs from single-threaded render", ncpu)
		}
	}
}

// Repeated renders of the same scene must be reproducible.
func TestDeterministicRepeat(t *testing.T) {
	a := render(t, 64, 64, 4, 2)
	b := render(t, 64, 64, 4, 2)
	if !buffersEqual(a, b) {
		t.Error("two identical renders differ")
	}
}

// Iterative rendering must keep accumulating new samples deterministically:
// each iteration uses a fresh sub-sequence, so the buffer keeps changing
// while remaining independent of the thread count.
func TestDeterministicIterations(t *testing.T) {
	renderIter := func(ncpu int) *Buffer {
		scene, camera := testScene()
		sampler := NewSampler(4, 4)
		renderer := NewRenderer(scene, camera, sampler, 64, 64)
		renderer.SamplesPerPixel = 1
		renderer.NumCPU = ncpu
		renderer.Verbose = false
		renderer.IterativeRender("out.png", 3)
		return renderer.Buffer
	}
	a := renderIter(1)
	b := renderIter(2)
	if !buffersEqual(a, b) {
		t.Error("iterative render with 2 threads differs from single-threaded render")
	}
	for _, p := range a.Pixels {
		if p.Samples != 3 {
			t.Errorf("expected 3 samples per pixel, got %d", p.Samples)
			break
		}
	}
}

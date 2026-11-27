package pt

import (
	"math"
	"testing"
)

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

func TestDeterministicThreadCount(t *testing.T) {
	reference := render(t, 64, 64, 4, 1)
	for _, ncpu := range []int{2, 3, 4, 7, 64} {
		buf := render(t, 64, 64, 4, ncpu)
		if !buffersEqual(reference, buf) {
			t.Errorf("render with %d threads differs from single-threaded render", ncpu)
		}
	}
}

func TestDeterministicRepeat(t *testing.T) {
	a := render(t, 64, 64, 4, 2)
	b := render(t, 64, 64, 4, 2)
	if !buffersEqual(a, b) {
		t.Error("two identical renders differ")
	}
}

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

func TestCameraPixelCenter(t *testing.T) {
	camera := LookAt(V(0, 0, -1), V(0, 0, 0), V(0, 1, 0), 90)

	// With fovy=90, m = 1/tan(45°) = 1.0
	// The image plane is at z=0, camera at z=-1

	// For a 2x2 image, pixel (0,0) center should map to:
	// px = ((0+0.5)/2)*2 - 1 = -0.5
	// py = ((0+0.5)/2)*2 - 1 = -0.5
	// Ray dir = (-px, -py, m) normalized = (0.5, 0.5, 1) normalized
	r := camera.CastRay(0, 0, 2, 2, 0.5, 0.5, nil)
	expected := V(0.5, 0.5, 1).Normalize()
	if math.Abs(r.Direction.X-expected.X) > 1e-9 ||
		math.Abs(r.Direction.Y-expected.Y) > 1e-9 ||
		math.Abs(r.Direction.Z-expected.Z) > 1e-9 {
		t.Errorf("pixel (0,0) center: got %v, want %v", r.Direction, expected)
	}

	// Pixel (1,1) center:
	// px = ((1+0.5)/2)*2 - 1 = 0.5
	// py = ((1+0.5)/2)*2 - 1 = 0.5
	r = camera.CastRay(1, 1, 2, 2, 0.5, 0.5, nil)
	expected = V(-0.5, -0.5, 1).Normalize()
	if math.Abs(r.Direction.X-expected.X) > 1e-9 ||
		math.Abs(r.Direction.Y-expected.Y) > 1e-9 ||
		math.Abs(r.Direction.Z-expected.Z) > 1e-9 {
		t.Errorf("pixel (1,1) center: got %v, want %v", r.Direction, expected)
	}
}

func TestAspectRatio32(t *testing.T) {
	camera := LookAt(V(0, 0, -1), V(0, 0, 0), V(0, 1, 0), 90)

	// 3:2 aspect ratio: w=300, h=200
	// aspect = 1.5
	// Pixel (0,0) center: px = (0.5/300)*2-1 = -1 + 1/300 = -0.996666...
	r := camera.CastRay(0, 0, 300, 200, 0.5, 0.5, nil)
	// px ≈ -0.996667, py = (0.5/200)*2-1 = -0.995
	// dir = (px*aspect, py, 1) normalized
	px := -0.9966666666666667
	py := -0.995
	expected := V(-px*1.5, -py, 1).Normalize()
	if math.Abs(r.Direction.X-expected.X) > 1e-6 ||
		math.Abs(r.Direction.Y-expected.Y) > 1e-6 ||
		math.Abs(r.Direction.Z-expected.Z) > 1e-6 {
		t.Errorf("3:2 pixel (0,0): got %v, want %v", r.Direction, expected)
	}
}

func TestAspectRatio169(t *testing.T) {
	camera := LookAt(V(0, 0, -1), V(0, 0, 0), V(0, 1, 0), 90)

	// 16:9 aspect ratio: w=320, h=180
	// aspect = 320/180 ≈ 1.7778
	r := camera.CastRay(0, 0, 320, 180, 0.5, 0.5, nil)
	px := -(1.0 - 0.5/320*2)
	py := -(1.0 - 0.5/180*2)
	aspect := 320.0 / 180.0
	expected := V(-px*aspect, -py, 1).Normalize()
	if math.Abs(r.Direction.X-expected.X) > 1e-6 ||
		math.Abs(r.Direction.Y-expected.Y) > 1e-6 ||
		math.Abs(r.Direction.Z-expected.Z) > 1e-6 {
		t.Errorf("16:9 pixel (0,0): got %v, want %v", r.Direction, expected)
	}
}

func TestConsistentProjectionAcrossResolutions(t *testing.T) {
	// Same scene rendered at two resolutions should have the same
	// pixel center ray for corresponding normalized positions.
	// The center pixel ray should be (0, 0, 1) in camera space.
	camera := LookAt(V(0, 0, -1), V(0, 0, 0), V(0, 1, 0), 90)

	// Center pixel at 3:2 (w=300, h=200): x=149, y=99
	r32 := camera.CastRay(149, 99, 300, 200, 0.5, 0.5, nil)

	// Center pixel at 16:9 (w=320, h=180): x=159, y=89
	r169 := camera.CastRay(159, 89, 320, 180, 0.5, 0.5, nil)

	// Both should point roughly straight ahead (0, 0, 1)
	if math.Abs(r32.Direction.X) > 1e-4 || math.Abs(r32.Direction.Y) > 1e-4 {
		t.Errorf("3:2 center pixel ray not straight: %v", r32.Direction)
	}
	if math.Abs(r169.Direction.X) > 1e-4 || math.Abs(r169.Direction.Y) > 1e-4 {
		t.Errorf("16:9 center pixel ray not straight: %v", r169.Direction)
	}
}

package pt

import (
	"image"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	redColor   = Color{1, 0, 0}
	greenColor = Color{0, 1, 0}
	blueColor  = Color{0, 0, 1}
)

func newTestTriangle(material Material) *Triangle {
	return NewTriangle(
		Vector{0, 0, 0}, Vector{1, 0, 0}, Vector{0, 1, 0},
		Vector{}, Vector{}, Vector{}, material)
}

func newTestMesh(material Material) *Mesh {
	a := newTestTriangle(material)
	b := newTestTriangle(material)
	return NewMesh([]*Triangle{a, b})
}

// Two meshes built from the same material share one *Material per triangle,
// mirroring two objects sharing one material on a page.
func sharedMeshes() (*Mesh, *Mesh, Material) {
	material := DiffuseMaterial(redColor)
	first := newTestMesh(material)
	second := newTestMesh(material)
	return first, second, material
}

// Mesh.Copy shares every triangle's *Material with the original, the same way
// two objects end up sharing a material through Copy or repeated OBJ usemtl.
func copiedMeshes() (*Mesh, *Mesh) {
	original := newTestMesh(DiffuseMaterial(redColor))
	return original, original.Copy()
}

// Replacing the material of one mesh produced through Copy must not write
// through the shared *Material, or the copied mesh changes as well.
func TestSetMaterialOnCopiedMeshKeepsSiblingIndependent(t *testing.T) {
	first, second := copiedMeshes()
	if first.Triangles[0].Material != second.Triangles[0].Material {
		t.Fatal("test setup: copied meshes do not actually share a material pointer")
	}

	first.SetMaterial(DiffuseMaterial(greenColor))

	if got := first.Triangles[0].MaterialAt(Vector{}).Color; got != greenColor {
		t.Fatalf("first mesh = %v, want greenColor", got)
	}
	if got := second.Triangles[0].MaterialAt(Vector{}).Color; got != redColor {
		t.Fatalf("copied sibling changed to %v, want redColor", got)
	}
	if first.Triangles[0].Material == second.Triangles[0].Material {
		t.Fatal("replacement did not give the first mesh its own material")
	}
}

// Replacing one object's material must leave the other object on the shared
// material: no in-place mutation of the shared *Material is allowed.
func TestSetMaterialKeepsSharedMaterialIndependent(t *testing.T) {
	first, second, shared := sharedMeshes()

	newMaterial := DiffuseMaterial(greenColor)
	first.SetMaterial(newMaterial)

	for i, triangle := range first.Triangles {
		if got := triangle.MaterialAt(Vector{}).Color; got != greenColor {
			t.Fatalf("first mesh triangle %d: got color %v, want greenColor", i, got)
		}
	}
	for i, triangle := range second.Triangles {
		if got := triangle.MaterialAt(Vector{}).Color; got != redColor {
			t.Fatalf("second mesh triangle %d changed to %v, want shared redColor", i, got)
		}
	}
	// The shared material value itself must be untouched.
	if shared.Color != redColor {
		t.Fatalf("shared material mutated: got %v, want redColor", shared.Color)
	}
}

// A replacement must swap references rather than write through the old
// pointer, otherwise every shape sharing that pointer changes as well.
func TestTriangleSetMaterialSwapsPointer(t *testing.T) {
	shared := DiffuseMaterial(redColor)
	first := newTestTriangle(shared)
	second := newTestTriangle(shared)
	oldPointer := first.Material

	first.SetMaterial(DiffuseMaterial(blueColor))

	if first.Material == oldPointer {
		t.Fatal("SetMaterial mutated the shared pointer instead of swapping it")
	}
	if first.Material == second.Material {
		t.Fatal("triangles still share a material after replacement")
	}
	if got := second.MaterialAt(Vector{}).Color; got != redColor {
		t.Fatalf("sibling triangle changed to %v, want redColor", got)
	}
}

// The old material stays usable after replacement, so a render batch that
// already captured it can finish with the material it started with.
func TestOldMaterialSurvivesReplacement(t *testing.T) {
	first, second, _ := sharedMeshes()
	old := second.Triangles[0].Material

	first.SetMaterial(DiffuseMaterial(blueColor))

	// A second replacement must not invalidate the material captured earlier.
	second.SetMaterial(DiffuseMaterial(greenColor))
	if got := old.Color; got != redColor {
		t.Fatalf("captured old material changed to %v, want redColor", got)
	}
	if old.Emittance != 0 || old.Index != 1 {
		t.Fatal("captured old material lost its scalar properties")
	}
}

// Clone produces an independent material: mutating the clone never reaches
// the original.
func TestMaterialCloneIsIndependent(t *testing.T) {
	original := DiffuseMaterial(redColor)
	clone := original.Clone()
	clone.Color = greenColor
	clone.Reflectivity = 0.5

	if original.Color != redColor {
		t.Fatalf("original changed through clone: got %v", original.Color)
	}
	if original.Reflectivity != -1 {
		t.Fatalf("original scalar changed through clone: got %v", original.Reflectivity)
	}
	if clone.Color != greenColor || clone.Reflectivity != 0.5 {
		t.Fatal("clone did not keep the mutation")
	}
}

// ReplaceMaterial installs an already owned (e.g. cloned) material wholesale.
func TestMeshReplaceMaterial(t *testing.T) {
	first, second, _ := sharedMeshes()
	clone := first.Triangles[0].Material.Clone()
	clone.Color = blueColor

	first.ReplaceMaterial(clone)

	for _, triangle := range first.Triangles {
		if triangle.Material != clone {
			t.Fatal("ReplaceMaterial did not install the given pointer")
		}
	}
	for _, triangle := range second.Triangles {
		if got := triangle.MaterialAt(Vector{}).Color; got != redColor {
			t.Fatalf("sibling mesh changed to %v, want redColor", got)
		}
	}
}

// The Shape-level helper works for every MaterialHolder, including value
// materials and transformed wrappers, and reports unsupported shapes.
func TestSetMaterialShapeHelper(t *testing.T) {
	sphere := NewSphere(Vector{}, 1, DiffuseMaterial(redColor))
	if !SetMaterial(sphere, DiffuseMaterial(greenColor)) {
		t.Fatal("sphere reported as not supporting SetMaterial")
	}
	if got := sphere.MaterialAt(Vector{}).Color; got != greenColor {
		t.Fatalf("sphere color = %v, want greenColor", got)
	}

	mesh := newTestMesh(DiffuseMaterial(redColor))
	wrapped := NewTransformedShape(mesh, Identity())
	if !SetMaterial(wrapped, DiffuseMaterial(blueColor)) {
		t.Fatal("transformed shape reported as not supporting SetMaterial")
	}
	for _, triangle := range mesh.Triangles {
		if got := triangle.MaterialAt(Vector{}).Color; got != blueColor {
			t.Fatalf("wrapped mesh color = %v, want blueColor", got)
		}
	}
}

// A failed texture load returns the input material untouched instead of a
// half-configured one, so the scene is not polluted by the failure.
func TestLoadMaterialTextureFailureDoesNotPollute(t *testing.T) {
	material := DiffuseMaterial(redColor)
	result, err := LoadMaterialTexture(material, filepath.Join(os.TempDir(), "pt-material-missing.png"))
	if err == nil {
		t.Fatal("expected an error for a missing texture file")
	}
	if result.Color != redColor || result.Texture != nil {
		t.Fatalf("material polluted by failed load: %+v", result)
	}
	if texture := GetTexture(filepath.Join(os.TempDir(), "pt-material-missing.png")); texture != nil {
		t.Fatal("failed texture load was cached")
	}
}

// A successful texture load installs the texture on an independent copy while
// leaving the caller's material value unchanged.
func TestLoadMaterialTextureSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "texture.png")
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range im.Pix {
		im.Pix[i] = 255
	}
	if err := SavePNG(path, im); err != nil {
		t.Fatal(err)
	}
	material := DiffuseMaterial(redColor)
	result, err := LoadMaterialTexture(material, path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Texture == nil {
		t.Fatal("texture not installed on returned material")
	}
	if material.Texture != nil {
		t.Fatal("caller's material was polluted by the successful load")
	}
}

// Material replacement must be race-free while another goroutine reads
// materials, as happens when swapping between render batches.
func TestMaterialSwapDuringConcurrentReads(t *testing.T) {
	mesh := newTestMesh(DiffuseMaterial(redColor))
	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				for _, triangle := range mesh.Triangles {
					_ = triangle.MaterialAt(Vector{}).Color
				}
			}
		}
	}()

	for i := 0; i < 1000; i++ {
		mesh.SetMaterial(DiffuseMaterial(Color{float64(i), 0, 0}))
	}
	close(stop)
	wg.Wait()
}

// A missing bump texture used to panic on a nil *Pow call and abort the whole
// scene load; it must now degrade gracefully while valid materials load.
func TestLoadOBJWithMissingBumpTexture(t *testing.T) {
	dir := t.TempDir()
	mtlPath := filepath.Join(dir, "scene.mtl")
	objPath := filepath.Join(dir, "scene.obj")
	mtl := "newmtl mat\nKd 1 0 0\nmap_bump does-not-exist.png\n"
	obj := "mtllib scene.mtl\nusemtl mat\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"
	if err := os.WriteFile(mtlPath, []byte(mtl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objPath, []byte(obj), 0o644); err != nil {
		t.Fatal(err)
	}

	mesh, err := LoadOBJ(objPath, DiffuseMaterial(White))
	if err != nil {
		t.Fatalf("LoadOBJ failed on missing bump texture: %v", err)
	}
	if len(mesh.Triangles) != 1 {
		t.Fatalf("expected 1 triangle, got %d", len(mesh.Triangles))
	}
	material := mesh.Triangles[0].MaterialAt(Vector{})
	if material.Color != redColor {
		t.Fatalf("material color = %v, want redColor", material.Color)
	}
	if material.NormalTexture != nil {
		t.Fatal("missing bump texture should stay nil rather than pollute the material")
	}
}

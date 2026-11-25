package main

import (
	"fmt"

	. "github.com/fogleman/pt/pt"
)

// This example shows how to replace one object's material without touching
// another object's material.
//
// Two meshes are rendered, each shown twice:
//
//   - sharedMesh is placed twice through the same *Mesh (as cube.go does), so
//     the two views share one material. Replacing its material updates both
//     views, which is the intended "shared material" behavior.
//   - clonedMesh gets a material clone first, so each object owns its
//     material. Replacing one object's material leaves the other untouched.
//
// A failed texture load is also attempted to show that such a failure leaves
// the scene's materials unchanged.
func main() {
	red := DiffuseMaterial(HexColor(0xD7263D))
	green := DiffuseMaterial(HexColor(0x1B9E77))
	blue := DiffuseMaterial(HexColor(0x2C7FB8))

	// --- two objects sharing one material ---------------------------------
	sharedMesh := NewCube(V(-0.5, 0, -0.5), V(0.5, 1, 0.5), red).Mesh()
	scene := Scene{}
	scene.Add(NewTransformedShape(sharedMesh, Translate(V(-1, 0, 0))))
	scene.Add(NewTransformedShape(sharedMesh, Translate(V(1, 0, 0))))
	for _, shape := range lightAndFloor() {
		scene.Add(shape)
	}
	render(scene, "shared-before.png")

	// One call recolors both views because they share the mesh's material.
	sharedMesh.SetMaterial(green)
	render(scene, "shared-after.png")
	fmt.Println("shared: both views are now green")

	// --- two objects with cloned, independent materials -------------------
	source := NewCube(V(-0.5, 0, -0.5), V(0.5, 1, 0.5), red).Mesh()
	objectA := source
	objectB := source.Copy() // Copy shares geometry but can take its own material
	objectB.SetMaterial(*red.Clone())

	scene = Scene{}
	scene.Add(NewTransformedShape(objectA, Translate(V(-1, 0, 0))))
	scene.Add(NewTransformedShape(objectB, Translate(V(1, 0, 0))))
	for _, shape := range lightAndFloor() {
		scene.Add(shape)
	}
	render(scene, "cloned-before.png")

	// Only object A changes: objectB keeps its independent material.
	objectA.SetMaterial(blue)
	render(scene, "cloned-after.png")
	fmt.Printf("cloned: objectA=%v objectB=%v\n",
		objectA.Triangles[0].MaterialAt(V(0, 0, 0)).Color,
		objectB.Triangles[0].MaterialAt(V(0, 0, 0)).Color)

	// --- a failed texture load must not pollute the material --------------
	safe := red
	if _, err := LoadMaterialTexture(safe, "examples/does-not-exist.png"); err != nil {
		fmt.Println("texture load failed, material kept color:", safe.Color)
	}
}

func lightAndFloor() []Shape {
	floor := NewPlane(V(0, 0, 0), V(0, 0, 1), DiffuseMaterial(HexColor(0x404040)))
	light := NewSphere(V(4, 5, 3), 1, LightMaterial(White, 12))
	return []Shape{floor, light}
}

func render(scene Scene, path string) {
	camera := LookAt(V(4, 2.5, 4), V(0, 0.5, 0), V(0, 0, 1), 45)
	sampler := NewSampler(1, 2)
	renderer := NewRenderer(&scene, &camera, sampler, 320, 180)
	renderer.Verbose = false
	im := renderer.Render()
	if err := SavePNG(path, im); err != nil {
		panic(err)
	}
	fmt.Println("wrote", path)
}

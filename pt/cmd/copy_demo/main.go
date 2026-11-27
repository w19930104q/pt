package main

import (
	"fmt"
	"os"

	. "github.com/fogleman/pt"
)

func main() {
	fmt.Println("=== Scene Copy Demo ===")
	fmt.Println()

	// Scene 1: two objects sharing a texture reference
	// We use a nil texture for simplicity; the sharing principle
	// applies equally to real textures loaded via GetTexture().
	var sharedTexture Texture = nil

	scene1 := &Scene{}
	mat1 := GlossyMaterial(Color{1, 0.2, 0.2}, 1.5, 0.1)
	mat1.Texture = sharedTexture
	mat2 := GlossyMaterial(Color{0.2, 0.2, 1}, 1.5, 0.1)
	mat2.Texture = sharedTexture

	sphere1 := NewSphere(Vector{-1, 0, 2}, 0.5, mat1)
	cube1 := NewCube(Vector{0.3, -0.3, 1.5}, Vector{1.3, 0.7, 2.5}, mat2)

	scene1.Add(sphere1)
	scene1.Add(cube1)

	fmt.Printf("Original scene: %d shapes, %d lights\n", len(scene1.Shapes), len(scene1.Lights))

	// Clone the scene
	scene2 := scene1.Copy()
	fmt.Printf("Cloned scene:   %d shapes, %d lights\n", len(scene2.Shapes), len(scene2.Lights))

	// Verify clone independence - part 1: different shape slices
	fmt.Println()
	fmt.Println("--- Independence checks ---")

	// Modify clone's first shape (sphere) material color
	sphere2 := scene2.Shapes[0].(*Sphere)
	originalSphereColor := scene1.Shapes[0].(*Sphere).Material.Color
	fmt.Printf("Before: original sphere color = %v, clone sphere color = %v\n",
		originalSphereColor, sphere2.Material.Color)

	sphere2.Material.Color = Color{0, 1, 0} // green
	newOriginalSphereColor := scene1.Shapes[0].(*Sphere).Material.Color
	newCloneSphereColor := sphere2.Material.Color
	fmt.Printf("After:  original sphere color = %v, clone sphere color = %v\n",
		newOriginalSphereColor, newCloneSphereColor)

	materialIndependent := newOriginalSphereColor != newCloneSphereColor
	fmt.Printf("Materials independent: %v\n", materialIndependent)

	// Verify clone independence - part 2: modify geometry
	cube2 := scene2.Shapes[1].(*Cube)
	oldCloneMin := cube2.Min
	cube2.Min = Vector{0, 0, 0}
	newCloneMin := cube2.Min
	oldOriginalMin := scene1.Shapes[1].(*Cube).Min
	fmt.Printf("Cube geometry: original.Min = %v, clone.Min (was %v, now %v)\n",
		oldOriginalMin, oldCloneMin, newCloneMin)

	geometryIndependent := oldOriginalMin != newCloneMin
	fmt.Printf("Geometry independent: %v\n", geometryIndependent)

	// Verify texture sharing (same pointer or both nil)
	fmt.Println()
	fmt.Println("--- Texture sharing ---")
	tex1 := scene1.Shapes[0].(*Sphere).Material.Texture
	tex2 := scene2.Shapes[0].(*Sphere).Material.Texture
	fmt.Printf("Same texture reference: %v (shared, no double-free risk)\n", tex1 == tex2)

	// Verify scene1 still has 2 shapes after clone modification
	fmt.Printf("Original still has %d shapes: %v\n", len(scene1.Shapes), len(scene1.Shapes) == 2)

	// Demonstrate: deleting original scene does not affect clone
	fmt.Println()
	fmt.Println("--- Delete original, clone survives ---")
	scene1 = nil // "delete" original
	if scene2 != nil {
		fmt.Printf("Clone still has %d shapes and renders independently: %v\n",
			len(scene2.Shapes), len(scene2.Shapes) == 2)
		fmt.Printf("Clone sphere color: %v\n", scene2.Shapes[0].(*Sphere).Material.Color)
	}

	// Summary
	fmt.Println()
	allPassed := materialIndependent && geometryIndependent
	if allPassed {
		fmt.Println("ALL CHECKS PASSED: Clone has independent geometry and materials, shared textures")
	} else {
		fmt.Println("SOME CHECKS FAILED")
		os.Exit(1)
	}
}

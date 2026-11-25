package pt

type Material struct {
	Color          Color
	Texture        Texture
	NormalTexture  Texture
	BumpTexture    Texture
	GlossTexture   Texture
	BumpMultiplier float64
	Emittance      float64
	Index          float64 // refractive index
	Gloss          float64 // reflection cone angle in radians
	Tint           float64 // specular and refractive tinting
	Reflectivity   float64 // metallic reflection
	Transparent    bool
}

func DiffuseMaterial(color Color) Material {
	return Material{color, nil, nil, nil, nil, 1, 0, 1, 0, 0, -1, false}
}

func SpecularMaterial(color Color, index float64) Material {
	return Material{color, nil, nil, nil, nil, 1, 0, index, 0, 0, -1, false}
}

func GlossyMaterial(color Color, index, gloss float64) Material {
	return Material{color, nil, nil, nil, nil, 1, 0, index, gloss, 0, -1, false}
}

func ClearMaterial(index, gloss float64) Material {
	return Material{Black, nil, nil, nil, nil, 1, 0, index, gloss, 0, -1, true}
}

func TransparentMaterial(color Color, index, gloss, tint float64) Material {
	return Material{color, nil, nil, nil, nil, 1, 0, index, gloss, tint, -1, true}
}

func MetallicMaterial(color Color, gloss, tint float64) Material {
	return Material{color, nil, nil, nil, nil, 1, 0, 1, gloss, tint, 1, false}
}

func LightMaterial(color Color, emittance float64) Material {
	return Material{color, nil, nil, nil, nil, 1, emittance, 1, 0, 0, -1, false}
}

// Clone returns an independent copy of the material. Mutating the returned
// material (or assigning it with SetMaterial) never affects shapes that
// still reference the original.
func (m Material) Clone() *Material {
	clone := m
	return &clone
}

// SetMaterial installs the material on the shape without touching any material
// that other shapes may share with it. The value is copied first, so callers
// cannot mutate the shape's material through a pointer they still hold.
// Shapes that cannot carry an independently replaceable material return false.
func SetMaterial(shape Shape, material Material) bool {
	holder, ok := shape.(MaterialHolder)
	if !ok {
		return false
	}
	holder.SetMaterial(material)
	return true
}

// LoadMaterialTexture loads the texture at path and, on success, installs it on
// the material through a cloned copy. A failed load returns the error together
// with the untouched material, so a failed load never partially pollutes a
// material that may already be attached to a rendered shape.
func LoadMaterialTexture(material Material, path string) (Material, error) {
	texture, err := LoadTexture(path)
	if err != nil {
		return material, err
	}
	material.Texture = texture
	return material, nil
}

func MaterialAt(shape Shape, point Vector) Material {
	material := shape.MaterialAt(point)
	uv := shape.UV(point)
	if material.Texture != nil {
		material.Color = material.Texture.Sample(uv.X, uv.Y)
	}
	if material.GlossTexture != nil {
		c := material.GlossTexture.Sample(uv.X, uv.Y)
		material.Gloss = (c.R + c.G + c.B) / 3
	}
	return material
}

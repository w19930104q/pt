package pt

import "math"

type Mesh struct {
	Triangles []*Triangle
	box       *Box
	tree      *Tree
}

func NewMesh(triangles []*Triangle) *Mesh {
	return &Mesh{triangles, nil, nil}
}

func (m *Mesh) dirty() {
	m.box = nil
	m.tree = nil
}

func (m *Mesh) Copy() *Mesh {
	triangles := make([]*Triangle, len(m.Triangles))
	for i, t := range m.Triangles {
		a := *t
		triangles[i] = &a
	}
	return NewMesh(triangles)
}

func (m *Mesh) Compile() {
	if m.tree == nil {
		// Degenerate (zero-area) triangles can never be hit; exclude them
		// explicitly so they don't inflate the k-d tree bounds.
		shapes := make([]Shape, 0, len(m.Triangles))
		for _, triangle := range m.Triangles {
			if triangle.Degenerate() {
				continue
			}
			shapes = append(shapes, triangle)
		}
		m.tree = NewTree(shapes)
	}
}

// UpdateTriangles notifies the mesh that the triangles at the given indices
// had their vertices (or the triangles themselves) modified. The k-d tree is
// invalidated and lazily rebuilt on the next Compile or Intersect, so
// intersection results are identical to a full rebuild. The Triangles slice
// is left untouched: unmodified triangles keep their original indices.
func (m *Mesh) UpdateTriangles(indices ...int) {
	for _, i := range indices {
		if i < 0 || i >= len(m.Triangles) {
			panic("pt: UpdateTriangles index out of range")
		}
	}
	m.dirty()
}

func (a *Mesh) Add(b *Mesh) {
	a.Triangles = append(a.Triangles, b.Triangles...)
	a.dirty()
}

func (m *Mesh) BoundingBox() Box {
	if m.box == nil {
		min := m.Triangles[0].V1
		max := m.Triangles[0].V1
		for _, t := range m.Triangles {
			min = min.Min(t.V1).Min(t.V2).Min(t.V3)
			max = max.Max(t.V1).Max(t.V2).Max(t.V3)
		}
		m.box = &Box{min, max}
	}
	return *m.box
}

func (m *Mesh) Intersect(r Ray) Hit {
	m.Compile() // no-op unless the tree was invalidated
	return m.tree.Intersect(r)
}

func (m *Mesh) UV(p Vector) Vector {
	return Vector{} // not implemented
}

func (m *Mesh) MaterialAt(p Vector) Material {
	return Material{} // not implemented
}

func (m *Mesh) NormalAt(p Vector) Vector {
	return Vector{} // not implemented
}

func smoothNormalsThreshold(normal Vector, normals []Vector, threshold float64) Vector {
	result := Vector{}
	for _, x := range normals {
		if x.Dot(normal) >= threshold {
			result = result.Add(x)
		}
	}
	return result.Normalize()
}

func (m *Mesh) SmoothNormalsThreshold(radians float64) {
	threshold := math.Cos(radians)
	lookup := make(map[Vector][]Vector)
	for _, t := range m.Triangles {
		lookup[t.V1] = append(lookup[t.V1], t.N1)
		lookup[t.V2] = append(lookup[t.V2], t.N2)
		lookup[t.V3] = append(lookup[t.V3], t.N3)
	}
	for _, t := range m.Triangles {
		t.N1 = smoothNormalsThreshold(t.N1, lookup[t.V1], threshold)
		t.N2 = smoothNormalsThreshold(t.N2, lookup[t.V2], threshold)
		t.N3 = smoothNormalsThreshold(t.N3, lookup[t.V3], threshold)
	}
}

func (m *Mesh) SmoothNormals() {
	lookup := make(map[Vector]Vector)
	for _, t := range m.Triangles {
		lookup[t.V1] = lookup[t.V1].Add(t.N1)
		lookup[t.V2] = lookup[t.V2].Add(t.N2)
		lookup[t.V3] = lookup[t.V3].Add(t.N3)
	}
	for k, v := range lookup {
		lookup[k] = v.Normalize()
	}
	for _, t := range m.Triangles {
		t.N1 = lookup[t.V1]
		t.N2 = lookup[t.V2]
		t.N3 = lookup[t.V3]
	}
}

func (m *Mesh) UnitCube() {
	m.FitInside(Box{Vector{}, Vector{1, 1, 1}}, Vector{})
	m.MoveTo(Vector{}, Vector{0.5, 0.5, 0.5})
}

func (m *Mesh) MoveTo(position, anchor Vector) {
	matrix := Translate(position.Sub(m.BoundingBox().Anchor(anchor)))
	m.Transform(matrix)
}

func (m *Mesh) FitInside(box Box, anchor Vector) {
	scale := box.Size().Div(m.BoundingBox().Size()).MinComponent()
	extra := box.Size().Sub(m.BoundingBox().Size().MulScalar(scale))
	matrix := Identity()
	matrix = matrix.Translate(m.BoundingBox().Min.Negate())
	matrix = matrix.Scale(Vector{scale, scale, scale})
	matrix = matrix.Translate(box.Min.Add(extra.Mul(anchor)))
	m.Transform(matrix)
}

func (m *Mesh) Transform(matrix Matrix) {
	for _, t := range m.Triangles {
		t.V1 = matrix.MulPosition(t.V1)
		t.V2 = matrix.MulPosition(t.V2)
		t.V3 = matrix.MulPosition(t.V3)
		t.N1 = matrix.MulDirection(t.N1)
		t.N2 = matrix.MulDirection(t.N2)
		t.N3 = matrix.MulDirection(t.N3)
	}
	m.dirty()
}

func (m *Mesh) SetMaterial(material Material) {
	for _, t := range m.Triangles {
		t.Material = &material
	}
}

func (m *Mesh) SaveSTL(path string) error {
	return SaveSTL(path, m)
}

package pt

import "sync/atomic"

type Scene struct {
	Color        Color
	Texture      Texture
	TextureAngle float64
	Shapes       []Shape
	Lights       []Shape
	tree         *Tree
	rays         uint64
}

func (s *Scene) Copy() *Scene {
	c := &Scene{}
	c.Color = s.Color
	c.Texture = s.Texture
	c.TextureAngle = s.TextureAngle
	c.Shapes = make([]Shape, len(s.Shapes))
	for i, shape := range s.Shapes {
		c.Shapes[i] = shape.CopyShape()
	}
	c.Lights = make([]Shape, len(s.Lights))
	for i, light := range s.Lights {
		c.Lights[i] = light.CopyShape()
	}
	return c
}

func (s *Scene) Compile() {
	for _, shape := range s.Shapes {
		shape.Compile()
	}
	if s.tree == nil {
		s.tree = NewTree(s.Shapes)
	}
}

func (s *Scene) Add(shape Shape) {
	s.Shapes = append(s.Shapes, shape)
	if shape.MaterialAt(Vector{}).Emittance > 0 {
		s.Lights = append(s.Lights, shape)
	}
}

func (s *Scene) RayCount() uint64 {
	return atomic.LoadUint64(&s.rays)
}

func (s *Scene) Intersect(r Ray) Hit {
	atomic.AddUint64(&s.rays, 1)
	return s.tree.Intersect(r)
}

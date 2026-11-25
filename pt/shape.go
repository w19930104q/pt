package pt

type Shape interface {
	Compile()
	BoundingBox() Box
	Intersect(Ray) Hit
	UV(Vector) Vector
	NormalAt(Vector) Vector
	MaterialAt(Vector) Material
}

type TransformedShape struct {
	Shape
	Matrix     Matrix
	Inverse    Matrix
	pending    Matrix
	hasPending bool
}

func NewTransformedShape(s Shape, m Matrix) Shape {
	return &TransformedShape{Shape: s, Matrix: m, Inverse: m.Inverse()}
}

// SetMatrix stages a new transform for the shape. The change is not visible
// to rendering until the next scene Commit, which applies the transform and
// rebuilds the dependent bounding boxes and ray index in one step.
func (s *TransformedShape) SetMatrix(m Matrix) {
	s.pending = m
	s.hasPending = true
}

// Transform stages an additional transform that is applied after the
// current (or previously staged) one.
func (s *TransformedShape) Transform(m Matrix) {
	s.SetMatrix(m.Mul(s.stagedMatrix()))
}

func (s *TransformedShape) stagedMatrix() Matrix {
	if s.hasPending {
		return s.pending
	}
	return s.Matrix
}

// Compile applies any staged transform and compiles the wrapped shape.
func (s *TransformedShape) Compile() {
	if s.hasPending {
		s.Matrix = s.pending
		s.Inverse = s.pending.Inverse()
		s.pending = Matrix{}
		s.hasPending = false
	}
	s.Shape.Compile()
}

func (s *TransformedShape) BoundingBox() Box {
	return s.Matrix.MulBox(s.Shape.BoundingBox())
}

func (s *TransformedShape) Intersect(r Ray) Hit {
	shapeRay := s.Inverse.MulRay(r)
	hit := s.Shape.Intersect(shapeRay)
	return transformHit(hit, shapeRay, r, s.Matrix, s.Inverse)
}

// transformHit converts a hit recorded in a transformed shape's local space
// back into world space.
func transformHit(hit Hit, shapeRay, worldRay Ray, matrix, inverse Matrix) Hit {
	if !hit.Ok() {
		return hit
	}
	shape := hit.Shape
	shapePosition := shapeRay.Position(hit.T)
	shapeNormal := shape.NormalAt(shapePosition)
	position := matrix.MulPosition(shapePosition)
	normal := inverse.Transpose().MulDirection(shapeNormal)
	material := MaterialAt(shape, shapePosition)
	inside := false
	if shapeNormal.Dot(shapeRay.Direction) > 0 {
		normal = normal.Negate()
		inside = true
	}
	ray := Ray{position, normal}
	info := HitInfo{shape, position, normal, ray, material, inside}
	hit.T = position.Sub(worldRay.Origin).Length()
	hit.HitInfo = &info
	return hit
}

package pt

// Group is a parent node in the scene graph: it applies a shared transform
// to its child shapes. Moving a node means staging a new transform with
// SetMatrix or Transform; the move becomes visible to rendering only at the
// next scene Commit, which applies the staged transform and rebuilds the
// group's bounding box together with the scene's ray index.
//
// Note that lights inside a group are not registered as scene lights by
// Scene.Add; add light sources to the scene directly.
type Group struct {
	Shapes     []Shape
	Matrix     Matrix
	Inverse    Matrix
	pending    Matrix
	hasPending bool
	box        *Box
}

func NewGroup(shapes ...Shape) *Group {
	return &Group{Shapes: shapes, Matrix: Identity(), Inverse: Identity()}
}

func (g *Group) Add(shapes ...Shape) {
	g.Shapes = append(g.Shapes, shapes...)
	g.box = nil
}

// SetMatrix stages a new transform for the node. The change is staged only:
// it takes effect at the next scene Commit.
func (g *Group) SetMatrix(m Matrix) {
	g.pending = m
	g.hasPending = true
}

// Transform stages an additional transform that is applied after the
// current (or previously staged) one.
func (g *Group) Transform(m Matrix) {
	g.SetMatrix(m.Mul(g.stagedMatrix()))
}

func (g *Group) stagedMatrix() Matrix {
	if g.hasPending {
		return g.pending
	}
	return g.Matrix
}

// Compile applies any staged transform, invalidates the cached bounding
// box and compiles the children. Called by Scene.Commit.
func (g *Group) Compile() {
	if g.hasPending {
		g.Matrix = g.pending
		g.Inverse = g.pending.Inverse()
		g.pending = Matrix{}
		g.hasPending = false
		g.box = nil
	}
	for _, shape := range g.Shapes {
		shape.Compile()
	}
}

func (g *Group) BoundingBox() Box {
	if g.box == nil {
		box := g.Matrix.MulBox(BoxForShapes(g.Shapes))
		g.box = &box
	}
	return *g.box
}

func (g *Group) Intersect(r Ray) Hit {
	shapeRay := g.Inverse.MulRay(r)
	hit := NoHit
	for _, shape := range g.Shapes {
		if h := shape.Intersect(shapeRay); h.T < hit.T {
			hit = h
		}
	}
	return transformHit(hit, shapeRay, r, g.Matrix, g.Inverse)
}

func (g *Group) UV(p Vector) Vector {
	return Vector{} // not implemented
}

func (g *Group) NormalAt(p Vector) Vector {
	return Vector{} // not implemented
}

func (g *Group) MaterialAt(p Vector) Material {
	return Material{} // not implemented
}

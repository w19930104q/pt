package pt

import (
	"sync"
	"sync/atomic"
)

// Scene holds the shapes, lights and environment for rendering.
//
// Rendering never reads Shapes directly; it reads an immutable snapshot
// (the k-d tree "ray index" plus all compiled bounding boxes) that is
// published atomically by Commit. Mutating the scene — moving a node with
// Group.SetMatrix / TransformedShape.SetMatrix, transforming a mesh, adding
// shapes — only stages changes. Renders started before Commit keep seeing
// the old snapshot; renders started after Commit see the new one. Node
// transforms, triangle mesh bounds and the ray index are therefore always
// updated together, in a single scene commit.
//
// Commit must not run concurrently with a render in progress; it is the
// synchronization point between frames.
type Scene struct {
	Color        Color
	Texture      Texture
	TextureAngle float64
	Shapes       []Shape
	Lights       []Shape
	mu           sync.Mutex   // serializes commits
	tree         atomic.Value // published snapshot, stores *Tree
	rays         uint64
}

// Compile publishes the scene if it has never been committed. It is called
// automatically by the renderer and normally does not need to be called
// directly. Use Commit to publish scene changes.
func (s *Scene) Compile() {
	if s.tree.Load() == nil {
		s.Commit()
	}
}

// Commit applies all staged changes and atomically publishes a new
// rendering snapshot: pending node transforms are applied, triangle mesh
// bounding boxes and trees are rebuilt, and a new k-d tree ray index is
// constructed over the updated shapes. The new snapshot replaces the old
// one in a single atomic swap, so a render always sees one consistent
// snapshot — never a mix of old bounding boxes and new geometry.
func (s *Scene) Commit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, shape := range s.Shapes {
		shape.Compile()
	}
	s.tree.Store(NewTree(s.Shapes))
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
	tree, _ := s.tree.Load().(*Tree)
	if tree == nil {
		s.Compile()
		tree, _ = s.tree.Load().(*Tree)
	}
	return tree.Intersect(r)
}

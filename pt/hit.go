package pt

import "sort"

type Hit struct {
	Shape   Shape
	T       float64
	HitInfo *HitInfo
}

type HitInfo struct {
	Shape    Shape
	Position Vector
	Normal   Vector
	Ray      Ray
	Material Material
	Inside   bool
}

var NoHit = Hit{nil, INF, nil}

func (hit *Hit) Ok() bool {
	return hit.T < INF
}

func (hit *Hit) Info(r Ray) HitInfo {
	if hit.HitInfo != nil {
		return *hit.HitInfo
	}
	shape := hit.Shape
	position := r.Position(hit.T)
	normal := shape.NormalAt(position)
	material := MaterialAt(shape, position)
	inside := false
	if normal.Dot(r.Direction) > 0 {
		normal = normal.Negate()
		inside = true
		switch shape.(type) {
		case *Volume, *SDFShape, *SphericalHarmonic:
			inside = false
		}
	}
	ray := Ray{position, normal}
	return HitInfo{shape, position, normal, ray, material, inside}
}

type Hits []Hit

func (hits Hits) Len() int           { return len(hits) }
func (hits Hits) Less(i, j int) bool { return hits[i].T < hits[j].T }
func (hits Hits) Swap(i, j int)      { hits[i], hits[j] = hits[j], hits[i] }

func (hits Hits) Sort() {
	sort.Stable(hits)
}

func (hits Hits) Closest() Hit {
	if len(hits) == 0 {
		return NoHit
	}
	return hits[0]
}

func (hits Hits) FirstOpaque(r Ray) Hit {
	for _, h := range hits {
		position := r.Position(h.T)
		material := MaterialAt(h.Shape, position)
		if !material.Transparent {
			return h
		}
	}
	return NoHit
}

package deepcopy

type Visited = map[any]any

func NewVisited() Visited {
	return make(Visited)
}

func CopyPtr[T any](v Visited, p *T) *T {
	if p == nil {
		return nil
	}

	if out, ok := v[p]; ok {
		return out.(*T)
	}

	if copier, ok := any(p).(interface{ deepcopy(Visited) *T }); ok {
		return copier.deepcopy(v)
	}

	out := new(T)
	*out = *p
	return out
}

func CopyDoublePtr[T any](v Visited, p **T) **T {
	if p == nil {
		return nil
	}
	out := new(*T)
	if *p != nil {
		*out = CopyPtr(v, *p)
	}
	return out
}

func CopySlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	copy(out, s)
	return out
}

func CopySlicePtr[T any](v Visited, s []*T) []*T {
	if s == nil {
		return nil
	}
	out := make([]*T, len(s))
	for i, p := range s {
		if p != nil {
			out[i] = CopyPtr(v, p)
		}
	}
	return out
}

func CopySliceSlice[T any](v Visited, s [][]T) [][]T {
	if s == nil {
		return nil
	}
	out := make([][]T, len(s))
	for i, elem := range s {
		if elem != nil {
			out[i] = make([]T, len(elem))
			for j, val := range elem {
				if copier, ok := any(val).(interface{ deepcopy(Visited) *T }); ok {
					out[i][j] = *copier.deepcopy(v)
				} else {
					out[i][j] = val
				}
			}
		}
	}
	return out
}

func CopySliceMap[K comparable, V any](v Visited, s []map[K]V) []map[K]V {
	if s == nil {
		return nil
	}
	out := make([]map[K]V, len(s))
	for i, m := range s {
		if m != nil {
			out[i] = make(map[K]V, len(m))
			for k, val := range m {
				if copier, ok := any(val).(interface{ deepcopy(Visited) *V }); ok {
					out[i][k] = *copier.deepcopy(v)
				} else {
					out[i][k] = val
				}
			}
		}
	}
	return out
}

func CopyMap[K comparable, V any](v Visited, m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, val := range m {
		if copier, ok := any(val).(interface{ deepcopy(Visited) *V }); ok {
			out[k] = *copier.deepcopy(v)
		} else {
			out[k] = val
		}
	}
	return out
}

func CopyMapPtr[K comparable, V any](v Visited, m map[K]*V) map[K]*V {
	if m == nil {
		return nil
	}
	out := make(map[K]*V, len(m))
	for k, p := range m {
		if p != nil {
			out[k] = CopyPtr(v, p)
		}
	}
	return out
}

func CopyMapSlice[K comparable, V any](v Visited, m map[K][]V) map[K][]V {
	if m == nil {
		return nil
	}
	out := make(map[K][]V, len(m))
	for k, s := range m {
		if s != nil {
			out[k] = make([]V, len(s))
			for i, val := range s {
				if copier, ok := any(val).(interface{ deepcopy(Visited) *V }); ok {
					out[k][i] = *copier.deepcopy(v)
				} else {
					out[k][i] = val
				}
			}
		} else {
			out[k] = nil
		}
	}
	return out
}

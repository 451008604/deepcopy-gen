package deepcopy

func CopyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}

	if copier, ok := any(p).(interface{ DeepCopy() *T }); ok {
		return copier.DeepCopy()
	}
	out := new(T)
	*out = *p
	return out
}

func CopyDoublePtr[T any](p **T) **T {
	if p == nil {
		return nil
	}
	out := new(*T)
	if *p != nil {
		if copier, ok := any(*p).(interface{ DeepCopy() *T }); ok {
			*out = copier.DeepCopy()
		} else {
			*out = new(T)
			**out = **p
		}
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

func CopySlicePtr[T any](s []*T) []*T {
	if s == nil {
		return nil
	}
	out := make([]*T, len(s))
	for i, v := range s {
		if v != nil {
			if copier, ok := any(v).(interface{ DeepCopy() *T }); ok {
				out[i] = copier.DeepCopy()
			} else {
				out[i] = new(T)
				*out[i] = *v
			}
		}
	}
	return out
}

func CopySliceSlice[T any](s [][]T) [][]T {
	if s == nil {
		return nil
	}
	out := make([][]T, len(s))
	for i, v := range s {
		if v != nil {
			out[i] = make([]T, len(v))
			for j, elem := range v {
				if copier, ok := any(elem).(interface{ DeepCopy() *T }); ok {
					out[i][j] = *copier.DeepCopy()
				} else {
					out[i][j] = elem
				}
			}
		}
	}
	return out
}

func CopySliceMap[K comparable, V any](s []map[K]V) []map[K]V {
	if s == nil {
		return nil
	}
	out := make([]map[K]V, len(s))
	for i, v := range s {
		if v != nil {
			out[i] = make(map[K]V, len(v))
			for mk, mv := range v {
				if copier, ok := any(mv).(interface{ DeepCopy() *V }); ok {
					out[i][mk] = *copier.DeepCopy()
				} else {
					out[i][mk] = mv
				}
			}
		}
	}
	return out
}

func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		if copier, ok := any(v).(interface{ DeepCopy() *V }); ok {
			out[k] = *copier.DeepCopy()
		} else {
			out[k] = v
		}
	}
	return out
}

func CopyMapPtr[K comparable, V any](m map[K]*V) map[K]*V {
	if m == nil {
		return nil
	}
	out := make(map[K]*V, len(m))
	for k, v := range m {
		if v != nil {
			if copier, ok := any(v).(interface{ DeepCopy() *V }); ok {
				out[k] = copier.DeepCopy()
			} else {
				out[k] = new(V)
				*out[k] = *v
			}
		} else {
			out[k] = nil
		}
	}
	return out
}

func CopyMapSlice[K comparable, V any](m map[K][]V) map[K][]V {
	if m == nil {
		return nil
	}
	out := make(map[K][]V, len(m))
	for k, v := range m {
		if v != nil {
			out[k] = make([]V, len(v))
			for i, elem := range v {
				if copier, ok := any(elem).(interface{ DeepCopy() *V }); ok {
					out[k][i] = *copier.DeepCopy()
				} else {
					out[k][i] = elem
				}
			}
		} else {
			out[k] = nil
		}
	}
	return out
}

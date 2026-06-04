package deepcopy

import "reflect"

// CopyPtr returns a deep copy of a pointer. Returns nil if p is nil.
func CopyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	out := new(T)
	*out = *p
	return out
}

// CopyDoublePtr returns a deep copy of a double pointer. Returns nil if p is nil.
func CopyDoublePtr[T any](p **T) **T {
	if p == nil {
		return nil
	}
	out := new(*T)
	if *p != nil {
		*out = new(T)
		**out = **p
	}
	return out
}

// CopySlice returns a deep copy of a slice. Returns nil if s is nil.
func CopySlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	copy(out, s)
	return out
}

// CopySlicePtr returns a deep copy of a slice of pointers.
func CopySlicePtr[T any](s []*T) []*T {
	if s == nil {
		return nil
	}
	out := make([]*T, len(s))
	for i, v := range s {
		if v != nil {
			out[i] = new(T)
			*out[i] = *v
		}
	}
	return out
}

// CopySliceSlice returns a deep copy of a slice of slices.
func CopySliceSlice[T any](s [][]T) [][]T {
	if s == nil {
		return nil
	}
	out := make([][]T, len(s))
	for i, v := range s {
		if v != nil {
			out[i] = make([]T, len(v))
			copy(out[i], v)
		}
	}
	return out
}

// CopySliceMap returns a deep copy of a slice of maps.
func CopySliceMap[K comparable, V any](s []map[K]V) []map[K]V {
	if s == nil {
		return nil
	}
	out := make([]map[K]V, len(s))
	for i, v := range s {
		if v != nil {
			out[i] = make(map[K]V, len(v))
			for mk, mv := range v {
				out[i][mk] = mv
			}
		}
	}
	return out
}

// CopyMap returns a deep copy of a map with value-type values.
func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// CopyMapPtr returns a deep copy of a map with pointer values.
func CopyMapPtr[K comparable, V any](m map[K]*V) map[K]*V {
	if m == nil {
		return nil
	}
	out := make(map[K]*V, len(m))
	for k, v := range m {
		if v != nil {
			out[k] = new(V)
			*out[k] = *v
		} else {
			out[k] = nil
		}
	}
	return out
}

// CopyMapSlice returns a deep copy of a map with slice values.
func CopyMapSlice[K comparable, V any](m map[K][]V) map[K][]V {
	if m == nil {
		return nil
	}
	out := make(map[K][]V, len(m))
	for k, v := range m {
		if v != nil {
			out[k] = make([]V, len(v))
			copy(out[k], v)
		} else {
			out[k] = nil
		}
	}
	return out
}

// DeepCopyAny returns a recursive deep copy of any value using reflection.
// Handles nil, basic types, pointers, slices, maps, and structs.
func DeepCopyAny(v any) any {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	return deepCopyReflect(val).Interface()
}

func deepCopyReflect(val reflect.Value) reflect.Value {
	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		out := reflect.New(val.Type().Elem())
		out.Elem().Set(deepCopyReflect(val.Elem()))
		return out

	case reflect.Slice:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		out := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			out.Index(i).Set(deepCopyReflect(val.Index(i)))
		}
		return out

	case reflect.Map:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		out := reflect.MakeMapWithSize(val.Type(), val.Len())
		iter := val.MapRange()
		for iter.Next() {
			out.SetMapIndex(
				deepCopyReflect(iter.Key()),
				deepCopyReflect(iter.Value()),
			)
		}
		return out

	case reflect.Struct:
		out := reflect.New(val.Type()).Elem()
		for i := 0; i < val.NumField(); i++ {
			if !val.Type().Field(i).IsExported() {
				continue
			}
			out.Field(i).Set(deepCopyReflect(val.Field(i)))
		}
		return out

	default:
		return val
	}
}

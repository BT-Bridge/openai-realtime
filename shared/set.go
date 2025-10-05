package shared

import (
	"fmt"
	"iter"
	"strings"
)

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewPtrSetCap[T comparable](cap int) *Set[T] {
	return &Set[T]{m: make(map[T]struct{}, cap)}
}

func NewPtrSet[T comparable](elements ...T) *Set[T] {
	set := Set[T]{m: make(map[T]struct{})}
	for _, element := range elements {
		set.m[element] = struct{}{}
	}
	return &set
}

func NewSetCap[T comparable](cap int) Set[T] {
	return Set[T]{m: make(map[T]struct{}, cap)}
}

func NewSet[T comparable](elements ...T) Set[T] {
	set := Set[T]{m: make(map[T]struct{})}
	for _, element := range elements {
		set.m[element] = struct{}{}
	}
	return set
}

func (s Set[T]) Contains(element T) bool {
	_, ok := s.m[element]
	return ok
}

func (s Set[T]) Add(element T) (existed bool) {
	_, existed = s.m[element]
	s.m[element] = struct{}{}
	return
}

func (s Set[T]) Remove(element T) (existed bool) {
	_, existed = s.m[element]
	delete(s.m, element)
	return
}

func (s Set[T]) ToSlice() []T {
	elements := make([]T, 0, len(s.m)+1)
	for element := range s.m {
		elements = append(elements, element)
	}
	return elements
}

func (s Set[T]) Size() int {
	return len(s.m)
}

func (s Set[T]) String() string {
	builder := &strings.Builder{}
	_, _ = builder.WriteString("{ ")
	for i, element := range s.ToSlice() {
		if i > 0 {
			_, _ = builder.WriteString(", ")
		}
		_, _ = fmt.Fprintf(builder, "%v", element)
	}
	_, _ = builder.WriteString(" }")
	return builder.String()
}

func (s Set[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for element := range s.m {
			if !yield(element) {
				return
			}
		}
	}
}

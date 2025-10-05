package shared

import (
	"reflect"
	"sort"
	"testing"
)

func TestNewSet(t *testing.T) {
	t.Run("EmptySet", func(t *testing.T) {
		set := NewSet[int]()
		if len(set.m) != 0 {
			t.Errorf("Expected empty set, got %d elements", len(set.m))
		}
	})

	t.Run("SetWithIntegers", func(t *testing.T) {
		set := NewSet(1, 2, 3, 2)
		if len(set.m) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(set.m))
		}
		for _, v := range []int{1, 2, 3} {
			if !set.Contains(v) {
				t.Errorf("Expected element %d to be in set", v)
			}
		}
	})

	t.Run("SetWithStrings", func(t *testing.T) {
		set := NewSet("a", "b", "a")
		if len(set.m) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(set.m))
		}
		for _, v := range []string{"a", "b"} {
			if !set.Contains(v) {
				t.Errorf("Expected element %s to be in set", v)
			}
		}
	})
}

func TestContains(t *testing.T) {
	set := NewSet(1, 2, 3)

	t.Run("ExistingElement", func(t *testing.T) {
		if !set.Contains(1) {
			t.Error("Expected Contains(1) to return true")
		}
	})

	t.Run("NonExistingElement", func(t *testing.T) {
		if set.Contains(4) {
			t.Error("Expected Contains(4) to return false")
		}
	})

	t.Run("EmptySet", func(t *testing.T) {
		emptySet := NewSet[int]()
		if emptySet.Contains(1) {
			t.Error("Expected Contains(1) to return false on empty set")
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("AddNewElement", func(t *testing.T) {
		set := NewSet(1, 2)
		existed := set.Add(3)
		if existed {
			t.Error("Expected Add(3) to return false for new element")
		}
		if !set.Contains(3) {
			t.Error("Expected element 3 to be in set after Add")
		}
		if len(set.m) != 3 {
			t.Errorf("Expected set size 3, got %d", len(set.m))
		}
	})

	t.Run("AddExistingElement", func(t *testing.T) {
		set := NewSet(1, 2)
		existed := set.Add(1)
		if !existed {
			t.Error("Expected Add(1) to return true for existing element")
		}
		if len(set.m) != 2 {
			t.Errorf("Expected set size 2, got %d", len(set.m))
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("RemoveExistingElement", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		existed := set.Remove(2)
		if !existed {
			t.Error("Expected Remove(2) to return true for existing element")
		}
		if set.Contains(2) {
			t.Error("Expected element 2 to be removed from set")
		}
		if len(set.m) != 2 {
			t.Errorf("Expected set size 2, got %d", len(set.m))
		}
	})

	t.Run("RemoveNonExistingElement", func(t *testing.T) {
		set := NewSet(1, 2)
		existed := set.Remove(3)
		if existed {
			t.Error("Expected Remove(3) to return false for non-existing element")
		}
		if len(set.m) != 2 {
			t.Errorf("Expected set size 2, got %d", len(set.m))
		}
	})

	t.Run("RemoveFromEmptySet", func(t *testing.T) {
		set := NewSet[int]()
		existed := set.Remove(1)
		if existed {
			t.Error("Expected Remove(1) to return false for empty set")
		}
		if len(set.m) != 0 {
			t.Errorf("Expected set size 0, got %d", len(set.m))
		}
	})
}

func TestToSlice(t *testing.T) {
	t.Run("NonEmptySet", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		slice := set.ToSlice()
		// Sort slice for consistent comparison
		sort.Ints(slice)
		expected := []int{1, 2, 3}
		if !reflect.DeepEqual(slice, expected) {
			t.Errorf("Expected slice %v, got %v", expected, slice)
		}
	})

	t.Run("EmptySet", func(t *testing.T) {
		set := NewSet[int]()
		slice := set.ToSlice()
		if len(slice) != 0 {
			t.Errorf("Expected empty slice, got %v", slice)
		}
	})

	t.Run("StringSet", func(t *testing.T) {
		set := NewSet("a", "b", "c")
		slice := set.ToSlice()
		// Sort slice for consistent comparison
		sort.Strings(slice)
		expected := []string{"a", "b", "c"}
		if !reflect.DeepEqual(slice, expected) {
			t.Errorf("Expected slice %v, got %v", expected, slice)
		}
	})
}

func TestSize(t *testing.T) {
	set := NewSet(1, 2, 3)
	if set.Size() != 3 {
		t.Errorf("Expected set size 3, got %d", set.Size())
	}
	set = NewSet[int]()
	if set.Size() != 0 {
		t.Errorf("Expected set size 0, got %d", set.Size())
	}
	set = NewSet(1, 2, 3, 3, 2, 1)
	if set.Size() != 3 {
		t.Errorf("Expected set size 3, got %d", set.Size())
	}
}

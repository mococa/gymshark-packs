package store

import (
	"errors"
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestNewInMemoryStore(t *testing.T) {
	defaultSizes := []int{250, 500, 1000}
	s := NewInMemoryStore(defaultSizes)

	sizes, err := s.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	sort.Ints(defaultSizes)
	if !reflect.DeepEqual(sizes, defaultSizes) {
		t.Errorf("NewInMemoryStore() sizes = %v, want %v", sizes, defaultSizes)
	}
}

func TestInMemoryStore_Add(t *testing.T) {
	s := NewInMemoryStore([]int{250})

	t.Run("add new size", func(t *testing.T) {
		if err := s.Add(500); err != nil {
			t.Errorf("Add(500) = %v, want nil", err)
		}
		if !s.Exists(500) {
			t.Error("Exists(500) = false after Add(500)")
		}
	})

	t.Run("add duplicate size", func(t *testing.T) {
		if err := s.Add(500); !errors.Is(err, ErrPackSizeExists) {
			t.Errorf("Add(500) duplicate = %v, want ErrPackSizeExists", err)
		}
	})
}

func TestInMemoryStore_Remove(t *testing.T) {
	s := NewInMemoryStore([]int{250, 500, 1000})

	t.Run("remove existing size", func(t *testing.T) {
		if err := s.Remove(500); err != nil {
			t.Errorf("Remove(500) = %v, want nil", err)
		}
		if s.Exists(500) {
			t.Error("Exists(500) = true after Remove(500)")
		}
	})

	t.Run("remove non-existent size", func(t *testing.T) {
		if err := s.Remove(2000); !errors.Is(err, ErrPackSizeNotFound) {
			t.Errorf("Remove(2000) = %v, want ErrPackSizeNotFound", err)
		}
	})

	t.Run("remove already removed size", func(t *testing.T) {
		if err := s.Remove(500); !errors.Is(err, ErrPackSizeNotFound) {
			t.Errorf("Remove(500) second time = %v, want ErrPackSizeNotFound", err)
		}
	})

	t.Run("cannot remove last pack size", func(t *testing.T) {
		single := NewInMemoryStore([]int{250})
		if err := single.Remove(250); !errors.Is(err, ErrLastPackSize) {
			t.Errorf("Remove last size = %v, want ErrLastPackSize", err)
		}
		if !single.Exists(250) {
			t.Error("last pack size was deleted despite error")
		}
	})
}

func TestInMemoryStore_GetAll(t *testing.T) {
	s := NewInMemoryStore([]int{1000, 250, 500})

	sizes, err := s.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	expected := []int{250, 500, 1000}
	if !reflect.DeepEqual(sizes, expected) {
		t.Errorf("GetAll() = %v, want %v (sorted)", sizes, expected)
	}
}

func TestInMemoryStore_Exists(t *testing.T) {
	s := NewInMemoryStore([]int{250, 500})

	if !s.Exists(250) {
		t.Error("Exists(250) = false, want true")
	}
	if s.Exists(1000) {
		t.Error("Exists(1000) = true, want false")
	}
}

func TestInMemoryStore_ThreadSafety(t *testing.T) {
	s := NewInMemoryStore([]int{250})
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(size int) {
			defer wg.Done()
			s.Add(size)
		}(i * 100)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.GetAll()
		}()
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(size int) {
			defer wg.Done()
			s.Remove(size)
		}(i * 100)
	}

	wg.Wait()

	sizes, err := s.GetAll()
	if err != nil {
		t.Fatalf("GetAll() after concurrent ops error = %v", err)
	}
	if len(sizes) < 1 {
		t.Error("store should have at least the initial size (250)")
	}
}

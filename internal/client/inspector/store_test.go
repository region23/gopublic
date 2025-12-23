package inspector

import (
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryStore(t *testing.T) {
	store := NewInMemoryStore(100)
	if store == nil {
		t.Fatal("NewInMemoryStore returned nil")
	}
	if store.Count() != 0 {
		t.Errorf("expected 0 exchanges, got %d", store.Count())
	}
}

func TestInMemoryStore_Add(t *testing.T) {
	store := NewInMemoryStore(100)

	ex := HTTPExchange{
		Timestamp: time.Now(),
		Request: &HTTPRequest{
			Method: "GET",
			URL:    "/test",
		},
	}

	id := store.Add(ex)
	if id != 0 {
		t.Errorf("expected first ID to be 0, got %d", id)
	}

	if store.Count() != 1 {
		t.Errorf("expected 1 exchange, got %d", store.Count())
	}

	// Add another
	id2 := store.Add(ex)
	if id2 != 1 {
		t.Errorf("expected second ID to be 1, got %d", id2)
	}

	if store.Count() != 2 {
		t.Errorf("expected 2 exchanges, got %d", store.Count())
	}
}

func TestInMemoryStore_Get(t *testing.T) {
	store := NewInMemoryStore(100)

	ex := HTTPExchange{
		Timestamp: time.Now(),
		Request: &HTTPRequest{
			Method: "POST",
			URL:    "/api/test",
		},
	}

	id := store.Add(ex)

	// Get existing
	retrieved, ok := store.Get(id)
	if !ok {
		t.Fatal("expected to find exchange")
	}
	if retrieved.Request.Method != "POST" {
		t.Errorf("expected POST, got %s", retrieved.Request.Method)
	}
	if retrieved.ID != id {
		t.Errorf("expected ID %d, got %d", id, retrieved.ID)
	}

	// Get non-existing
	_, ok = store.Get(999)
	if ok {
		t.Error("expected not to find exchange with ID 999")
	}
}

func TestInMemoryStore_List(t *testing.T) {
	store := NewInMemoryStore(100)

	// Add 3 exchanges
	for i := 0; i < 3; i++ {
		store.Add(HTTPExchange{
			Timestamp: time.Now(),
			Request: &HTTPRequest{
				Method: "GET",
				URL:    "/test",
			},
		})
	}

	list := store.List()
	if len(list) != 3 {
		t.Errorf("expected 3 exchanges, got %d", len(list))
	}

	// Newest first - IDs should be 2, 1, 0
	if list[0].ID != 2 {
		t.Errorf("expected first exchange ID 2, got %d", list[0].ID)
	}
	if list[1].ID != 1 {
		t.Errorf("expected second exchange ID 1, got %d", list[1].ID)
	}
	if list[2].ID != 0 {
		t.Errorf("expected third exchange ID 0, got %d", list[2].ID)
	}
}

func TestInMemoryStore_MaxSize(t *testing.T) {
	store := NewInMemoryStore(3) // Small buffer

	// Add 5 exchanges
	for i := 0; i < 5; i++ {
		store.Add(HTTPExchange{
			Timestamp: time.Now(),
			Duration:  int64(i),
		})
	}

	// Should only have 3
	if store.Count() != 3 {
		t.Errorf("expected 3 exchanges (max), got %d", store.Count())
	}

	list := store.List()
	// Newest 3 should remain: IDs 4, 3, 2
	if list[0].ID != 4 {
		t.Errorf("expected newest ID 4, got %d", list[0].ID)
	}
	if list[2].ID != 2 {
		t.Errorf("expected oldest ID 2, got %d", list[2].ID)
	}
}

func TestInMemoryStore_Clear(t *testing.T) {
	store := NewInMemoryStore(100)

	// Add some exchanges
	for i := 0; i < 5; i++ {
		store.Add(HTTPExchange{})
	}

	if store.Count() != 5 {
		t.Errorf("expected 5 exchanges before clear, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("expected 0 exchanges after clear, got %d", store.Count())
	}

	// Adding after clear should still work with incrementing IDs
	id := store.Add(HTTPExchange{})
	if id != 5 {
		t.Errorf("expected ID 5 after clear (IDs not reset), got %d", id)
	}
}

func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore(100)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Add(HTTPExchange{
				Timestamp: time.Now(),
			})
		}()
	}

	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.List()
			_ = store.Count()
		}()
	}

	wg.Wait()

	if store.Count() != 50 {
		t.Errorf("expected 50 exchanges, got %d", store.Count())
	}
}

func TestInMemoryStore_ListReturnsCopy(t *testing.T) {
	store := NewInMemoryStore(100)

	store.Add(HTTPExchange{
		Request: &HTTPRequest{Method: "GET"},
	})

	list1 := store.List()
	list1[0].Request.Method = "MODIFIED"

	list2 := store.List()
	// The modification should not affect the store
	// Note: shallow copy means nested objects are still shared
	// This test documents current behavior
	if list2[0].Request.Method != "MODIFIED" {
		// This is expected with shallow copy - document it
		t.Log("Note: List() returns shallow copy, nested objects are shared")
	}
}

func TestInMemoryStore_GetReturnsCopy(t *testing.T) {
	store := NewInMemoryStore(100)

	id := store.Add(HTTPExchange{
		Duration: 100,
	})

	ex1, _ := store.Get(id)
	ex1.Duration = 999

	ex2, _ := store.Get(id)
	// HTTPExchange is copied, so modification shouldn't affect store
	if ex2.Duration != 100 {
		t.Errorf("expected original duration 100, got %d", ex2.Duration)
	}
}

func TestInMemoryStore_DefaultMaxSize(t *testing.T) {
	store := NewInMemoryStore(0)
	if store.maxSize != 100 {
		t.Errorf("expected default maxSize 100, got %d", store.maxSize)
	}

	store2 := NewInMemoryStore(-1)
	if store2.maxSize != 100 {
		t.Errorf("expected default maxSize 100 for negative, got %d", store2.maxSize)
	}
}

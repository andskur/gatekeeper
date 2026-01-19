package memory

import (
	"context"
	"testing"
	"time"
)

func TestStorageCRUD(t *testing.T) {
	ctx := context.Background()
	s := New()

	if _, err := s.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	if err := s.Set(ctx, "a", 1); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, err := s.Get(ctx, "a")
	if err != nil || v.(int) != 1 {
		t.Fatalf("get: %v val %v", err, v)
	}

	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, "a"); err == nil {
		t.Fatalf("expected not found")
	}
}

func TestStorageCountKeys(t *testing.T) {
	ctx := context.Background()
	s := New()
	_ = s.Set(ctx, "user:1", 1)
	_ = s.Set(ctx, "user:2", 2)
	_ = s.Set(ctx, "other", 3)

	c, err := s.CountKeys(ctx, "user:")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if c != 2 {
		t.Fatalf("expected 2, got %d", c)
	}
}

func TestStorageTTL(t *testing.T) {
	ctx := context.Background()
	s := New()
	if err := s.SetWithExpire(ctx, "tmp", 1, 50*time.Millisecond); err != nil {
		t.Fatalf("set expire: %v", err)
	}
	time.Sleep(70 * time.Millisecond)
	if _, err := s.Get(ctx, "tmp"); err == nil {
		t.Fatalf("expected expired key")
	}
}

func TestStrSetOperations(t *testing.T) {
	ctx := context.Background()
	s := New()
	set := s.StrSet("key")

	if err := set.Add(ctx, "a"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := set.AddExpire(ctx, "b", 50*time.Millisecond); err != nil {
		t.Fatalf("add expire: %v", err)
	}

	exists, err := set.Check(ctx, "a")
	if err != nil || !exists {
		t.Fatalf("check a: %v exists %v", err, exists)
	}

	time.Sleep(70 * time.Millisecond)
	exists, err = set.Check(ctx, "b")
	if err == nil || exists {
		t.Fatalf("expected b to expire")
	}

	if err := set.Remove(ctx, "a"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := set.Check(ctx, "a"); err == nil {
		t.Fatalf("expected remove error")
	}

	vals, err := set.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vals) != 0 {
		t.Fatalf("expected empty list, got %v", vals)
	}
}

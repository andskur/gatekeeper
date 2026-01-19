package jwt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/andskur/sessions"
	"github.com/andskur/sessions/persistance"
	"github.com/andskur/sessions/persistance/memory"
)

const testSecret = "super-secret-key"

// helper to create session
func newSession(t *testing.T, expire time.Duration, st persistance.IStorage) *SessionJwt {
	t.Helper()
	s, err := NewJwtSession([]byte(testSecret), expire, st)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	return s.(*SessionJwt)
}

func TestCreateAndGetWithoutStorage(t *testing.T) {
	ctx := context.Background()
	s := newSession(t, time.Hour, nil)

	data := map[string]interface{}{"user_id": 123, "email": "user@example.com"}
	token, err := s.Create(ctx, data)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	got, err := s.Get(ctx, token)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if got["user_id"].(float64) != 123 {
		t.Fatalf("expected user_id 123, got %v", got["user_id"])
	}
	if got["email"].(string) != "user@example.com" {
		t.Fatalf("unexpected email: %v", got["email"])
	}
}

func TestCreateWithStorageAndDelete(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	s := newSession(t, time.Hour, st)
	uid := uuid.Must(uuid.NewV4())
	data := map[string]interface{}{"uuid": uid, "role": "admin"}

	token, err := s.Create(ctx, data)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Get works
	got, err := s.Get(ctx, token)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if gotUUID, ok := got["uuid"].(uuid.UUID); !ok || gotUUID != uid {
		t.Fatalf("uuid mismatch: %v", got["uuid"])
	}

	// Delete makes token invalid
	if err := s.Delete(ctx, token); err != nil {
		t.Fatalf("delete token: %v", err)
	}
	if _, err := s.Get(ctx, token); !errors.Is(err, sessions.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateWithStorageRequiresUUID(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	s := newSession(t, time.Hour, st)

	_, err := s.Create(ctx, map[string]interface{}{"name": "no-uuid"})
	if err == nil {
		t.Fatalf("expected error without uuid")
	}
	if !errors.Is(err, sessions.ErrDataNotValid) {
		t.Fatalf("expected ErrDataNotValid, got %v", err)
	}
}

func TestGetExpiredToken(t *testing.T) {
	ctx := context.Background()
	s := newSession(t, 1*time.Second, nil)

	token, err := s.Create(ctx, map[string]interface{}{"user": "a"})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	time.Sleep(2 * time.Second)

	if _, err := s.Get(ctx, token); !errors.Is(err, sessions.ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestDeleteWithoutStorage(t *testing.T) {
	ctx := context.Background()
	s := newSession(t, time.Hour, nil)
	token, err := s.Create(ctx, map[string]interface{}{"user": "a"})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := s.Delete(ctx, token); !errors.Is(err, sessions.ErrNoStorage) {
		t.Fatalf("expected ErrNoStorage, got %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	s := newSession(t, time.Hour, st)
	uid := uuid.Must(uuid.NewV4())

	token, err := s.Create(ctx, map[string]interface{}{"uuid": uid, "x": 1})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	newToken, err := s.RefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if string(newToken) == string(token) {
		t.Fatalf("expected new token different from old")
	}

	if _, err := s.Get(ctx, token); !errors.Is(err, sessions.ErrNotFound) {
		t.Fatalf("expected old token invalidated, got %v", err)
	}
	if data, err := s.Get(ctx, newToken); err != nil || data["x"].(float64) != 1 {
		t.Fatalf("expected new token valid, got data %v err %v", data, err)
	}
}

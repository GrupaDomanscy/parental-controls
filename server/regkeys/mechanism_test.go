package regkeys

import (
	"context"
	"testing"
	"time"
)

func TestGenerateNewRegkeyForEmail(t *testing.T) {
	t.Run("generates random string every time", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		store := InitializeStore(ctx, time.Second*5)

		email := "test@example.com"
		regkey1, err := store.GenerateNewRegkeyForEmail(email)
		if err != nil {
			t.Fatalf("failed to generate regkey: %v", err)
		}

		regkey2, err := store.GenerateNewRegkeyForEmail(email)
		if err != nil {
			t.Fatalf("failed to generate regkey: %v", err)
		}

		if regkey1 == regkey2 {
			t.Errorf("expected two different regkeys, got same regkey: %v", regkey1)
		}
	})

	t.Run("saves regkey in the map", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		t.Parallel()
		store := InitializeStore(ctx, time.Second*5)

		email := "test@example.com"
		regkey, err := store.GenerateNewRegkeyForEmail(email)
		if err != nil {
			t.Fatalf("failed to generate regkey: %v", err)
		}

		if store.regkeys[regkey] != email {
			t.Errorf("expected regkey to be saved with email %v, but got %v", email, store.regkeys[regkey])
		}
	})

	t.Run("deleteAfterTTL is working properly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ttl := time.Millisecond * 100
		store := InitializeStore(ctx, ttl)

		email := "test@example.com"
		regkey, err := store.GenerateNewRegkeyForEmail(email)
		if err != nil {
			t.Fatalf("failed to generate regkey: %v", err)
		}

		// Ensure the regkey exists initially
		if _, exists := store.regkeys[regkey]; !exists {
			t.Errorf("expected regkey to be in the map but it's not")
		}

		// Wait for TTL to expire
		time.Sleep(ttl + time.Millisecond*50)

		// Ensure the regkey has been deleted after TTL
		if _, exists := store.regkeys[regkey]; exists {
			t.Errorf("expected regkey to be deleted after TTL, but it still exists")
		}
	})
}

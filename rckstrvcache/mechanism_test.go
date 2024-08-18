package rckstrvcache

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

		value := "test@example.com"
		key, err := store.PutAndGenerateRandomKeyForValue(value)
		if err != nil {
			t.Fatalf("failed to generate random key: %v", err)
		}

		key2, err := store.PutAndGenerateRandomKeyForValue(value)
		if err != nil {
			t.Fatalf("failed to generate random key: %v", err)
		}

		if key == key2 {
			t.Errorf("expected two different random values, got same: %v", key)
		}
	})

	t.Run("saves key in the map", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		t.Parallel()
		store := InitializeStore(ctx, time.Second*5)

		value := "test@example.com"
		key, err := store.PutAndGenerateRandomKeyForValue(value)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		if store.data[key] != value {
			t.Errorf("expected key to be saved with data %v, but got %v", value, store.data[key])
		}
	})

	t.Run("deleteAfterTTL is working properly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ttl := time.Millisecond * 100
		store := InitializeStore(ctx, ttl)

		value := "test@example.com"
		regkey, err := store.PutAndGenerateRandomKeyForValue(value)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		if _, exists := store.data[regkey]; !exists {
			t.Errorf("expected key to be in the map but it's not")
		}

		// Wait for TTL to expire
		time.Sleep(ttl + time.Millisecond*50)

		// Ensure the key has been deleted after TTL
		if _, exists := store.data[regkey]; exists {
			t.Errorf("expected key to be deleted after TTL, but it still exists")
		}
	})
}

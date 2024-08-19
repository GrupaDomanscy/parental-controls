package rckstrvcache

import (
	"errors"
	"testing"
	"time"
)

func TestGenerateNewRegkeyForEmail(t *testing.T) {
	t.Run("generates random key every time", func(t *testing.T) {
		store, errCh, err := InitializeStore(time.Second * 5)
		if err != nil {
			t.Fatalf("failed to initialize store: %v", err)
		}
		defer func() {
			err = store.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		value := "test@example.com"
		key, err := store.Put(value)
		if err != nil {
			t.Fatalf("failed to generate random key: %v", err)
		}

		key2, err := store.Put(value)
		if err != nil {
			t.Fatalf("failed to generate random key: %v", err)
		}

		if key == key2 {
			t.Errorf("expected two different random values, got same: %v", key)
		}

		select {
		case err = <-errCh:
			t.Errorf("received err from listener channel: %v", err)
		default:
			// nothing
		}
	})

	t.Run("saves data properly", func(t *testing.T) {
		store, errCh, err := InitializeStore(time.Second * 5)
		if err != nil {
			t.Fatalf("failed to initialize store: %v", err)
		}
		defer func(store *Store) {
			err := store.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(store)

		value := "test@example.com"
		key, err := store.Put(value)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		valueAfterPut, exists, err := store.Get(key)
		if err != nil {
			t.Fatal(err)
		}

		if !exists {
			t.Errorf("expected value to be saved in database")
		}

		if valueAfterPut != value {
			t.Errorf("expected key to be saved with data %v, but got %v", value, valueAfterPut)
		}

		select {
		case err = <-errCh:
			t.Errorf("received err from listener channel: %v", err)
		default:
			// nothing
		}
	})

	t.Run("delete goroutine is working properly", func(t *testing.T) {
		ttl := time.Millisecond * 100
		store, errCh, err := InitializeStore(ttl)
		if err != nil {
			t.Fatalf("failed to initialize store: %v", err)
		}
		defer func() {
			err = store.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		value := "test@example.com"
		regkey, err := store.Put(value)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		valueAfterPut, exists, err := store.Get(regkey)
		if err != nil {
			t.Fatal(err)
		}

		if !exists {
			t.Errorf("expected value to be saved in database")
		}

		if valueAfterPut != value {
			t.Errorf("expected value '%s', but received '%s'", value, valueAfterPut)
		}

		// Wait for TTL to expire
		time.Sleep(ttl * 2)

		// Ensure the key has been deleted after TTL
		valueAfterPut, exists, err = store.Get(regkey)
		if err != nil {
			t.Fatal(err)
		}

		if exists {
			t.Errorf("expected value to not exist in database")
		}

		select {
		case err = <-errCh:
			t.Errorf("received err from listener channel: %v", err)
		default:
			// nothing
		}
	})

	t.Run("in transaction reverts tx on error", func(t *testing.T) {
		store, errCh, err := InitializeStore(time.Millisecond * 100)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = store.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		err = store.InTransaction(func(store StoreCompatible) error {
			_, err := store.Put("helloworld")
			if err != nil {
				t.Fatal(err)
			}

			return errors.New("some error")
		})
		if err != nil {
			t.Fatal(err)
		}

		_, exists, err := store.Get("helloworld")
		if err != nil {
			return
		}

		if exists {
			t.Errorf("expected value to not exist in db")
		}

		select {
		case err = <-errCh:
			t.Fatal(err)
		default:
			//nothing
		}
	})

	t.Run("in transaction commits data on nil", func(t *testing.T) {
		store, errCh, err := InitializeStore(time.Millisecond * 100)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = store.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		var key string

		err = store.InTransaction(func(store StoreCompatible) error {
			key, err = store.Put("helloworld")
			if err != nil {
				t.Fatal(err)
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		_, exists, err := store.Get(key)
		if err != nil {
			return
		}

		if !exists {
			t.Errorf("expected value to exist in db")
		}

		select {
		case err = <-errCh:
			t.Fatal(err)
		default:
			//nothing
		}
	})
}

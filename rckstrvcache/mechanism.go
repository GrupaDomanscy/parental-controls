package rckstrvcache

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Store struct {
	ctx   context.Context
	mutex sync.Mutex
	data  map[string]string
	ttl   int64
}

func (store *Store) deleteAfterTTL(regkey string) {
	regkeyTTL := time.Now().UnixMilli() + store.ttl

	for {
		select {
		case <-store.ctx.Done():
			return
		default:
			if regkeyTTL < time.Now().UnixMilli() {
				store.mutex.Lock()
				delete(store.data, regkey)
				store.mutex.Unlock()
				return
			}
		}
	}
}

func InitializeStore(ctx context.Context, ttl time.Duration) *Store {
	return &Store{
		ctx:   ctx,
		mutex: sync.Mutex{},
		data:  make(map[string]string),
		ttl:   ttl.Milliseconds(),
	}
}

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("an unknown error occured while trying to generate random bytes using crypto/rand.Read: %w", err)
	}
	return base32.StdEncoding.EncodeToString(b), nil
}

func (store *Store) PutAndGenerateRandomKeyForValue(value string) (string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	key, err := generateRandomString(128)
	if err != nil {
		return "", fmt.Errorf("an unknown error occured while trying to generate a random key: %w", err)
	}
	store.data[key] = value

	go store.deleteAfterTTL(key)

	return key, nil
}

var ErrEntryDoesNotExist = errors.New("entry does not exist")

func (store *Store) Get(key string) (string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	val, ok := store.data[key]
	if !ok {
		return "", ErrEntryDoesNotExist
	}

	return val, nil
}

func (store *Store) Delete(key string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	delete(store.data, key)
}

func (store *Store) GetAllKeys() []string {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	keys := make([]string, 0)

	for key, _ := range store.data {
		keys = append(keys, key)
	}

	return keys
}

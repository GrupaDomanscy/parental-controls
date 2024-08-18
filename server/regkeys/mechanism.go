package regkeys

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type Store struct {
	ctx     context.Context
	mutex   sync.Mutex
	regkeys map[string]string
	ttl     int64
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
				delete(store.regkeys, regkey)
				store.mutex.Unlock()
				return
			}
		}
	}
}

func InitializeStore(ctx context.Context, ttl time.Duration) *Store {
	return &Store{
		ctx:     ctx,
		mutex:   sync.Mutex{},
		regkeys: make(map[string]string),
		ttl:     ttl.Milliseconds(),
	}
}

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("an unknown error occured while trying to generate random bytes using crypto/rand.Read: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (store *Store) GenerateNewRegkeyForEmail(email string) (string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	regkey, err := generateRandomString(128)
	if err != nil {
		return "", fmt.Errorf("an unknown error occured while trying to generate a random string: %w", err)
	}
	store.regkeys[regkey] = email

	go store.deleteAfterTTL(regkey)

	return regkey, nil
}

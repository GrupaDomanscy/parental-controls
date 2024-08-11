package client

import (
	"context"
	"testing"
)

func TestServer(t *testing.T) {
	StartServer(8080, context.Background())
}

package netgear

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStaleTokenReturnsSessionExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><input type="hidden" id="rand" value="1234567"></html>`)
	}))
	defer srv.Close()
	tm := NewMemoryTokenManager()
	address := strings.TrimPrefix(srv.URL, "http://")
	if err := tm.StoreToken(context.Background(), address, "stale", ModelGS308EPP); err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(address, WithTokenManager(tm))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.POE().GetStatus(context.Background())
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("got %v", err)
	}
	if client.IsAuthenticated() {
		t.Fatal("token not cleared")
	}
}

package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

type memoryBackend struct{ value string }

func (b *memoryBackend) Get(_, _ string) (string, error) {
	if b.value == "" {
		return "", keyring.ErrNotFound
	}
	return b.value, nil
}
func (b *memoryBackend) Set(_, _, value string) error { b.value = value; return nil }
func (b *memoryBackend) Delete(_, _ string) error     { b.value = ""; return nil }

func TestStoreRoundTripsCredentialWithoutExposingPlainStorageToCallers(t *testing.T) {
	backend := &memoryBackend{}
	store := NewStoreWithBackend(backend)
	credential := Credential{Token: "secret-token", APIURL: "https://redan.example", ExpiresAt: time.Now().Add(time.Hour), UserID: 7, UserName: "Admin", UserEmail: "admin@example.test"}
	if err := store.Save(credential); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Token != credential.Token || loaded.APIURL != credential.APIURL || !loaded.ExpiresAt.Equal(credential.ExpiresAt) || loaded.UserID != credential.UserID || loaded.UserName != credential.UserName || loaded.UserEmail != credential.UserEmail {
		t.Fatalf("loaded credential = %+v", loaded)
	}
	if err := store.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load after delete = %v", err)
	}
}

func TestStoreDeletesExpiredSession(t *testing.T) {
	backend := &memoryBackend{}
	store := NewStoreWithBackend(backend)
	if err := store.Save(Credential{Token: "expired", ExpiresAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrExpired) {
		t.Fatalf("load expired session = %v", err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session was not deleted: %v", err)
	}
}

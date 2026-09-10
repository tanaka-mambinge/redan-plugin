package auth

import (
	"encoding/json"
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "redan-faq-plugin"
	keyringAccount = "default"
)

var ErrNotFound = errors.New("Redan FAQ authentication is not configured")

type Backend interface {
	Get(service, account string) (string, error)
	Set(service, account, secret string) error
	Delete(service, account string) error
}

type keyringBackend struct{}

func (keyringBackend) Get(service, account string) (string, error) {
	return keyring.Get(service, account)
}
func (keyringBackend) Set(service, account, secret string) error {
	return keyring.Set(service, account, secret)
}
func (keyringBackend) Delete(service, account string) error { return keyring.Delete(service, account) }

type Store struct {
	backend Backend
	service string
	account string
}

type Credential struct {
	Token  string `json:"token"`
	APIURL string `json:"api_url,omitempty"`
}

func NewStore() *Store { return NewStoreWithBackend(keyringBackend{}) }

func NewStoreWithBackend(backend Backend) *Store {
	if backend == nil {
		backend = keyringBackend{}
	}
	return &Store{backend: backend, service: keyringService, account: keyringAccount}
}

func (s *Store) Save(credential Credential) error {
	if credential.Token == "" {
		return errors.New("Redan FAQ API token cannot be empty")
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	return s.backend.Set(s.service, s.account, string(encoded))
}

func (s *Store) Load() (Credential, error) {
	secret, err := s.backend.Get(s.service, s.account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return Credential{}, ErrNotFound
		}
		return Credential{}, errors.New("read Redan FAQ credential from OS keyring: " + err.Error())
	}
	var credential Credential
	if err := json.Unmarshal([]byte(secret), &credential); err != nil {
		return Credential{}, errors.New("Redan FAQ credential in OS keyring is invalid")
	}
	if credential.Token == "" {
		return Credential{}, errors.New("Redan FAQ credential in OS keyring is incomplete; configure it again")
	}
	return credential, nil
}

func (s *Store) Delete() error {
	err := s.backend.Delete(s.service, s.account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

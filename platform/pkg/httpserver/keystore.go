package httpserver

import "sync"

type Key struct {
	Secret  []byte
	Version uint32
}

type KeyStore struct {
	mu   sync.RWMutex
	keys map[string]Key
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		keys: make(map[string]Key),
	}
}

func (s *KeyStore) SetKey(serviceID string, secret []byte, version uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[serviceID] = Key{Secret: secret, Version: version}
}

func (s *KeyStore) GetKey(serviceID string) ([]byte, uint32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.keys[serviceID]
	return k.Secret, k.Version, ok
}

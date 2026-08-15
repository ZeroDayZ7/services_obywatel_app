package hmac

import "sync"

type HMACKey struct {
	Secret  []byte
	Version uint32
}

type GatewayKeyStore struct {
	mu   sync.RWMutex
	keys map[string]HMACKey
}

func NewGatewayKeyStore() *GatewayKeyStore {
	return &GatewayKeyStore{
		keys: make(map[string]HMACKey),
	}
}

func (s *GatewayKeyStore) SetKey(serviceID string, secret []byte, version uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[serviceID] = HMACKey{Secret: secret, Version: version}
}

func (s *GatewayKeyStore) GetKey(serviceID string) ([]byte, uint32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.keys[serviceID]
	return k.Secret, k.Version, ok
}

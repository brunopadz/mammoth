package proxy

import "sync"

type BackendSecrets struct {
	mtx *sync.Mutex
	m   map[int32]map[int32]secret
}

type secret struct {
	origSecret int32
	hostPort   string
}

func NewBackendSecrets() *BackendSecrets {
	return &BackendSecrets{
		m:   make(map[int32]map[int32]secret),
		mtx: &sync.Mutex{},
	}
}

func (s *BackendSecrets) Add(pid, origSecret int32, hostPort string) int32 {
	s.mtx.Lock()
	secrets, ok := s.m[pid]
	if !ok {
		secrets = make(map[int32]secret)
		s.m[pid] = secrets
	}
	newSecret := origSecret
	// Technically this could loop forever, but ... it shouldn't unless
	// you have over 4 billion connections to the same server.
	for {
		_, exists := secrets[newSecret]
		if !exists {
			secrets[newSecret] = secret{
				hostPort:   hostPort,
				origSecret: origSecret,
			}
			break
		}
		newSecret++
	}
	s.mtx.Unlock()
	return newSecret
}

func (s *BackendSecrets) Get(pid, newSecret int32) (secret, bool) {
	s.mtx.Lock()
	secrets, ok := s.m[pid]
	if !ok {
		return secret{}, false
	}
	origSecret, ok := secrets[newSecret]
	if !ok {
		return origSecret, false
	}
	s.mtx.Unlock()
	return origSecret, true
}

func (s *BackendSecrets) Remove(pid, newSecret int32) bool {
	s.mtx.Lock()
	secrets, ok := s.m[pid]
	if !ok {
		return false
	}
	_, ok = secrets[newSecret]
	if !ok {
		return false
	}
	delete(secrets, newSecret)
	if len(secrets) == 0 {
		delete(s.m, pid)
	}
	s.mtx.Unlock()
	return true
}

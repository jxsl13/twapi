package browser

import (
	"sync"
)

func newTokenCache() *tokenCache {
	return &tokenCache{
		tokenMap:      make(map[string]*Token),
		expiringQueue: newExpiringQueue(),
	}
}

type tokenCache struct {
	mu            sync.Mutex
	tokenMap      map[string]*Token
	expiringQueue expiringQueue
}

func (tc *tokenCache) clean() {
	for _, key := range tc.expiringQueue.ExpiredKeys() {
		delete(tc.tokenMap, key)
	}
}

// Get a token for a specific address.
// returns nil in case there is no token to be found
func (tc *tokenCache) Get(address string) *Token {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.clean() // always cleanup expired tokens
	t, ok := tc.tokenMap[address]
	if !ok {
		return nil
	}
	return t
}

// Add a token for a specific server address to the token cache
func (tc *tokenCache) Add(address string, token *Token) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.clean() // always cleanup expired tokens
	tc.tokenMap[address] = token
	tc.expiringQueue.Push(address, token.ExpiresAt)
}

package browser

import (
	"sync"
	"time"
)

// NewConcurrentMap creates a new concurrent map
func NewConcurrentMap(size int) (cm ConcurrentMap) {
	cm.Map = make(map[string]ExpiringServerInfo, size)
	return
}

// ExpiringServerInfo is a serverinfo that has an expiration date
// It is considered as not expiring, when the ExpiresAt value is the time.Time zero value
type ExpiringServerInfo struct {
	ServerInfo
	ExpiresAt time.Time
}

// Expired returns true if the value expired, false otherwise
func (esi *ExpiringServerInfo) Expired() bool {
	// if empty, no expiration was set
	if esi.ExpiresAt.Equal(time.Time{}) {
		return false
	}
	return time.Now().After(esi.ExpiresAt)
}

// ConcurrentMap maps a server address ip:port to an expiring
type ConcurrentMap struct {
	Map map[string]ExpiringServerInfo
	sync.RWMutex
}

// Len returns the number of entries in the map
func (cm *ConcurrentMap) Len() int {
	cm.RLock()
	defer cm.RUnlock()
	return len(cm.Map)
}

// Add an element to the map
func (cm *ConcurrentMap) Add(si ServerInfo, expiresIn time.Duration) {
	esi := ExpiringServerInfo{ServerInfo: si}

	if expiresIn <= 0 {
		esi.ExpiresAt = time.Time{}
	} else {
		esi.ExpiresAt = time.Now().Add(expiresIn)
	}

	cm.Lock()
	cm.Map[esi.Address] = esi
	cm.Unlock()
}

// Get retrieves the ServerInfo
func (cm *ConcurrentMap) Get(key string) (si ServerInfo, ok bool) {
	cm.RLock()
	esi, ok := cm.Map[key]
	cm.RUnlock()

	if ok {
		si = esi.ServerInfo
	}
	return
}

// Delete removes an entry from the map
func (cm *ConcurrentMap) Delete(key string) (ok bool) {
	cm.Lock()
	_, ok = cm.Map[key]
	delete(cm.Map, key)
	cm.Unlock()
	return
}

// Values returns a list with all servers infos
func (cm *ConcurrentMap) Values() (infos []ServerInfo) {
	cm.RLock()
	size := len(cm.Map)
	cm.RUnlock()

	infos = make([]ServerInfo, 0, size+32) // add more apacity if some new servers are added in the mean time.

	cm.RLock()
	for _, value := range cm.Map {
		info := value.ServerInfo
		infos = append(infos, info)
	}
	cm.RUnlock()
	return
}

// Cleanup removes all expired entries
func (cm *ConcurrentMap) Cleanup() {
	cm.Lock()
	for key, value := range cm.Map {
		if value.Expired() {
			delete(cm.Map, key)
		}
	}
	cm.Unlock()
}

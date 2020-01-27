package main

import (
	"sort"
	"sync"
)

// Cache is a simple IP to ServerInfo map
type Cache map[string]*ServerInfo

// ConcurrentServerCache alows for adding and reading data in a multithreaded environment
type ConcurrentServerCache struct {
	*Cache
	sync.RWMutex
}

// NewConcurrentServerCache creates a new server cache with an underlying size of size
func NewConcurrentServerCache(size int) *ConcurrentServerCache {
	m := make(Cache, size)
	c := ConcurrentServerCache{}
	c.Cache = &m
	return &c
}

//CleanUp empties the database
func (c *ConcurrentServerCache) CleanUp() {
	c.Lock()
	cacheSize := len(*c.Cache)
	*c.Cache = make(Cache, cacheSize)
	c.Unlock()
}

// Add adds a new element to the cache
func (c *ConcurrentServerCache) Add(s *ServerInfo) {
	c.Lock()
	defer c.Unlock()

	(*c.Cache)[s.Address] = s
}

// Get eturns the server info of a passed key IP address
func (c *ConcurrentServerCache) Get(key string) (info ServerInfo) {
	c.RLock()
	defer c.RUnlock()

	infoPtr, ok := (*c.Cache)[key]
	info = *infoPtr
	if !ok {
		info = ServerInfo{}
	}
	return
}

// Keys returns a list of Keys
func (c *ConcurrentServerCache) Keys() (keys []string) {
	c.RLock()
	defer c.RUnlock()

	keys = getSortedKeys(c.Cache)
	return
}

// Keys defines string map keys
type keys []string

func (a keys) Len() int           { return len(a) }
func (a keys) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a keys) Less(i, j int) bool { return a[i] < a[j] }

func getSortedKeys(m *Cache) []string {
	keys := make(keys, len(*m))

	i := 0
	for k := range *m {
		keys[i] = k
		i++
	}

	sort.Sort(keys)
	return []string(keys)
}

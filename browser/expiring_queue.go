package browser

import (
	"sort"
	"time"
)

type expiringKey struct {
	Key       string
	ExpiresAt time.Time
}

func (ek *expiringKey) IsExpired() bool {
	return time.Now().After(ek.ExpiresAt)
}

type byExpiresAt []expiringKey

func (a byExpiresAt) Len() int           { return len(a) }
func (a byExpiresAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byExpiresAt) Less(i, j int) bool { return a[i].ExpiresAt.Before(a[j].ExpiresAt) }

func newExpiringQueue() expiringQueue {
	return make(expiringQueue, 0)
}

type expiringQueue []expiringKey

func (pq expiringQueue) Push(key string, expiresAt time.Time) {
	pq = append(pq, expiringKey{
		Key:       key,
		ExpiresAt: expiresAt,
	})
	sort.Sort(byExpiresAt(pq))
}

func (pq expiringQueue) peek() *expiringKey {
	if len(pq) == 0 {
		return nil
	}
	return &pq[0]
}

func (pq *expiringQueue) pop() *expiringKey {
	if pq == nil || len(*pq) == 0 {
		return nil
	}
	ek := (*pq)[0]
	*pq = (*pq)[1:]
	return &ek
}

func (pq expiringQueue) ExpiredKeys() []string {
	if len(pq) == 0 {
		return nil
	}
	expired := []string{}
	for pq.peek().IsExpired() {
		expired = append(expired, pq.pop().Key)
	}
	return expired
}

package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
	"time"
)

type rateEntry struct {
	count   int
	expires time.Time
}
type limiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
}

// Expired entries are pruned and the map has a hard cap; untrusted addresses
// cannot grow process memory without bound. Forwarded headers are not trusted.
func (l *limiter) allow(r *http.Request, scope, email string) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	keys := []string{scope + ":ip:" + ip}
	limits := []int{600}
	if email != "" {
		h := sha256.Sum256([]byte(email))
		keys = append(keys, scope+":email:"+hex.EncodeToString(h[:]))
		limits = append(limits, 12)
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, e := range l.entries {
		if !e.expires.After(now) {
			delete(l.entries, k)
		}
	}
	needed := 0
	for i, k := range keys {
		e := l.entries[k]
		if e.count >= limits[i] {
			return false
		}
		if e.count == 0 {
			needed++
		}
	}
	if len(l.entries)+needed > 10000 {
		return false
	}
	for _, k := range keys {
		e := l.entries[k]
		if e.count == 0 {
			e.expires = now.Add(15 * time.Minute)
		}
		e.count++
		l.entries[k] = e
	}
	return true
}

func (l *limiter) resetEmail(scope, email string) {
	h := sha256.Sum256([]byte(email))
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, scope+":email:"+hex.EncodeToString(h[:]))
}

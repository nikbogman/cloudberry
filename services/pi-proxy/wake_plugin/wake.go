// Package wakeplugin triggers a wake by calling the Pi API's
// existing /wake action, instead of building and broadcasting a
// Wake-on-LAN packet itself (ADR-0012).
package wakeplugin

import (
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// throttleWindow matches caddy-wol's prior per-host throttling, so a
	// burst of held/retried requests during one wake cycle doesn't flood
	// the Pi API.
	throttleWindow = 10 * time.Minute
	requestTimeout = 3 * time.Second
)

type outcome int

const (
	outcomeSucceeded outcome = iota
	outcomeFailed
	outcomeTimedOut
	outcomeThrottled
)

// wakeCaller POSTs to a Pi API wake URL, throttled per URL.
// Call never blocks on or propagates an error: a failed or timed-out
// call is logged and otherwise ignored, so the caller can always proceed
// to hold/retry the proxied request regardless of outcome (ADR-0012).
type wakeCaller struct {
	client *http.Client
	now    func() time.Time

	mu       sync.Mutex
	lastCall map[string]time.Time
}

func newWakeCaller() *wakeCaller {
	return &wakeCaller{
		client:   &http.Client{Timeout: requestTimeout},
		now:      time.Now,
		lastCall: make(map[string]time.Time),
	}
}

func (c *wakeCaller) Call(url string) outcome {
	if !c.shouldCall(url) {
		log.Printf("call_wake_api: throttled, skipping %s", url)
		return outcomeThrottled
	}

	resp, err := c.client.Post(url, "application/json", nil)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			log.Printf("call_wake_api: request to %s timed out: %v", url, err)
			return outcomeTimedOut
		}
		log.Printf("call_wake_api: request to %s failed: %v", url, err)
		return outcomeFailed
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("call_wake_api: %s responded %s", url, resp.Status)
		return outcomeSucceeded
	}
	log.Printf("call_wake_api: %s responded %s (treated as failed)", url, resp.Status)
	return outcomeFailed
}

func (c *wakeCaller) shouldCall(url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if last, called := c.lastCall[url]; called && now.Sub(last) < throttleWindow {
		return false
	}
	c.lastCall[url] = now
	return true
}

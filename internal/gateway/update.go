package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nikbogman/metal/internal/routes"
	"github.com/nikbogman/metal/internal/selfupdate"
)

// Updates keeps gateway current without a Deploy: every Every, it fetches
// the routes file from hostd (only answers while blackberry is awake, which is
// also the only time the stacks change) and checks for a new gateway release.
// Either one ends Run, so main exits and systemd restarts gateway on the new
// routes or binary.
type Updates struct {
	HostdURL   string
	RoutesFile string
	Current    routes.Routes
	Binary     *selfupdate.Updater // nil: routes only
	Router     *Router
	Logger     EventLogger
	Every      time.Duration
	Client     *http.Client

	lastRejected []byte
	lastErr      string
}

// Run returns once gateway should restart, after waiting up to a minute for
// in-flight requests and wakes to finish.
func (u *Updates) Run(ctx context.Context) {
	if u.Client == nil {
		u.Client = &http.Client{Timeout: 10 * time.Second}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(u.Every):
		}
		if u.routesChanged(ctx) || u.binaryUpdated(ctx) {
			break
		}
	}
	for deadline := time.Now().Add(time.Minute); !u.Router.Idle() && time.Now().Before(deadline); {
		time.Sleep(time.Second)
	}
}

func (u *Updates) routesChanged(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.HostdURL+"/routes", nil)
	if err != nil {
		return false
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return false // asleep, most likely: nothing to log
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	next, err := routes.Parse(body, "hostd /routes")
	if err != nil {
		if !bytes.Equal(body, u.lastRejected) { // once per bad file, not every poll
			u.lastRejected = body
			u.Logger.SendEvent("routes_rejected", "failed", nil, map[string]any{"error": err.Error()})
		}
		return false
	}
	if sameRoutes(next, u.Current) {
		return false
	}
	tmp := u.RoutesFile + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err == nil {
		err = os.Rename(tmp, u.RoutesFile)
	}
	if err != nil {
		u.Logger.SendEvent("update_failed", "failed", nil, map[string]any{"error": err.Error()})
		return false
	}
	u.Logger.SendEvent("routes_updated", "succeeded", nil, nil)
	return true
}

func (u *Updates) binaryUpdated(ctx context.Context) bool {
	if u.Binary == nil {
		return false
	}
	tag, updated, err := u.Binary.Update(ctx)
	if err != nil {
		// Once per distinct error: a GitHub outage or a local build would
		// otherwise log every poll.
		if err.Error() != u.lastErr {
			u.lastErr = err.Error()
			if errors.Is(err, selfupdate.ErrLocalBuild) {
				log.Printf("self-update off: %v", err)
			} else {
				u.Logger.SendEvent("update_failed", "failed", nil, map[string]any{"error": err.Error()})
			}
		}
		return false
	}
	u.lastErr = ""
	if updated {
		u.Logger.SendEvent("binary_updated", "succeeded", nil, map[string]any{"from": u.Binary.Version, "to": tag})
	}
	return updated
}

func sameRoutes(a, b routes.Routes) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return bytes.Equal(ja, jb)
}

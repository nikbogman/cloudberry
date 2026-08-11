// Package eventlog ships structured events straight to Grafana Cloud's
// Loki endpoint -- no local collector, not worth running one for this repo.
//
// event_type/outcome/app are Loki stream labels (low-cardinality);
// identity/extra ride in the log line. SendEvent swallows transport
// errors so a logging failure never breaks the caller's actual action.
package eventlog

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const defaultTimeout = 5 * time.Second

type GrafanaCloudLogger struct {
	lokiURL string
	user    string
	apiKey  string
	app     string
	client  *http.Client
}

func NewGrafanaCloudLogger(lokiURL, lokiUser, lokiAPIKey, app string) *GrafanaCloudLogger {
	return &GrafanaCloudLogger{
		lokiURL: lokiURL,
		user:    lokiUser,
		apiKey:  lokiAPIKey,
		app:     app,
		client:  &http.Client{Timeout: defaultTimeout},
	}
}

type lokiPayload struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream lokiStreamLabels `json:"stream"`
	// Loki's push-API format: [unix-nano-timestamp, log line] pairs.
	Values [][2]string `json:"values"`
}

type lokiStreamLabels struct {
	EventType string `json:"event_type"`
	Outcome   string `json:"outcome"`
	App       string `json:"app"`
}

type lokiLine struct {
	Identity *string        `json:"identity"`
	Extra    map[string]any `json:"extra"`
}

// SendEvent ships an event to Grafana Cloud. identity may be nil (marshals
// as JSON null); extra may be nil (marshals as an empty JSON object).
func (g *GrafanaCloudLogger) SendEvent(eventType, outcome string, identity *string, extra map[string]any) {
	if extra == nil {
		extra = map[string]any{}
	}

	line, err := json.Marshal(lokiLine{Identity: identity, Extra: extra})
	if err != nil {
		log.Printf("failed to ship event %s to Grafana Cloud: %v", eventType, err)
		return
	}

	payload := lokiPayload{
		Streams: []lokiStream{
			{
				Stream: lokiStreamLabels{EventType: eventType, Outcome: outcome, App: g.app},
				Values: [][2]string{{strconv.FormatInt(time.Now().UnixNano(), 10), string(line)}},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to ship event %s to Grafana Cloud: %v", eventType, err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, g.lokiURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("failed to ship event %s to Grafana Cloud: %v", eventType, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(g.user, g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		log.Printf("failed to ship event %s to Grafana Cloud: %v", eventType, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("failed to ship event %s to Grafana Cloud: unexpected status %d: %s", eventType, resp.StatusCode, respBody)
	}
}

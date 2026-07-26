// Package eventlog ships structured events straight to Grafana Cloud's
// Loki endpoint. No local collector in front of this -- Grafana Cloud
// doesn't run one on your behalf (Alloy is always self-hosted), and this
// repo has nothing else worth standing up and maintaining a whole agent
// process for (ADR-0014).
//
// event_type/outcome/app are stream labels; identity/extra ride in the log
// line itself, since Loki labels are meant for low-cardinality dimensions,
// not free-form identity strings.
//
// A logging failure here must never break the caller's actual action, so
// SendEvent swallows transport errors.
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
	Values [][2]string      `json:"values"`
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

// SendEvent: identity may be nil (marshals as JSON null); extra may be nil
// (marshals as an empty JSON object).
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

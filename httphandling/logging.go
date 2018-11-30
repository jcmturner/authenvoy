package httphandling

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/jcmturner/authenvoy/config"
)

type accessLog struct {
	SourceIP    string        `json:"SourceIP"`
	StatusCode  int           `json:"StatusCode"`
	Method      string        `json:"Method"`
	ServerHost  string        `json:"ServerHost"`
	Path        string        `json:"Path"`
	QueryString string        `json:"QueryString"`
	Time        time.Time     `json:"Time"`
	Duration    time.Duration `json:"Duration"`
}

func accessLogger(inner http.Handler, c *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UTC()
		ww := NewResponseWriterWrapper(w)
		inner.ServeHTTP(ww, r)
		l := accessLog{
			SourceIP:    r.RemoteAddr,
			StatusCode:  ww.Status(),
			Method:      r.Method,
			ServerHost:  r.Host,
			Path:        r.URL.Path,
			QueryString: r.URL.RawQuery,
			Time:        start,
			Duration:    time.Since(start),
		}
		c.AccessLog(l)
	})
}

type eventLog struct {
	EventID              string    `json:"EventID"`
	Time                 time.Time `json:"Time"`
	LoginName            string    `json:"LoginName"`
	Domain               string    `json:"Domain"`
	Validated            bool      `json:"Validated"`
	ValidationSuccessful bool      `json:"ValidationSuccessful"`
	Message              string    `json:"Message"`
}

// NewEvent creates a new event log item
func NewEvent(loginName, domain string) (eventLog, error) {
	eid, err := uuid.GenerateUUID()
	if err != nil {
		return eventLog{}, err
	}
	return eventLog{
		EventID:   eid,
		Time:      time.Now().UTC(),
		LoginName: loginName,
		Domain:    domain,
	}, nil
}

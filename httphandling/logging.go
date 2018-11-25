package httphandling

import (
	"net/http"
	"time"

	"git-codecommit.eu-west-2.amazonaws.com/v1/repos/awskmsluks/config"
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

package identity

import "time"

type Identity struct {
	Domain      string
	LoginName   string
	DisplayName string
	Email       string
	Groups      []string
	AuthTime    time.Time
	SessionID   string
	Expiry      time.Time
}

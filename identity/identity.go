package identity

import "time"

type Identity struct {
	ValidAuth   bool      `json:"ValidAuth"`
	Domain      string    `json:"Domain"`
	LoginName   string    `json:"LoginName"`
	DisplayName string    `json:"DisplayName"`
	Groups      []string  `json:"Groups"`
	AuthTime    time.Time `json:"AuthTime"`
	SessionID   string    `json:"SessionID"`
	Expiry      time.Time `json:"Expiry"`
}

type Credentials struct {
	LoginName string
	Domain    string
	Password  string
}

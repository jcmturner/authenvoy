package httphandling

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"git-codecommit.eu-west-2.amazonaws.com/v1/repos/awskmsluks/appcodes"
	"git-codecommit.eu-west-2.amazonaws.com/v1/repos/awskmsluks/key"
)

func credentials(c *config.Config) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auditLine, err := newAuditLogLine("Admin Remove Authorized Hosts", c)
		if err != nil {
			respondGeneric(w, http.StatusInternalServerError, appcodes.KeyAdminActionError, "Error processing request")
		}
		keyUUID := requestToKeyUUID(r)
		h, err := hostsFromPost(c, r)
		if err != nil {
			respondGeneric(w, http.StatusBadRequest, appcodes.BadData, "posted data invalid")
			return
		}
		k, err := key.RemoveAuthHosts(c, keyUUID, h.Hosts)
		if err != nil {
			respondGeneric(w, http.StatusInternalServerError, appcodes.KeyAdminActionError, "error updating authorized hosts")
			return
		}
		auditLine.EventType = "Successful Admin Action: Remove Authorized Hosts"
		msg := fmt.Sprintf("Hosts %v removed from authorized list on key %s", h.Hosts, keyUUID)
		auditLog(auditLine, msg, r, c)
		respondWithJSON(w, http.StatusAccepted, k)
		return
	})
}

type Credentials struct {
	LoginName string
	Password  string
}

func credsFromPost(c *config.Config, r *http.Request) (creds Credentials, err error) {
	switch r.Header.Get("Content-Type") {
	case "application/json":

	case "application/x-www-form-urlencoded":

	default:
	}

	reader := io.LimitReader(r.Body, 1024)
	defer r.Body.Close()
	dec := json.NewDecoder(reader)
	err = dec.Decode(&h)
	if err != nil {
		c.ApplicationLogf("error decoding provided JSON into host list: %v", err)
	}
	return
}

func credsJSON(c *config.Config, r *http.Request) (creds Credentials, err error) {
	reader := io.LimitReader(r.Body, 1024)
	defer r.Body.Close()
	dec := json.NewDecoder(reader)
	err = dec.Decode(&creds)
	if err != nil {
		c.ApplicationLogf("error decoding provided JSON into host list: %v", err)
	}
	return
}

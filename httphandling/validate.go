package httphandling

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jcmturner/authenvoy/appcode"
	"github.com/jcmturner/authenvoy/config"
	"github.com/jcmturner/authenvoy/identity"
	"gopkg.in/jcmturner/gokrb5.v6/client"
)

func credentials(c *config.Config) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creds, err := credsFromPost(c, r)
		if err != nil {
			respondGeneric(w, http.StatusBadRequest, appcode.BadData, "posted data invalid")
			return
		}
		event, err := NewEvent(creds.LoginName)
		if err != nil {
			respondGeneric(w, http.StatusInternalServerError, appcode.LoggingErr, "Error processing request")
		}
		c.EventLog(event)

		respondWithJSON(w, http.StatusAccepted)
		return
	})
}

type Credentials struct {
	LoginName string
	Domain    string
	Password  string
}

func credsFromPost(c *config.Config, r *http.Request) (creds Credentials, err error) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		return credsJSON(c, r)
	case "application/x-www-form-urlencoded":
		return credsForm(c, r)
	default:
		return credsJSON(c, r)
	}
	return
}

func credsJSON(c *config.Config, r *http.Request) (creds Credentials, err error) {
	reader := io.LimitReader(r.Body, 1024)
	defer r.Body.Close()
	dec := json.NewDecoder(reader)
	err = dec.Decode(&creds)
	if err != nil {
		c.ApplicationLogf("error decoding provided JSON into credentials: %v", err)
	}
	return
}

func credsForm(c *config.Config, r *http.Request) (creds Credentials, err error) {
	l := r.FormValue("loginName")
	if l == "" {
		err = errors.New("no loginName provided in form data")
		c.ApplicationLogf("error processing form provided credentials: %v", err)
		return
	}
	p := r.FormValue("password")
	if p == "" {
		err = errors.New("no password provided in form data")
		c.ApplicationLogf("error processing form provided credentials: %v", err)
		return
	}
	creds.LoginName = l
	creds.Password = p
	return
}

func krbValidate(c *config.Config, creds Credentials, event eventLog) (bool, identity.Identity, error) {
	cl := client.NewClientWithPassword(creds.LoginName, creds.Domain, creds.Password)
	cl.WithConfig(c.KRB5Conf)
	cl.GoKrb5Conf.DisablePAFXFast = true
	err := cl.Login()
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - login error: %v", err)
		event.Message = err.Error()
		event.ValidationFailed = true
		event.Validated = false
		event.Time = time.Now().UTC()
		c.EventLog(event)
		return false, identity.Identity{}, err
	}
	tkt, key, err := cl.GetServiceTicket(fmt.Sprintf("%s@%s", creds.LoginName, creds.Domain))
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - service ticket error: %v", err)
		event.Message = err.Error()
		event.ValidationFailed = true
		event.Validated = false
		event.Time = time.Now().UTC()
		c.EventLog(event)
		return false, identity.Identity{}, err
	}
}

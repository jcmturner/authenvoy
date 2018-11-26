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
	"gopkg.in/jcmturner/gokrb5.v6/crypto"
	"gopkg.in/jcmturner/gokrb5.v6/iana/adtype"
	"gopkg.in/jcmturner/gokrb5.v6/iana/keyusage"
	"gopkg.in/jcmturner/gokrb5.v6/messages"
	"gopkg.in/jcmturner/gokrb5.v6/pac"
	"gopkg.in/jcmturner/gokrb5.v6/types"
)

func authenticate(c *config.Config) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creds, err := credsFromPost(c, r)
		if err != nil {
			c.ApplicationLogf("bad request: %v", err)
			respondGeneric(w, http.StatusBadRequest, appcode.BadData, "posted data invalid")
			return
		}
		event, err := NewEvent(creds.LoginName, creds.Domain)
		if err != nil {
			c.ApplicationLogf("error generating new event: %v", err)
			respondGeneric(w, http.StatusInternalServerError, appcode.LoggingErr, "Error processing request")
		}
		c.EventLog(event)
		valid, id, err := krbValidate(c, creds, event)
		if err != nil {
			c.ApplicationLogf("error validating: %v", err)
			respondGeneric(w, http.StatusInternalServerError, appcode.ValidationErr, "Validation error")
		}
		if !valid {
			respondWithJSON(w, http.StatusUnauthorized, id)
			return
		}
		respondWithJSON(w, http.StatusAccepted, id)
		return
	})
}

func credsFromPost(c *config.Config, r *http.Request) (creds identity.Credentials, err error) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		return credsJSON(c, r)
	case "application/x-www-form-urlencoded":
		return credsForm(c, r)
	}
	return credsJSON(c, r)
}

func credsJSON(c *config.Config, r *http.Request) (creds identity.Credentials, err error) {
	reader := io.LimitReader(r.Body, 1024)
	defer r.Body.Close()
	dec := json.NewDecoder(reader)
	err = dec.Decode(&creds)
	if err != nil {
		c.ApplicationLogf("error decoding provided JSON into credentials: %v", err)
	}
	return
}

func credsForm(c *config.Config, r *http.Request) (creds identity.Credentials, err error) {
	l := r.FormValue("login-name")
	if l == "" {
		err = errors.New("no loginName provided in form data")
		c.ApplicationLogf("error processing form provided credentials: %v", err)
		return
	}
	d := r.FormValue("domain")
	if d == "" {
		err = errors.New("no domain provided in form data")
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
	creds.Domain = d
	creds.Password = p
	return
}

func krbValidate(c *config.Config, creds identity.Credentials, event eventLog) (bool, identity.Identity, error) {
	id := identity.Identity{
		Domain:      creds.Domain,
		LoginName:   creds.LoginName,
		DisplayName: creds.LoginName,
	}

	//Set up krb client
	cl := client.NewClientWithPassword(creds.LoginName, creds.Domain, creds.Password)
	cl.WithConfig(c.KRB5Conf)
	cl.GoKrb5Conf.DisablePAFXFast = true

	//Login the client
	k, err := login(cl)
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - login error: %v", err)
		validationErrEvent(c, event, err)
		return false, id, err
	}
	//Get a service ticket to itself
	tkt, _, err := cl.GetServiceTicket(fmt.Sprintf("%s@%s", creds.LoginName, creds.Domain))
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - service ticket error: %v", err)
		validationErrEvent(c, event, err)
		return false, id, err
	}
	key, _, err := crypto.GetKeyFromPassword(creds.Password, k.CName, k.CRealm, tkt.EncPart.EType, k.PAData)
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - could not get key from password: %v", err)
		validationErrEvent(c, event, err)
		return false, id, err
	}
	err = ticketDecrypt(&tkt, key)
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - could decrypt service ticket: %v", err)
		validationErrEvent(c, event, err)
		return false, id, err
	}
	//Get additional identity info from service ticket
	id, err = getIdentityInfo(creds, tkt, key, event)
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - could not get identity information: %v", err)
		validationErrEvent(c, event, err)
		return false, id, err
	}
	id.ValidAuth = true
	validationSuccessEvent(c, event)
	return true, id, nil
}

func validationErrEvent(c *config.Config, event eventLog, err error) {
	event.Message = err.Error()
	event.ValidationFailed = true
	event.Validated = false
	event.Time = time.Now().UTC()
	c.EventLog(event)
}

func validationSuccessEvent(c *config.Config, event eventLog) {
	event.Message = "authentication successful"
	event.ValidationFailed = false
	event.Validated = true
	c.EventLog(event)
}

func login(cl client.Client) (messages.ASRep, error) {
	if ok, err := cl.IsConfigured(); !ok {
		return messages.ASRep{}, err
	}
	ASReq, err := messages.NewASReqForTGT(cl.Credentials.Realm, cl.Config, cl.Credentials.CName)
	if err != nil {
		return messages.ASRep{}, fmt.Errorf("error generating new AS_REQ: %v", err)
	}
	return cl.ASExchange(cl.Credentials.Realm, ASReq, 0)
}

func ticketDecrypt(tkt *messages.Ticket, key types.EncryptionKey) error {
	b, err := crypto.DecryptEncPart(tkt.EncPart, key, keyusage.KDC_REP_TICKET)
	if err != nil {
		return fmt.Errorf("error decrypting Ticket EncPart: %v", err)
	}
	var denc messages.EncTicketPart
	err = denc.Unmarshal(b)
	if err != nil {
		return fmt.Errorf("error unmarshaling encrypted part: %v", err)
	}
	tkt.DecryptedEncPart = denc
	return nil
}

func getIdentityInfo(creds identity.Credentials, tkt messages.Ticket, key types.EncryptionKey, event eventLog) (identity.Identity, error) {
	isPAC, pac, err := getPAC(tkt, key)
	if isPAC && err != nil {
		return identity.Identity{}, err
	}
	event.Time = tkt.DecryptedEncPart.AuthTime
	if isPAC {
		// There is a valid PAC. Adding attributes to creds
		dn := creds.LoginName
		if pac.KerbValidationInfo.FullName.String() != "" {
			dn = pac.KerbValidationInfo.FullName.String()
		}
		return identity.Identity{
			Domain:      creds.Domain,
			LoginName:   creds.LoginName,
			DisplayName: dn,
			Groups:      pac.KerbValidationInfo.GetGroupMembershipSIDs(),
			AuthTime:    tkt.DecryptedEncPart.AuthTime,
			SessionID:   event.EventID,
			Expiry:      tkt.DecryptedEncPart.EndTime,
		}, nil
	}
	return identity.Identity{
		Domain:      creds.Domain,
		LoginName:   creds.LoginName,
		DisplayName: creds.LoginName,
		AuthTime:    tkt.DecryptedEncPart.AuthTime,
		SessionID:   event.EventID,
		Expiry:      tkt.DecryptedEncPart.EndTime,
	}, nil
}

func getPAC(tkt messages.Ticket, key types.EncryptionKey) (bool, pac.PACType, error) {
	var isPAC bool
	for _, ad := range tkt.DecryptedEncPart.AuthorizationData {
		if ad.ADType == adtype.ADIfRelevant {
			var ad2 types.AuthorizationData
			err := ad2.Unmarshal(ad.ADData)
			if err != nil {
				continue
			}
			if ad2[0].ADType == adtype.ADWin2KPAC {
				isPAC = true
				var p pac.PACType
				err = p.Unmarshal(ad2[0].ADData)
				if err != nil {
					return isPAC, p, fmt.Errorf("error unmarshaling PAC: %v", err)
				}
				err = p.ProcessPACInfoBuffers(key)
				return isPAC, p, err
			}
		}
	}
	return isPAC, pac.PACType{}, nil
}

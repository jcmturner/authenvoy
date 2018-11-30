package httphandling

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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
			respondGeneric(w, http.StatusBadRequest, "posted data invalid")
			return
		}
		event, err := NewEvent(creds.LoginName, creds.Domain)
		if err != nil {
			c.ApplicationLogf("error generating new event: %v", err)
			respondGeneric(w, http.StatusInternalServerError, "Error processing request")
			return
		}
		event.Message = "new authentication request"
		c.EventLog(event)
		id := krbValidate(c, creds, event)
		code := http.StatusUnauthorized
		if id.Valid {
			code = http.StatusAccepted
		}
		respondWithJSON(w, code, id)
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

func krbValidate(c *config.Config, creds identity.Credentials, event eventLog) identity.Identity {
	id := identity.Identity{
		Domain:      creds.Domain,
		LoginName:   creds.LoginName,
		DisplayName: creds.LoginName,
		SessionID:   event.EventID,
	}

	//Set up krb client
	cl := client.NewClientWithPassword(creds.LoginName, creds.Domain, creds.Password)
	defer cl.Destroy() // Client no longer needed so destroy it.
	cl.WithConfig(c.KRB5Conf)
	cl.GoKrb5Conf.DisablePAFXFast = true

	//Login the client
	k, err := login(cl)
	if err != nil {
		err = fmt.Errorf("validation of credentials failed - login error: %v", err)
		validationErrEvent(c, &event, err)
		return id
	}
	//Login completed without error so user is valid
	id.Valid = true
	id.AuthTime = k.DecryptedEncPart.AuthTime
	id.Expiry = k.DecryptedEncPart.EndTime
	event.Time = k.DecryptedEncPart.AuthTime
	validationSuccessEvent(c, &event)

	//Get a service ticket to itself
	tgsReq, err := messages.NewUser2UserTGSReq(k.CName, k.CRealm, cl.Config, k.Ticket, k.DecryptedEncPart.Key, k.CName, false, k.Ticket)
	if err != nil {
		err = fmt.Errorf("getting identity info failed - error generating TGS_REQ: %v", err)
		validationErrEvent(c, &event, err)
		return id
	}
	_, tgsRep, err := cl.TGSREQ(tgsReq, k.CRealm, k.Ticket, k.DecryptedEncPart.Key, 0)
	if err != nil {
		err = fmt.Errorf("getting identity info failed - service ticket error: %v", err)
		validationErrEvent(c, &event, err)
		return id
	}
	err = ticketDecrypt(&tgsRep.Ticket, k.DecryptedEncPart.Key)
	if err != nil {
		err = fmt.Errorf("getting identity info failed - could decrypt service ticket: %v", err)
		validationErrEvent(c, &event, err)
		return id
	}
	//Get additional identity info from service ticket
	err = addIdentityInfo(&id, creds, tgsRep.Ticket, k.DecryptedEncPart.Key)
	if err != nil {
		err = fmt.Errorf("getting identity info failed - could not get identity information: %v", err)
		validationErrEvent(c, &event, err)
		return id
	}
	return id
}

func validationErrEvent(c *config.Config, event *eventLog, err error) {
	event.Message = err.Error()
	event.ValidationSuccessful = false
	event.Validated = false
	event.Time = time.Now().UTC()
	c.EventLog(*event)
}

func validationSuccessEvent(c *config.Config, event *eventLog) {
	event.Message = "authentication successful"
	event.ValidationSuccessful = true
	event.Validated = true
	c.EventLog(*event)
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

func addIdentityInfo(id *identity.Identity, creds identity.Credentials, tkt messages.Ticket, key types.EncryptionKey) error {
	isPAC, pac, err := getPAC(tkt, key)
	if isPAC && err != nil {
		return err
	}
	if isPAC {
		// There is a valid PAC. Adding attributes to creds
		dn := creds.LoginName
		if pac.KerbValidationInfo.FullName.String() != "" {
			dn = pac.KerbValidationInfo.FullName.String()
		}
		id.DisplayName = dn
		id.Groups = pac.KerbValidationInfo.GetGroupMembershipSIDs()
	}
	return nil
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
				err = p.PACInfoMandatoryBuffers(key)
				return isPAC, p, err
			}
		}
	}
	return isPAC, pac.PACType{}, nil
}

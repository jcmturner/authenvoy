package httphandling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jcmturner/authenvoy/config"
	"github.com/jcmturner/authenvoy/identity"
	"github.com/stretchr/testify/assert"
)

const (
	krb5Conf = `[libdefaults]
  default_realm = TEST.GOKRB5
  dns_lookup_realm = false
  dns_lookup_kdc = false
  ticket_lifetime = 24h
  forwardable = yes
  default_tkt_enctypes = aes256-cts-hmac-sha1-96
  default_tgs_enctypes = aes256-cts-hmac-sha1-96
  noaddresses = false

[realms]
 TEST.GOKRB5 = {
  kdc = 127.0.0.1:88
  admin_server = 127.0.0.1:749
  default_domain = test.gokrb5
 }

[domain_realm]
 .test.gokrb5 = TEST.GOKRB5
 test.gokrb5 = TEST.GOKRB5
`
)

func TestAuthenticateSuccess(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test")
	}

	cf, _ := ioutil.TempFile(os.TempDir(), "TEST-krb5.conf")
	defer os.Remove(cf.Name())
	cf.WriteString(krb5Conf)

	c, err := config.New(8020, cf.Name(), "stdout")
	if err != nil {
		t.Fatalf("could not create new config: %v", err)
	}
	rt := NewRouter(c)

	// Authenticate
	cred := identity.Credentials{
		LoginName: "testuser1",
		Domain:    "TEST.GOKRB5",
		Password:  "passwordvalue",
	}
	pb, _ := json.Marshal(cred)
	url := fmt.Sprintf("/%s/authenticate", APIVersion)
	request, err := http.NewRequest("POST", url, bytes.NewReader(pb))
	if err != nil {
		t.Fatalf("error building request: %v", err)
	}

	// Check authentication is required
	response := httptest.NewRecorder()
	rt.ServeHTTP(response, request)
	assert.Equal(t, http.StatusAccepted, response.Code, "Expected 202 Accepted")

	// Unmarshal into identity
	b := response.Body.Bytes()
	i := new(identity.Identity)
	err = json.Unmarshal(b, i)
	if err != nil {
		t.Fatalf("Response cannot be unmarshaled into a identity struct: %v", err)
	}
	assert.Equal(t, true, i.Valid)
	assert.Equal(t, "testuser1", i.LoginName)
	assert.Equal(t, "TEST.GOKRB5", i.Domain)
	assert.NotEqual(t, "", i.SessionID)
}

func TestAuthenticateFailure(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test")
	}

	cf, _ := ioutil.TempFile(os.TempDir(), "TEST-krb5.conf")
	defer os.Remove(cf.Name())
	cf.WriteString(krb5Conf)

	c, err := config.New(8020, cf.Name(), "stdout")
	if err != nil {
		t.Fatalf("could not create new config: %v", err)
	}
	rt := NewRouter(c)

	// Authenticate
	cred := identity.Credentials{
		LoginName: "testuser1",
		Domain:    "TEST.GOKRB5",
		Password:  "wrongpassword",
	}
	pb, _ := json.Marshal(cred)
	url := fmt.Sprintf("/%s/authenticate", APIVersion)
	request, err := http.NewRequest("POST", url, bytes.NewReader(pb))
	if err != nil {
		t.Fatalf("error building request: %v", err)
	}

	// Check authentication is required
	response := httptest.NewRecorder()
	rt.ServeHTTP(response, request)
	assert.Equal(t, http.StatusUnauthorized, response.Code, "Expected 401 Unauthorized")

	// Unmarshal into identity
	b := response.Body.Bytes()
	i := new(identity.Identity)
	err = json.Unmarshal(b, i)
	if err != nil {
		t.Fatalf("Response cannot be unmarshaled into a identity struct: %v", err)
	}
	assert.Equal(t, false, i.Valid)
	assert.Equal(t, "testuser1", i.LoginName)
	assert.Equal(t, "TEST.GOKRB5", i.Domain)
	assert.Equal(t, "", i.SessionID)
}

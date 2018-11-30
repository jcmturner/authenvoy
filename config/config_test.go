package config

import (
	"io/ioutil"
	"os"
	"testing"

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
 RESDOM.GOKRB5 = {
  kdc = 10.80.88.88:188
  admin_server = 127.0.0.1:749
  default_domain = resdom.gokrb5
 }
  USER.GOKRB5 = {
  kdc = 10.80.88.48:88
  admin_server = 10.80.88.48:464
  default_domain = user.gokrb5
 }
  RES.GOKRB5 = {
  kdc = 10.80.88.49:88
  admin_server = 10.80.88.49:464
  default_domain = res.gokrb5
 }

[domain_realm]
 .test.gokrb5 = TEST.GOKRB5
 test.gokrb5 = TEST.GOKRB5
 .resdom.gokrb5 = RESDOM.GOKRB5
 resdom.gokrb5 = RESDOM.GOKRB5
  .user.gokrb5 = USER.GOKRB5
 user.gokrb5 = USER.GOKRB5
  .res.gokrb5 = RES.GOKRB5
 res.gokrb5 = RES.GOKRB5
`
)

func TestConfig_New(t *testing.T) {
	cf, _ := ioutil.TempFile(os.TempDir(), "TEST-krb5.conf")
	defer os.Remove(cf.Name())
	cf.WriteString(krb5Conf)

	c, err := New(8020, cf.Name(), os.TempDir())
	if err != nil {
		t.Fatalf("could not create new config: %v", err)
	}
	assert.Equal(t, 8020, c.Port)
	assert.Equal(t, os.TempDir(), c.LogPath)
	assert.NotNil(t, c.KRB5Conf)
	assert.Equal(t, os.TempDir()+"/"+AppLog, c.Loggers.Application)
	assert.Equal(t, os.TempDir()+"/"+EventLog, c.Loggers.Event)
	assert.Equal(t, os.TempDir()+"/"+AccessLog, c.Loggers.Access)
	assert.NotNil(t, c.Loggers.ApplicationWriter)
	assert.NotNil(t, c.Loggers.EventWriter)
	assert.NotNil(t, c.Loggers.AccessWriter)
	_, err = New(802000, cf.Name(), os.TempDir())
	if err == nil {
		t.Fatal("should have errored for port number that's too large")
	}
	_, err = New(-123, cf.Name(), os.TempDir())
	if err == nil {
		t.Fatal("should have errored for port number that's too small")
	}
	_, err = New(8088, "/does/not/exist", os.TempDir())
	if err == nil {
		t.Fatal("should have errored for krb5.conf file that does not exist")
	}
	_, err = New(8088, cf.Name(), "/does/not/exist")
	if err == nil {
		t.Fatal("should have errored for a log path that does not exist")
	}

	lps := []string{
		"stdout",
		"stderr",
		"null",
	}
	for _, lp := range lps {
		c, err := New(8020, cf.Name(), lp)
		if err != nil {
			t.Fatalf("could not create new config with log path %s: %v", lp, err)
		}
		assert.Equal(t, 8020, c.Port)
		assert.NotNil(t, c.KRB5Conf)
		assert.Equal(t, lp, c.Loggers.Application)
		assert.Equal(t, lp, c.Loggers.Event)
		assert.Equal(t, lp, c.Loggers.Access)
		assert.NotNil(t, c.Loggers.ApplicationWriter)
		assert.NotNil(t, c.Loggers.EventWriter)
		assert.NotNil(t, c.Loggers.AccessWriter)
	}

}

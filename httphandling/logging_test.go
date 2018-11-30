package httphandling

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jcmturner/authenvoy/config"
	"github.com/stretchr/testify/assert"
)

func TestAccessLogger(t *testing.T) {
	// Simple inner handler func
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		return
	})

	cf, _ := ioutil.TempFile(os.TempDir(), "TEST-krb5.conf")
	defer os.Remove(cf.Name())
	cf.WriteString(krb5Conf)

	// Set the config to have a byte buffer for the access encoder
	c, err := config.New(8088, cf.Name(), os.TempDir())
	defer os.Remove(os.TempDir() + "/" + config.AccessLog)
	defer os.Remove(os.TempDir() + "/" + config.EventLog)
	defer os.Remove(os.TempDir() + "/" + config.AppLog)
	if err != nil {
		t.Fatalf("could not configure: %v", err)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	c.SetAccessLogWriter(enc)

	// Form the request
	request, err := http.NewRequest("GET", "/url?query=string", nil)
	if err != nil {
		t.Fatalf("error building request: %v", err)
	}
	request.Host = "shost"
	request.RemoteAddr = "1.2.3.4:1234"

	// Create accessLogger handler and send request
	response := httptest.NewRecorder()
	handler := accessLogger(inner, c)
	handler.ServeHTTP(response, request)

	assert.True(t, strings.HasPrefix(b.String(), `{"SourceIP":"1.2.3.4:1234","StatusCode":204,"Method":"GET","ServerHost":"shost","Path":"/url","QueryString":"query=string","Time":"`), "Log line is %s", b.String())

	j := json.NewDecoder(&b)
	var l accessLog
	err = j.Decode(&l)
	if err != nil {
		t.Errorf("could not decode access log into JSON object: %v", err)
	}
	assert.NotZero(t, l.Time, "Time in access log is not set")
	assert.NotZero(t, l.Duration, "Duration of request is zero")
}

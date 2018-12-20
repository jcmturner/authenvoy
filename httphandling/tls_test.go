package httphandling

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestListenAndServeTLS(t *testing.T) {
	serverr := make(chan error, 1)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", testHandler)
		serverr <- ListenAndServeTLS("127.0.0.1:10443", mux)
	}()
	select {
	case err := <-serverr:
		t.Fatalf("ListenAndServeTLS: %v", err)
	case <-time.After(2 * time.Second):
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get("https://127.0.0.1:10443/")
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Errorf("did not get expected response from TLS server: %v", err)
		}
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

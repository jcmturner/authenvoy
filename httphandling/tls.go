package httphandling

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"time"
)

// generateSelfSignedTLSKeyPairData generates a self signed key pair for testing use.
func generateSelfSignedCert() (cert tls.Certificate, key *rsa.PrivateKey, err error) {
	key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 20 * 365 * 24)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Authentication Envoy"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
	template.DNSNames = append(template.DNSNames, "localhost")
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return
	}
	certPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	cert, err = tls.X509KeyPair(certPEMBytes, keyPEMBytes)
	return
}

// ListenAndServeTLS starts a HTTPS listener with an auto generated self signed certificate.
func ListenAndServeTLS(addr string, handler http.Handler) error {
	cert, _, err := generateSelfSignedCert()
	if err != nil {
		return err
	}
	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		Rand:                     rand.Reader,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites:             []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		// The black list includes the cipher suite that TLS 1.2 makes mandatory, which means that TLS 1.2 deployments
		// could have non-intersecting sets of permitted cipher suites. To avoid this problem causing TLS handshake
		// failures, deployments of HTTP/2 that use TLS 1.2 MUST support TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
		// [TLS-ECDHE] with the P-256 elliptic curve [FIPS186].
	}
	s := http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: &cfg,
	}
	return s.ListenAndServeTLS("", "")
}

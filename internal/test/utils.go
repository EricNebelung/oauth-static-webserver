package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/labstack/gommon/random"

	log "github.com/sirupsen/logrus"
)

func RandHex(n uint8) string {
	r := random.New()
	return r.String(n, random.Hex)
}

func PrepareContentFolder() (func(), string, error) {
	// generate random 8 hex chars for tmp dir
	chars := RandHex(8)
	dirName, err := os.MkdirTemp("", fmt.Sprintf("oauth-static-webserver-%s", chars))
	if err != nil {
		return nil, "", err
	}
	// create 3 page dirs with file.txt (content the counter 1 to 4)
	for i := 0; i < 4; i++ {
		pageDir := fmt.Sprintf("%s/page%d", dirName, i+1)
		err := os.Mkdir(pageDir, 0o755)
		if err != nil {
			_ = os.RemoveAll(dirName)
			return nil, "", err
		}
		f, err := os.Create(fmt.Sprintf("%s/file.txt", pageDir))
		if err != nil {
			_ = os.RemoveAll(dirName)
			return nil, "", err
		}
		_, err = f.WriteString(fmt.Sprintf("page=%d", i+1))
		if err != nil {
			_ = os.RemoveAll(dirName)
			return nil, "", err
		}
		_ = f.Close()
	}
	return func() {
		_ = os.RemoveAll(dirName)
	}, dirName, nil
}

func HttpClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Timeout: time.Second * 10, Jar: jar}
}

func AssertBodyString(t *testing.T, res *http.Response, expected string) {
	t.Helper()
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	assert.Equal(t, expected, body)
}

// GetFreePort asks the kernel for a free open port that is ready to use.
// From: https://gist.github.com/sevkin/96bdae9274465b2d09191384f86ef39d
func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer func(l *net.TCPListener) {
				err := l.Close()
				if err != nil {
					log.WithError(err).Error("Failed to close listener")
				}
			}(l)
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

// CreateTempCert creates a temporary self-signed certificate and key for testing purposes.
// It returns a cleanup function to remove the temporary files, along with the paths to the certificate and key.
func CreateTempCert(t *testing.T) (func(), string, string) {
	t.Helper()
	certDir, err := os.MkdirTemp("", "oauth2-proxy-test-cert")
	if err != nil {
		t.Fatal(err)
	}
	certPath := fmt.Sprintf("%s/cert.pem", certDir)
	keyPath := fmt.Sprintf("%s/key.pem", certDir)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Co"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatal(err)
	}
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		t.Fatal(err)
	}
	err = certOut.Close()
	if err != nil {
		t.Fatal(err)
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		t.Fatal(err)
	}
	err = keyOut.Close()
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		_ = os.RemoveAll(certDir)
	}, certPath, keyPath
}

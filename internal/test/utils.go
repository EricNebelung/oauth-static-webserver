package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/labstack/gommon/random"
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
	// create 3 page dirs with file.txt (content the counter 1 to 3)
	for i := 0; i < 3; i++ {
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

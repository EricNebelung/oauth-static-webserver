package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"oauth-static-webserver/test"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/oauth2-proxy/mockoidc"
)

func prepare(t *testing.T) (func(), *mockoidc.MockOIDC, error) {
	t.Helper()
	shutdown, m, ws, err := test.SetupSWS()
	if err != nil {
		t.Fatal(err)
	}
	wsShutdown, err := ws.StartAsync()
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		wsShutdown()
		shutdown()
	}, m, nil
}

func httpClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Timeout: time.Second * 10, Jar: jar}
}

func assertBodyString(t *testing.T, res *http.Response, expected string) {
	t.Helper()
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	assert.Equal(t, expected, body)
}

func TestGetAllUser1(t *testing.T) {
	shutdown, m, err := prepare(t)
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()
	m.QueueUser(test.User1)

	client := httpClient(t)

	res, err := client.Get("http://localhost:8454/page1/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assertBodyString(t, res, "page=1")

	res, err = client.Get("http://localhost:8454/page2/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assertBodyString(t, res, "page=2")

	res, err = client.Get("http://localhost:8454/page3/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assertBodyString(t, res, "page=3")
}

func TestGetAllUser2(t *testing.T) {
	shutdown, m, err := prepare(t)
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()
	m.QueueUser(test.User2)

	client := httpClient(t)

	res, err := client.Get("http://localhost:8454/page1/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assertBodyString(t, res, "page=1")

	res, err = client.Get("http://localhost:8454/page2/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assertBodyString(t, res, "page=2")

	res, err = client.Get("http://localhost:8454/page3/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.StatusCode)
}

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	testHelper "oauth-static-webserver/internal/test"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/oauth2-proxy/mockoidc"
	"golang.org/x/net/http2"
)

var (
	User1 = &mockoidc.MockUser{
		Subject:           "1",
		PreferredUsername: "mocker1",
		Groups:            []string{"group-test"},
	}
	User2 = &mockoidc.MockUser{
		Subject:           "2",
		PreferredUsername: "mocker2",
		Groups:            nil,
	}
)

type httpTestEnv struct {
	M      *mockoidc.MockOIDC
	WS     *Webserver
	Client *http.Client
	Config *Config
}

func (h *httpTestEnv) Close() error {
	err := h.M.Shutdown()
	if err != nil {
		return err
	}
	return h.WS.Close()
}

func (h *httpTestEnv) url(path string) string {
	return fmt.Sprintf("%s/%s", h.Config.Content.OIDC.BaseUrl, path)
}

func (h *httpTestEnv) resetClient(t *testing.T) {
	h.Client = testHelper.HttpClient(t)
}

func newHttpTestEnv(t *testing.T, tls *SettingsTLS) httpTestEnv {
	t.Helper()
	cfg, m, ws, err := SetupSWS(tls)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.StartAsync()
	if err != nil {
		t.Fatal(err)
	}
	return httpTestEnv{m, ws, testHelper.HttpClient(t), cfg}
}

// ------ TEST PROTECTION ------

func TestSuccessfulGet(t *testing.T) {
	env := newHttpTestEnv(t, &SettingsTLS{Enabled: false})
	defer func() {
		err := env.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	testGet := func(page int, expectedStatus int, expectedBody string) {
		testGet(t, env, page, expectedStatus, expectedBody)
	}

	// --- test without user ---
	testGet(1, 200, "page=1")
	// no test without a user on protected paths -> mockoidc insert default user
	//testGet(2, 403, "")
	//testGet(3, 403, "")

	// --- test with user1 ---
	env.M.QueueUser(User1)
	env.resetClient(t)

	testGet(1, 200, "page=1")
	testGet(2, 200, "page=2")
	testGet(3, 200, "page=3")
	testGet(4, 403, "You do not have the required permissions to access this resource.")

	// --- test with user2 ---
	env.M.QueueUser(User2)
	env.resetClient(t)

	testGet(1, 200, "page=1")
	testGet(2, 200, "page=2")
	testGet(3, 403, "You do not have the required permissions to access this resource.")
	testGet(4, 200, "page=4")
}

func TestMockOIDCError(t *testing.T) {
	env := newHttpTestEnv(t, &SettingsTLS{Enabled: false})
	defer func() {
		err := env.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	testGet := func(page int, expectedStatus int, expectedBody string) {
		testGet(t, env, page, expectedStatus, expectedBody)
	}

	env.M.QueueError(&mockoidc.ServerError{
		Code:  http.StatusInternalServerError,
		Error: mockoidc.InternalServerError,
	})
	env.M.QueueError(&mockoidc.ServerError{
		Code: http.StatusNotImplemented,
	})

	testGet(1, 200, "page=1")
	testGet(2, 500, "")
	testGet(3, 501, "")
	// after all errors, normal operation should continue
	testGet(1, http.StatusOK, "page=1")
	// auth with default user -> page2 allow access to all authenticated users
	testGet(2, http.StatusOK, "")
	// page3 require group, which default user does not have
	testGet(3, http.StatusForbidden, "")
}

// ------ TEST TLS and HTTP2 ------

func TestHttp2(t *testing.T) {
	env := newHttpTestEnv(t, &SettingsTLS{Enabled: false})
	defer func() {
		err := env.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			// Use non-TLS connection for HTTP2
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	res, err := client.Get(env.url("page1/file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, res.StatusCode)
	proto := res.ProtoMajor
	if proto != 2 {
		t.Fatalf("expected HTTP2 response, got HTTP/%d", proto)
	}
}

func TestHttps(t *testing.T) {
	certClose, certFile, keyFile := testHelper.CreateTempCert(t)
	env := newHttpTestEnv(t, &SettingsTLS{
		Enabled:      true,
		HTTPRedirect: true,
		CertFile:     certFile,
		KeyFile:      keyFile,
	})
	defer func() {
		certClose()
		err := env.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// wait a bit for server to start
	time.Sleep(250 * time.Millisecond)

	// client that skips cert verification (self-signed cert)
	client := &http.Client{Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	res, err := client.Get(strings.Replace(env.url("page1/file.txt"), "http://", "https://", 1))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, res.StatusCode)
	proto := res.ProtoMajor
	if proto != 2 {
		t.Fatalf("expected HTTP2 response, got HTTP/%d", proto)
	}
}

// ------ HELPERS ------

// testGet function to simplify repetitive tests
func testGet(t *testing.T, env httpTestEnv, page int, expectedStatus int, expectedBody string) {
	t.Helper()
	result, err := env.Client.Get(env.url(fmt.Sprintf("page%d/file.txt", page)))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expectedStatus, result.StatusCode)
	if expectedBody != "" {
		testHelper.AssertBodyString(t, result, expectedBody)
	}
}

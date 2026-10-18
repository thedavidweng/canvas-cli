package oauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTransportRefreshOnceAcrossClientsAndPages(t *testing.T) {
	refreshes, requests := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > 1 && r.Header.Get("Authorization") != "Bearer replacement" {
			t.Error("replacement bearer not reused")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	origin, _ := url.Parse(server.URL)
	transport := &Transport{Origin: origin, Token: "expired", Base: http.DefaultTransport, Refresh: func(context.Context) (string, error) { refreshes++; return "replacement", nil }}
	for range 2 {
		client := &http.Client{Transport: transport}
		resp, err := client.Get(server.URL + "/api/v1/courses")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 401 {
			t.Fatal(resp.StatusCode)
		}
	}
	if requests != 3 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d", requests, refreshes)
	}
}

func TestTransportReplaysBodyAndBlocksForeignOrigin(t *testing.T) {
	calls, refreshes := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if string(body) != "mutation-body" {
			t.Error("body lost")
		}
		if calls == 1 {
			w.WriteHeader(401)
		}
	}))
	defer server.Close()
	origin, _ := url.Parse(server.URL)
	transport := &Transport{Origin: origin, Token: "old", Base: http.DefaultTransport, Refresh: func(context.Context) (string, error) { refreshes++; return "new", nil }}
	client := &http.Client{Transport: transport}
	resp, err := client.Post(server.URL, "text/plain", strings.NewReader("mutation-body"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if calls != 2 || refreshes != 1 {
		t.Fatal("body not replayed after refresh")
	}
	for _, foreign := range []string{"http://127.0.0.1:1/", "https://" + origin.Host + "/"} {
		if _, err := client.Get(foreign); err == nil {
			t.Fatal("foreign origin allowed")
		}
	}
	if calls != 2 || refreshes != 1 {
		t.Fatal("foreign origin triggered request/refresh")
	}
}

func TestTransportDoesNotReplayStreamingBody(t *testing.T) {
	refreshes, calls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(401) }))
	defer server.Close()
	origin, _ := url.Parse(server.URL)
	transport := &Transport{Origin: origin, Token: "old", Base: http.DefaultTransport, Refresh: func(context.Context) (string, error) { refreshes++; return "new", nil }}
	client := &http.Client{Transport: transport}
	resp, err := client.Post(server.URL, "text/plain", io.NopCloser(strings.NewReader("stream")))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if refreshes != 0 || calls != 1 {
		t.Fatal("stream was refreshed or replayed")
	}
}

func TestTokenEndpointErrorsDoNotExposeCredentials(t *testing.T) {
	for _, scenario := range []string{"reject", "malformed", "redirect", "missing-access"} {
		t.Run(scenario, func(t *testing.T) {
			redirected := false
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
			defer target.Close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch scenario {
				case "reject":
					w.WriteHeader(401)
					fmt.Fprint(w, "secret-access secret-refresh secret-client secret-code secret-verifier")
				case "malformed":
					fmt.Fprint(w, `{"access_token":"secret-access","expires_in":"secret-refresh"}`)
				case "redirect":
					http.Redirect(w, r, target.URL+"/?code=secret-code", http.StatusFound)
				case "missing-access":
					fmt.Fprint(w, `{"refresh_token":"secret-refresh"}`)
				}
			}))
			defer server.Close()
			client := Client{BaseURL: server.URL, ClientID: "client", ClientSecret: "secret-client"}
			_, err := client.Exchange(context.Background(), "secret-code", "secret-verifier")
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "secret-") || redirected {
				t.Fatal("credential leaked or redirect followed")
			}
		})
	}
}

func TestOAuthBaseURLSecurity(t *testing.T) {
	for _, raw := range []string{"http://canvas.example", "https://user@canvas.example", "https://canvas.example/path", "https://canvas.example?secret=value", "ftp://127.0.0.1"} {
		if _, err := ValidateBaseURL(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

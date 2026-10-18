package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Transport struct {
	Origin  *url.URL
	Token   string
	Refresh func(context.Context) (string, error)
	Base    http.RoundTripper

	mu        sync.Mutex
	attempted bool
}

func (t *Transport) String() string { return "OAuth transport (credentials redacted)" }

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !strings.EqualFold(req.URL.Scheme, t.Origin.Scheme) || !strings.EqualFold(req.URL.Host, t.Origin.Host) {
		return nil, fmt.Errorf("OAuth request blocked: URL must match the configured Canvas origin")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	first := req.Clone(req.Context())
	first.Header.Set("Authorization", "Bearer "+t.Token)
	resp, err := t.Base.RoundTrip(first)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || t.attempted || t.Refresh == nil {
		return resp, err
	}
	if req.Body != nil && req.GetBody == nil {
		return resp, nil
	}
	t.attempted = true
	resp.Body.Close()
	token, err := t.Refresh(req.Context())
	if err != nil {
		return nil, err
	}
	t.Token = token
	retry := req.Clone(req.Context())
	retry.Header.Set("Authorization", "Bearer "+token)
	if req.GetBody != nil {
		retry.Body, err = req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("OAuth request body could not be replayed")
		}
	}
	return t.Base.RoundTrip(retry)
}

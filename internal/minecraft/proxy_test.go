package minecraft

import (
	"net/http"
	"testing"
)

func TestSetProxyConfiguresTransports(t *testing.T) {
	defer func() { _ = SetProxy("") }()

	if err := SetProxy("http://127.0.0.1:7890"); err != nil {
		t.Fatal(err)
	}
	hc := NewHTTPClient(Source{ID: "test"})
	tr, ok := hc.Client.Transport.(*http.Transport)
	if !ok || tr == nil {
		t.Fatalf("transport = %#v", hc.Client.Transport)
	}
	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
	u, err := tr.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if u == nil || u.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %v", u)
	}

	if err := SetProxy(""); err != nil {
		t.Fatal(err)
	}
	hc = NewHTTPClient(Source{ID: "test"})
	tr = hc.Client.Transport.(*http.Transport)
	if u, _ := tr.Proxy(req); u != nil && u.String() == "http://127.0.0.1:7890" {
		t.Fatal("proxy should be cleared")
	}
}

func TestSetProxyRejectsInvalidValues(t *testing.T) {
	for _, bad := range []string{"127.0.0.1:7890", "ftp://127.0.0.1:21", "http://"} {
		if err := SetProxy(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
	_ = SetProxy("")
}

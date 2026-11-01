package minecraft

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSkinProfile(t *testing.T) {
	textures := map[string]any{
		"textures": map[string]any{
			"SKIN": map[string]any{
				"url":      "https://textures.example/skin.png",
				"metadata": map[string]string{"model": "slim"},
			},
			"CAPE": map[string]any{
				"url": "https://textures.example/cape.png",
			},
		},
	}
	raw, _ := json.Marshal(textures)
	encoded := base64.StdEncoding.EncodeToString(raw)

	mux := http.NewServeMux()
	mux.HandleFunc("/session/minecraft/profile/abc123", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("unsigned") != "false" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		writeTestJSON(t, w, map[string]any{
			"id":   "abc123",
			"name": "Alex",
			"properties": []any{
				map[string]any{"name": "textures", "value": encoded},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := sessionServerBase
	defer func() { sessionServerBase = old }()
	sessionServerBase = server.URL
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	profile, err := FetchSkinProfile(context.Background(), hc, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Alex" || profile.SkinURL != "https://textures.example/skin.png" {
		t.Fatalf("profile = %#v", profile)
	}
	if profile.CapeURL != "https://textures.example/cape.png" || !profile.Slim {
		t.Fatalf("profile = %#v", profile)
	}
	if profile.SkinModel() != "Slim" {
		t.Fatalf("model = %q", profile.SkinModel())
	}
}

func TestFetchSkinProfileMissingTextures(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session/minecraft/profile/def456", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{"id": "def456", "name": "Steve", "properties": []any{}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := sessionServerBase
	defer func() { sessionServerBase = old }()
	sessionServerBase = server.URL
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	if _, err := FetchSkinProfile(context.Background(), hc, "def456"); err == nil {
		t.Fatal("expected missing textures error")
	}
	if _, err := FetchSkinProfile(context.Background(), hc, ""); err == nil {
		t.Fatal("expected missing UUID error")
	}
}

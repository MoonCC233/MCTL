package minecraft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginMicrosoftDeviceFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/devicecode", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("client_id") != "test-client" {
			t.Fatalf("client_id = %q", r.Form.Get("client_id"))
		}
		writeTestJSON(t, w, map[string]any{
			"device_code":      "device-1",
			"user_code":        "ABCD-EFGH",
			"verification_uri": "https://example.invalid/device",
			"expires_in":       600,
			"interval":         1,
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
			t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
		}
		writeTestJSON(t, w, map[string]any{
			"access_token":  "ms-access",
			"refresh_token": "ms-refresh",
			"expires_in":    3600,
		})
	})
	mux.HandleFunc("/xbox", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"Token": "xbl-token",
			"DisplayClaims": map[string]any{
				"xui": []any{map[string]any{"uhs": "uhs-1", "xid": "xuid-1"}},
			},
		})
	})
	mux.HandleFunc("/xsts", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"Token": "xsts-token",
			"DisplayClaims": map[string]any{
				"xui": []any{map[string]any{"uhs": "uhs-1", "xid": "xuid-1"}},
			},
		})
	})
	mux.HandleFunc("/mclogin", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["identityToken"] != "XBL3.0 x=uhs-1;xsts-token" {
			t.Fatalf("identityToken = %q", body["identityToken"])
		}
		writeTestJSON(t, w, map[string]any{"access_token": "mc-access", "expires_in": 86400})
	})
	mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mc-access" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		writeTestJSON(t, w, map[string]any{
			"id":   "0123456789abcdef0123456789abcdef",
			"name": "Alex",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldDevice, oldToken := microsoftDeviceCodeURL, microsoftTokenURL
	oldXbox, oldXSTS := xboxLiveAuthURL, xstsAuthURL
	oldLogin, oldProfile := minecraftLoginURL, minecraftProfileURL
	oldClient := microsoftHTTPClient
	defer func() {
		microsoftDeviceCodeURL, microsoftTokenURL = oldDevice, oldToken
		xboxLiveAuthURL, xstsAuthURL = oldXbox, oldXSTS
		minecraftLoginURL, minecraftProfileURL = oldLogin, oldProfile
		microsoftHTTPClient = oldClient
	}()
	microsoftDeviceCodeURL = server.URL + "/devicecode"
	microsoftTokenURL = server.URL + "/token"
	xboxLiveAuthURL = server.URL + "/xbox"
	xstsAuthURL = server.URL + "/xsts"
	minecraftLoginURL = server.URL + "/mclogin"
	minecraftProfileURL = server.URL + "/profile"
	microsoftHTTPClient = server.Client()

	var got MicrosoftDeviceCode
	res, err := LoginMicrosoft(context.Background(), "test-client", func(code MicrosoftDeviceCode) {
		got = code
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.UserCode != "ABCD-EFGH" || got.VerificationURI != "https://example.invalid/device" {
		t.Fatalf("device code = %#v", got)
	}
	if res.Name != "Alex" || res.UUID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("result = %#v", res)
	}
	if res.AccessToken != "mc-access" || res.RefreshToken != "ms-refresh" || res.XUID != "xuid-1" {
		t.Fatalf("tokens = %#v", res)
	}
	if res.ExpiresAt == 0 {
		t.Fatal("expiry should be set")
	}
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

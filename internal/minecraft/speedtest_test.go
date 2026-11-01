package minecraft

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestSpeedTestMeasuresTargets(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 512<<10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "" {
			t.Error("speed test should send a Range header")
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	results := SpeedTest(context.Background(), hc, []SpeedTarget{
		{ID: "ok", Name: "OK", URL: server.URL},
		{ID: "dead", Name: "Dead", URL: deadURL},
	}, 128<<10)
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	ok := results[0]
	if ok.Err != nil {
		t.Fatal(ok.Err)
	}
	if ok.Bytes != 128<<10 {
		t.Fatalf("bytes = %d", ok.Bytes)
	}
	if ok.Latency <= 0 || ok.Latency > 10*time.Second {
		t.Fatalf("latency = %v", ok.Latency)
	}
	if ok.MBps <= 0 {
		t.Fatalf("MBps = %f", ok.MBps)
	}
	if results[1].Err == nil {
		t.Fatal("closed server should produce an error")
	}
}

func TestSpeedTestDefaults(t *testing.T) {
	targets := DefaultSpeedTargets()
	if len(targets) < 4 {
		t.Fatalf("targets = %#v", targets)
	}
	for _, target := range targets {
		if target.ID == "" || target.Name == "" || target.URL == "" {
			t.Fatalf("invalid target %#v", target)
		}
	}
}

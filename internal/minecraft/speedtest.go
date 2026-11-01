package minecraft

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SpeedTarget is a network endpoint used by the speed test.
type SpeedTarget struct {
	ID   string
	Name string
	URL  string
}

// SpeedResult is the outcome of one speed test.
type SpeedResult struct {
	Target  SpeedTarget
	Latency time.Duration
	Bytes   int64
	MBps    float64
	Err     error
}

// DefaultSpeedTargets returns the endpoints checked by the speed test.
func DefaultSpeedTargets() []SpeedTarget {
	return []SpeedTarget{
		{ID: "bmclapi", Name: "BMCLAPI 镜像", URL: "https://bmclapi2.bangbang93.com/mc/game/version_manifest_v2.json"},
		{ID: "mojang", Name: "Mojang 官方", URL: "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"},
		{ID: "modrinth", Name: "Modrinth API", URL: "https://api.modrinth.com/v2/search?limit=1"},
		{ID: "github", Name: "GitHub 发布源", URL: "https://github.com/MoonCC233/MCTL/releases.atom"},
		{ID: "adoptium", Name: "Adoptium", URL: "https://api.adoptium.net/v3/info/available_releases"},
	}
}

// SpeedTest samples each target with a ranged request and reports latency and
// throughput. limitBytes caps the amount read per target.
func SpeedTest(ctx context.Context, hc *HTTPClient, targets []SpeedTarget, limitBytes int64) []SpeedResult {
	if limitBytes <= 0 {
		limitBytes = 256 << 10
	}
	client := http.DefaultClient
	if hc != nil && hc.Client != nil {
		client = hc.Client
	}
	results := make([]SpeedResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, speedTestOne(ctx, client, target, limitBytes))
	}
	return results
}

func speedTestOne(ctx context.Context, client *http.Client, target SpeedTarget, limit int64) SpeedResult {
	res := SpeedResult{Target: target}
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target.URL, nil)
	if err != nil {
		res.Err = err
		return res
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", limit-1))
	req.Header.Set("Accept", "*/*")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		res.Err = err
		return res
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		res.Err = fmt.Errorf("HTTP %d", resp.StatusCode)
		return res
	}
	n, err := io.CopyN(io.Discard, resp.Body, limit)
	if err != nil && err != io.EOF {
		res.Err = err
		return res
	}
	elapsed := time.Since(start)
	if elapsed <= 0 {
		elapsed = time.Microsecond
	}
	res.Bytes = n
	res.Latency = elapsed
	if elapsed > 0 {
		res.MBps = float64(n) / (1024 * 1024) / elapsed.Seconds()
	}
	if n == 0 {
		res.Err = fmt.Errorf("没有收到数据")
	}
	return res
}

package minecraft

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UserAgent identifies MCTL to download servers.
const UserAgent = "MCTL/1.0.33 (Minecraft launcher)"

// HTTPClient wraps retries, mirror rewriting and checksum verification.
type HTTPClient struct {
	Client  *http.Client
	Source  Source
	Retries int
}

// NewHTTPClient builds a client for the given source.
func NewHTTPClient(source Source) *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Timeout:   90 * time.Second,
			Transport: newTransport(),
		},
		Source:  source,
		Retries: 3,
	}
}

func (c *HTTPClient) url(raw string) string { return c.Source.Rewrite(raw) }

// Get fetches a small resource into memory.
func (c *HTTPClient) Get(ctx context.Context, rawURL string) ([]byte, error) {
	u := c.url(rawURL)
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, time.Duration(attempt)*600*time.Millisecond); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)
		resp, err := c.Client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 128<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		lastErr = fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
			break
		}
	}
	return nil, lastErr
}

// DownloadFile downloads rawURL to dest. When sha1 is non-empty the content is
// verified before the file is moved into place.
func (c *HTTPClient) DownloadFile(ctx context.Context, rawURL, dest, wantSHA1 string, onBytes func(done, total int64)) error {
	u := c.url(rawURL)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, time.Duration(attempt)*700*time.Millisecond); err != nil {
				return err
			}
		}
		err := c.downloadOnce(ctx, u, dest, wantSHA1, onBytes)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		var pe permanentError
		if errors.As(err, &pe) {
			return pe.error
		}
		lastErr = err
	}
	return lastErr
}

func (c *HTTPClient) downloadOnce(ctx context.Context, u, dest, wantSHA1 string, onBytes func(done, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
			return permanentError{err}
		}
		return err
	}
	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha1.New()
	written := int64(0)
	buf := make([]byte, 256<<10)
	total := resp.ContentLength
	lastReport := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(tmp)
				return werr
			}
			_, _ = h.Write(buf[:n])
			written += int64(n)
			if onBytes != nil && time.Since(lastReport) > 120*time.Millisecond {
				lastReport = time.Now()
				onBytes(written, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(tmp)
			return rerr
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if wantSHA1 != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA1) {
			os.Remove(tmp)
			return fmt.Errorf("校验失败 %s: 期望 %s，实际 %s", filepath.Base(dest), wantSHA1, got)
		}
	}
	if onBytes != nil {
		onBytes(written, written)
	}
	return os.Rename(tmp, dest)
}

type permanentError struct{ error }

func (p permanentError) Unwrap() error { return p.error }

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// FileSHA1 returns the lowercase hex SHA-1 of a file.
func FileSHA1(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

package minecraft

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	proxyMu   sync.RWMutex
	proxyFunc func(*http.Request) (*url.URL, error) = http.ProxyFromEnvironment
)

// SetProxy configures the HTTP proxy for downloads and API calls. An empty
// value restores environment-based proxy detection.
func SetProxy(raw string) error {
	raw = strings.TrimSpace(raw)
	var fn func(*http.Request) (*url.URL, error) = http.ProxyFromEnvironment
	if raw != "" {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("代理地址无效: %s", raw)
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https", "socks5":
		default:
			return fmt.Errorf("不支持的代理协议: %s", u.Scheme)
		}
		fn = http.ProxyURL(u)
	}
	proxyMu.Lock()
	proxyFunc = fn
	proxyMu.Unlock()
	for _, c := range []*http.Client{microsoftHTTPClient, curseForgeHTTPClient} {
		if c == nil {
			continue
		}
		transportFor(c).Proxy = fn
	}
	return nil
}

func currentProxyFunc() func(*http.Request) (*url.URL, error) {
	proxyMu.RLock()
	defer proxyMu.RUnlock()
	return proxyFunc
}

func newTransport() *http.Transport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{Proxy: currentProxyFunc()}
	}
	tr := base.Clone()
	tr.Proxy = currentProxyFunc()
	return tr
}

func transportFor(c *http.Client) *http.Transport {
	if tr, ok := c.Transport.(*http.Transport); ok && tr != nil {
		return tr
	}
	tr := newTransport()
	c.Transport = tr
	return tr
}

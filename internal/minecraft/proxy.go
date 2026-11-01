package minecraft

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// installerProxy reports the proxy host and port used for HTTPS downloads so
// loader installers (which run in a separate JVM) can be pointed at it too.
func installerProxy() (string, string, bool) {
	probe := &http.Request{URL: &url.URL{Scheme: "https", Host: "files.minecraftforge.net"}}
	u, err := currentProxyFunc()(probe)
	if err != nil || u == nil || u.Host == "" {
		return "", "", false
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return host, port, true
}

// ensureLauncherProfiles writes a minimal launcher profile so installers that
// require an existing Minecraft launcher can run on a fresh game directory.
func ensureLauncherProfiles(dir string) error {
	path := filepath.Join(dir, "launcher_profiles.json")
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data := []byte("{\n  \"profiles\": {},\n  \"settings\": {},\n  \"version\": 3\n}\n")
	return os.WriteFile(path, data, 0o644)
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

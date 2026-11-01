// Package config stores MCTL user preferences and accounts.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Account is a Minecraft account known to the launcher.
type Account struct {
	Name         string `json:"name"`
	UUID         string `json:"uuid"`
	Type         string `json:"type"` // "offline" or "microsoft"
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
	XUID         string `json:"xuid,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
}

// Favorite is a saved online project.
type Favorite struct {
	Kind string `json:"kind"` // modpack, resourcepack, shader or mod
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Config is the persisted launcher configuration.
type Config struct {
	GameDir           string     `json:"game_dir"`
	JavaPath          string     `json:"java_path"`
	MinMemory         int        `json:"min_memory"`
	MaxMemory         int        `json:"max_memory"`
	JVMArgs           string     `json:"jvm_args"`
	GameArgs          string     `json:"game_args"`
	Width             int        `json:"width"`
	Height            int        `json:"height"`
	Fullscreen        bool       `json:"fullscreen"`
	DownloadSource    string     `json:"download_source"`
	SelectedVersion   string     `json:"selected_version"`
	SelectedAccount   string     `json:"selected_account"`
	Accounts          []Account  `json:"accounts"`
	Favorites         []Favorite `json:"favorites,omitempty"`
	MicrosoftClientID string     `json:"microsoft_client_id,omitempty"`
	CurseForgeAPIKey  string     `json:"curseforge_api_key,omitempty"`
	AutoJava          bool       `json:"auto_java"`
	ShowSnapshots     bool       `json:"show_snapshots"`
	IsolateGameDir    bool       `json:"isolate_game_dir"`
	LastVersionType   string     `json:"last_version_type"`
	WindowSizeCustom  bool       `json:"window_size_custom"`
	Theme             string     `json:"theme,omitempty"`
	Proxy             string     `json:"proxy,omitempty"`
}

// Store owns the configuration file and serializes writes.
type Store struct {
	mu   sync.Mutex
	Path string
	Cfg  Config
}

// Dir returns the MCTL configuration directory.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("定位配置目录: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "MCTL"), nil
}

// LogDir returns the directory used for launcher log files.
func LogDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}

// DefaultGameDir picks a sensible Minecraft directory. Portable installs (a
// .minecraft folder or mctl.portable marker beside the executable) win.
func DefaultGameDir() string {
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(base, "mctl.portable")); err == nil {
			return filepath.Join(base, ".minecraft")
		}
		if st, err := os.Stat(filepath.Join(base, ".minecraft")); err == nil && st.IsDir() {
			return filepath.Join(base, ".minecraft")
		}
	}
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, ".minecraft")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".minecraft"
	}
	return filepath.Join(home, ".minecraft")
}

func defaults() Config {
	return Config{
		GameDir:         DefaultGameDir(),
		MinMemory:       512,
		MaxMemory:       4096,
		Width:           854,
		Height:          480,
		DownloadSource:  "bmclapi",
		AutoJava:        true,
		ShowSnapshots:   false,
		LastVersionType: "release",
		Accounts: []Account{
			{Name: "Steve", Type: "offline"},
		},
		SelectedAccount: "Steve",
	}
}

func normalize(c *Config) {
	if c.GameDir == "" {
		c.GameDir = DefaultGameDir()
	}
	if c.MinMemory < 256 {
		c.MinMemory = 256
	}
	if c.MaxMemory < c.MinMemory {
		c.MaxMemory = c.MinMemory
	}
	if c.MaxMemory > 65536 {
		c.MaxMemory = 65536
	}
	if c.Width < 320 {
		c.Width = 854
	}
	if c.Height < 240 {
		c.Height = 480
	}
	if c.DownloadSource != "bmclapi" && c.DownloadSource != "official" {
		c.DownloadSource = "bmclapi"
	}
	if len(c.Accounts) == 0 {
		c.Accounts = []Account{{Name: "Steve", Type: "offline"}}
	}
	if c.SelectedAccount == "" {
		c.SelectedAccount = c.Accounts[0].Name
	}
	found := false
	for _, a := range c.Accounts {
		if a.Name == c.SelectedAccount {
			found = true
			break
		}
	}
	if !found {
		c.SelectedAccount = c.Accounts[0].Name
	}
	if c.LastVersionType == "" {
		c.LastVersionType = "release"
	}
	if c.Theme == "" {
		c.Theme = "opencode"
	}
	c.Proxy = strings.TrimSpace(c.Proxy)
}

// Load reads the configuration, creating a default one when absent.
func Load() (*Store, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建配置目录: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	cfg := defaults()
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			// Keep a copy of the broken file rather than destroying it.
			_ = os.WriteFile(path+".broken", data, 0o644)
			cfg = defaults()
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("读取配置: %w", err)
	}
	normalize(&cfg)
	st := &Store{Path: path, Cfg: cfg}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := st.Save(); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// Save atomically writes the current configuration.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalize(&s.Cfg)
	data, err := json.MarshalIndent(s.Cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// Update applies fn under lock and saves.
func (s *Store) Update(fn func(*Config)) error {
	s.mu.Lock()
	fn(&s.Cfg)
	s.mu.Unlock()
	return s.Save()
}

// Snapshot returns a copy safe to read from another goroutine.
func (s *Store) Snapshot() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.Cfg
	c.Accounts = append([]Account(nil), s.Cfg.Accounts...)
	c.Favorites = append([]Favorite(nil), s.Cfg.Favorites...)
	return c
}

// Account returns the selected account.
func (c Config) Account() Account {
	for _, a := range c.Accounts {
		if a.Name == c.SelectedAccount {
			return a
		}
	}
	if len(c.Accounts) > 0 {
		return c.Accounts[0]
	}
	return Account{Name: "Steve", Type: "offline"}
}

// SetAccount adds or updates an account and selects it.
func (s *Store) SetAccount(a Account) error {
	return s.Update(func(c *Config) {
		for i := range c.Accounts {
			if c.Accounts[i].Name == a.Name {
				c.Accounts[i] = a
				c.SelectedAccount = a.Name
				return
			}
		}
		c.Accounts = append(c.Accounts, a)
		c.SelectedAccount = a.Name
	})
}

// SetFavorite adds or removes a saved online project.
func (s *Store) SetFavorite(fav Favorite, on bool) error {
	fav.Kind = strings.TrimSpace(fav.Kind)
	fav.ID = strings.TrimSpace(fav.ID)
	fav.Name = strings.TrimSpace(fav.Name)
	if fav.Kind == "" || fav.ID == "" {
		return errors.New("收藏项缺少类型或 ID")
	}
	return s.Update(func(c *Config) {
		out := c.Favorites[:0]
		for _, f := range c.Favorites {
			if f.Kind == fav.Kind && f.ID == fav.ID {
				continue
			}
			out = append(out, f)
		}
		if on {
			out = append(out, fav)
		}
		c.Favorites = out
	})
}

// IsFavorite reports whether a project is saved.
func (c Config) IsFavorite(kind, id string) bool {
	for _, f := range c.Favorites {
		if f.Kind == kind && f.ID == id {
			return true
		}
	}
	return false
}

// FavoritesOf returns saved projects of one kind.
func (c Config) FavoritesOf(kind string) []Favorite {
	out := make([]Favorite, 0, len(c.Favorites))
	for _, f := range c.Favorites {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

// RemoveAccount removes an account, keeping at least one.
func (s *Store) RemoveAccount(name string) error {
	return s.Update(func(c *Config) {
		if len(c.Accounts) <= 1 {
			return
		}
		out := c.Accounts[:0]
		for _, a := range c.Accounts {
			if a.Name != name {
				out = append(out, a)
			}
		}
		c.Accounts = out
		if c.SelectedAccount == name {
			c.SelectedAccount = c.Accounts[0].Name
		}
	})
}

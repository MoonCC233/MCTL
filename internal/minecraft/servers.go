package minecraft

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Server is one multiplayer entry stored in servers.dat.
type Server struct {
	Name           string
	IP             string
	Icon           string
	AcceptTextures bool
	Hidden         bool
}

// ServersPath returns the multiplayer server list file for a game directory.
func ServersPath(gameDir string) string {
	return filepath.Join(gameDir, "servers.dat")
}

// ListServers reads servers.dat from a game directory. A missing file is not
// an error and yields an empty list.
func ListServers(gameDir string) ([]Server, error) {
	data, err := os.ReadFile(ServersPath(gameDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return decodeServers(data)
}

// AddServer appends a server entry and rewrites servers.dat atomically.
func AddServer(gameDir string, server Server) error {
	server.Name = sanitizeServerName(server.Name)
	if server.IP == "" {
		return fmt.Errorf("服务器地址不能为空")
	}
	list, err := ListServers(gameDir)
	if err != nil {
		return err
	}
	list = append(list, server)
	return writeServers(gameDir, list)
}

// UpdateServer replaces one entry, identified by its position in the file.
func UpdateServer(gameDir string, index int, server Server) error {
	if server.IP == "" {
		return fmt.Errorf("服务器地址不能为空")
	}
	server.Name = sanitizeServerName(server.Name)
	list, err := ListServers(gameDir)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(list) {
		return fmt.Errorf("服务器序号无效: %d", index)
	}
	list[index] = server
	return writeServers(gameDir, list)
}

// DeleteServer removes one entry, identified by its position in the file.
func DeleteServer(gameDir string, index int) error {
	list, err := ListServers(gameDir)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(list) {
		return fmt.Errorf("服务器序号无效: %d", index)
	}
	list = append(list[:index], list[index+1:]...)
	return writeServers(gameDir, list)
}

func sanitizeServerName(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		if r == '\n' || r == '\r' || r == '\x00' {
			continue
		}
		out = append(out, r)
	}
	start, end := 0, len(out)
	for start < end && (out[start] == ' ' || out[start] == '\t') {
		start++
	}
	for end > start && (out[end-1] == ' ' || out[end-1] == '\t') {
		end--
	}
	out = out[start:end]
	if len(out) > 32 {
		out = out[:32]
	}
	if len(out) == 0 {
		return "Minecraft Server"
	}
	return string(out)
}

func decodeServers(data []byte) ([]Server, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("解压 servers.dat: %w", err)
		}
		raw, err := io.ReadAll(io.LimitReader(zr, 32<<20))
		closeErr := zr.Close()
		if err != nil {
			return nil, fmt.Errorf("解压 servers.dat: %w", err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("解压 servers.dat: %w", closeErr)
		}
		data = raw
	}
	_, root, err := readNBT(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解析 servers.dat: %w", err)
	}
	list, _ := root["servers"].(nbtListValue)
	out := make([]Server, 0, len(list.Values))
	for _, raw := range list.Values {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Server{
			Name:           nbtStringValue(entry["name"]),
			IP:             nbtStringValue(entry["ip"]),
			Icon:           nbtStringValue(entry["icon"]),
			AcceptTextures: nbtBoolValue(entry["acceptTextures"]),
			Hidden:         nbtBoolValue(entry["hidden"]),
		})
	}
	return out, nil
}

func encodeServers(list []Server) ([]byte, error) {
	entries := make([]any, 0, len(list))
	for _, s := range list {
		entry := map[string]any{
			"name": nbtStringValue(s.Name),
			"ip":   nbtStringValue(s.IP),
		}
		if s.Icon != "" {
			entry["icon"] = s.Icon
		}
		if s.AcceptTextures {
			entry["acceptTextures"] = int8(1)
		}
		if s.Hidden {
			entry["hidden"] = int8(1)
		}
		entries = append(entries, entry)
	}
	root := map[string]any{
		"servers": nbtListValue{ElementType: nbtCompound, Values: entries},
	}
	var raw bytes.Buffer
	if err := writeNBT(&raw, "", root); err != nil {
		return nil, fmt.Errorf("写入 servers.dat: %w", err)
	}
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	if _, err := zw.Write(raw.Bytes()); err != nil {
		return nil, fmt.Errorf("压缩 servers.dat: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("压缩 servers.dat: %w", err)
	}
	return out.Bytes(), nil
}

func writeServers(gameDir string, list []Server) error {
	data, err := encodeServers(list)
	if err != nil {
		return err
	}
	dir := filepath.Dir(ServersPath(gameDir))
	tmp, err := os.CreateTemp(dir, "servers-*.dat")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, ServersPath(gameDir)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func nbtStringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func nbtBoolValue(v any) bool {
	switch n := v.(type) {
	case int8:
		return n != 0
	case int16:
		return n != 0
	case int32:
		return n != 0
	case int64:
		return n != 0
	}
	return false
}

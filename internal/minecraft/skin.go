package minecraft

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

var sessionServerBase = "https://sessionserver.mojang.com"

// SkinProfile is the public skin and cape information of a player.
type SkinProfile struct {
	Name    string
	UUID    string
	SkinURL string
	CapeURL string
	Slim    bool
}

// FetchSkinProfile reads public skin textures for a UUID.
func FetchSkinProfile(ctx context.Context, hc *HTTPClient, uuid string) (*SkinProfile, error) {
	uuid = strings.ReplaceAll(strings.TrimSpace(uuid), "-", "")
	if uuid == "" {
		return nil, fmt.Errorf("缺少玩家 UUID")
	}
	if hc == nil {
		return nil, fmt.Errorf("缺少 HTTP 客户端")
	}
	endpoint := fmt.Sprintf("%s/session/minecraft/profile/%s?unsigned=false",
		strings.TrimRight(sessionServerBase, "/"), url.PathEscape(uuid))
	data, err := hc.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("没有找到该账户的公开皮肤（离线账户通常没有）: %w", err)
	}
	var resp struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Properties []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析皮肤档案: %w", err)
	}
	profile := &SkinProfile{Name: resp.Name, UUID: resp.ID}
	if profile.UUID == "" {
		profile.UUID = uuid
	}
	var texturesValue string
	for _, p := range resp.Properties {
		if p.Name == "textures" {
			texturesValue = p.Value
			break
		}
	}
	if texturesValue == "" {
		return nil, fmt.Errorf("该玩家没有公开的皮肤数据")
	}
	decoded, err := base64.StdEncoding.DecodeString(texturesValue)
	if err != nil {
		return nil, fmt.Errorf("解码皮肤数据: %w", err)
	}
	var textures struct {
		Textures struct {
			Skin struct {
				URL      string `json:"url"`
				Metadata struct {
					Model string `json:"model"`
				} `json:"metadata"`
			} `json:"SKIN"`
			Cape struct {
				URL string `json:"url"`
			} `json:"CAPE"`
		} `json:"textures"`
	}
	if err := json.Unmarshal(decoded, &textures); err != nil {
		return nil, fmt.Errorf("解析皮肤纹理: %w", err)
	}
	profile.SkinURL = textures.Textures.Skin.URL
	profile.CapeURL = textures.Textures.Cape.URL
	profile.Slim = strings.EqualFold(textures.Textures.Skin.Metadata.Model, "slim")
	if profile.SkinURL == "" {
		return nil, fmt.Errorf("该玩家没有公开的皮肤")
	}
	return profile, nil
}

// SkinModel returns a display label for the skin model.
func (p *SkinProfile) SkinModel() string {
	if p != nil && p.Slim {
		return "Slim"
	}
	return "Classic"
}

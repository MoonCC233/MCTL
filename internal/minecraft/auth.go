package minecraft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Microsoft device-code login endpoints. They are variables so tests can
// substitute local test servers.
var (
	microsoftDeviceCodeURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	microsoftTokenURL      = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	xboxLiveAuthURL        = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuthURL            = "https://xsts.auth.xboxlive.com/xsts/authorize"
	minecraftLoginURL      = "https://api.minecraftservices.com/authentication/login_with_xbox"
	minecraftProfileURL    = "https://api.minecraftservices.com/minecraft/profile"
	microsoftHTTPClient    = &http.Client{Timeout: 45 * time.Second}
)

const microsoftScope = "XboxLive.signin offline_access"

// MicrosoftDeviceCode is shown to the user while the browser flow runs.
type MicrosoftDeviceCode struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Message         string
	ExpiresIn       int
	Interval        int
}

// MicrosoftAuthResult is a completed Minecraft profile and its tokens.
type MicrosoftAuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	Name         string `json:"name"`
	UUID         string `json:"uuid"`
	XUID         string `json:"xuid"`
}

type microsoftDeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Message         string `json:"message"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Error           string `json:"error"`
	ErrorDesc       string `json:"error_description"`
}

type microsoftTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type xboxAuthResponse struct {
	Token         string `json:"Token"`
	DisplayClaims struct {
		XUI []struct {
			UHS string `json:"uhs"`
			XID string `json:"xid"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
	XErr    int64  `json:"XErr"`
	Message string `json:"Message"`
}

type minecraftLoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorMsg    string `json:"errorMessage"`
}

type minecraftProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LoginMicrosoft runs the Microsoft device-code flow and returns a Minecraft
// profile. onCode is called once the user code is ready.
func LoginMicrosoft(ctx context.Context, clientID string, onCode func(MicrosoftDeviceCode)) (*MicrosoftAuthResult, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("未配置 Microsoft 客户端 ID")
	}
	code, err := requestDeviceCode(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if onCode != nil {
		onCode(*code)
	}
	token, err := pollDeviceToken(ctx, clientID, code)
	if err != nil {
		return nil, err
	}
	return completeMicrosoftLogin(ctx, token.AccessToken, token.RefreshToken)
}

// RefreshMicrosoft refreshes an expired Minecraft session.
func RefreshMicrosoft(ctx context.Context, clientID, refreshToken string) (*MicrosoftAuthResult, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, fmt.Errorf("未配置 Microsoft 客户端 ID")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("缺少 Microsoft 刷新令牌，请重新登录")
	}
	form := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {microsoftScope},
	}
	var tr microsoftTokenResponse
	if err := postForm(ctx, microsoftTokenURL, form, &tr); err != nil {
		return nil, fmt.Errorf("刷新 Microsoft 令牌: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, tokenError(tr.Error, tr.ErrorDesc)
	}
	if tr.RefreshToken == "" {
		tr.RefreshToken = refreshToken
	}
	return completeMicrosoftLogin(ctx, tr.AccessToken, tr.RefreshToken)
}

func requestDeviceCode(ctx context.Context, clientID string) (*MicrosoftDeviceCode, error) {
	form := url.Values{
		"client_id": {clientID},
		"scope":     {microsoftScope},
	}
	var resp microsoftDeviceCodeResponse
	if err := postForm(ctx, microsoftDeviceCodeURL, form, &resp); err != nil {
		return nil, fmt.Errorf("请求 Microsoft 设备代码: %w", err)
	}
	if resp.Error != "" {
		return nil, tokenError(resp.Error, resp.ErrorDesc)
	}
	if resp.DeviceCode == "" || resp.UserCode == "" {
		return nil, fmt.Errorf("Microsoft 返回了无效的设备代码")
	}
	if resp.Interval <= 0 {
		resp.Interval = 5
	}
	return &MicrosoftDeviceCode{
		DeviceCode:      resp.DeviceCode,
		UserCode:        resp.UserCode,
		VerificationURI: resp.VerificationURI,
		Message:         resp.Message,
		ExpiresIn:       resp.ExpiresIn,
		Interval:        resp.Interval,
	}, nil
}

func pollDeviceToken(ctx context.Context, clientID string, code *MicrosoftDeviceCode) (*microsoftTokenResponse, error) {
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)
	interval := time.Duration(code.Interval) * time.Second
	if interval < time.Second {
		interval = time.Second
	}
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("Microsoft 登录已超时，请重试")
		}
		form := url.Values{
			"client_id":   {clientID},
			"device_code": {code.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}
		var resp microsoftTokenResponse
		if err := postForm(ctx, microsoftTokenURL, form, &resp); err != nil {
			return nil, fmt.Errorf("等待 Microsoft 登录: %w", err)
		}
		if resp.AccessToken != "" {
			return &resp, nil
		}
		switch resp.Error {
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "":
			return nil, fmt.Errorf("Microsoft 返回了空令牌")
		default:
			return nil, tokenError(resp.Error, resp.ErrorDesc)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func completeMicrosoftLogin(ctx context.Context, msAccessToken, refreshToken string) (*MicrosoftAuthResult, error) {
	xbl, uhs, xuid, err := authenticateXbox(ctx, msAccessToken)
	if err != nil {
		return nil, err
	}
	xsts, xstsUHS, xstsXUID, err := authorizeXSTS(ctx, xbl)
	if err != nil {
		return nil, err
	}
	if xstsUHS != "" {
		uhs = xstsUHS
	}
	if xstsXUID != "" {
		xuid = xstsXUID
	}
	mcToken, expiresIn, err := loginMinecraft(ctx, uhs, xsts)
	if err != nil {
		return nil, err
	}
	profile, err := fetchMinecraftProfile(ctx, mcToken)
	if err != nil {
		return nil, err
	}
	if expiresIn <= 0 {
		expiresIn = 86400
	}
	return &MicrosoftAuthResult{
		AccessToken:  mcToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
		Name:         profile.Name,
		UUID:         profile.ID,
		XUID:         xuid,
	}, nil
}

func authenticateXbox(ctx context.Context, msAccessToken string) (token, uhs, xuid string, err error) {
	body := map[string]any{
		"Properties": map[string]any{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + msAccessToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}
	var resp xboxAuthResponse
	if err := postJSON(ctx, xboxLiveAuthURL, body, &resp); err != nil {
		return "", "", "", fmt.Errorf("Xbox Live 认证: %w", err)
	}
	if resp.Token == "" {
		return "", "", "", fmt.Errorf("Xbox Live 认证失败: %s", resp.Message)
	}
	if len(resp.DisplayClaims.XUI) > 0 {
		uhs = resp.DisplayClaims.XUI[0].UHS
		xuid = resp.DisplayClaims.XUI[0].XID
	}
	return resp.Token, uhs, xuid, nil
}

func authorizeXSTS(ctx context.Context, xblToken string) (token, uhs, xuid string, err error) {
	body := map[string]any{
		"Properties": map[string]any{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xblToken},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}
	var resp xboxAuthResponse
	if err := postJSON(ctx, xstsAuthURL, body, &resp); err != nil {
		return "", "", "", fmt.Errorf("Xbox XSTS 授权: %w", err)
	}
	if resp.Token == "" {
		if resp.XErr != 0 {
			return "", "", "", fmt.Errorf("Xbox XSTS 授权失败（%d）: %s", resp.XErr, resp.Message)
		}
		return "", "", "", fmt.Errorf("Xbox XSTS 授权失败: %s", resp.Message)
	}
	if len(resp.DisplayClaims.XUI) > 0 {
		uhs = resp.DisplayClaims.XUI[0].UHS
		xuid = resp.DisplayClaims.XUI[0].XID
	}
	return resp.Token, uhs, xuid, nil
}

func loginMinecraft(ctx context.Context, uhs, xstsToken string) (string, int, error) {
	body := map[string]string{
		"identityToken": "XBL3.0 x=" + uhs + ";" + xstsToken,
	}
	var resp minecraftLoginResponse
	if err := postJSON(ctx, minecraftLoginURL, body, &resp); err != nil {
		return "", 0, fmt.Errorf("Minecraft 登录: %w", err)
	}
	if resp.AccessToken == "" {
		msg := resp.ErrorMsg
		if msg == "" {
			msg = resp.Error
		}
		if msg == "" {
			msg = "没有返回访问令牌"
		}
		return "", 0, fmt.Errorf("Minecraft 登录失败: %s", msg)
	}
	return resp.AccessToken, resp.ExpiresIn, nil
}

func fetchMinecraftProfile(ctx context.Context, accessToken string) (*minecraftProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, minecraftProfileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := microsoftHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取 Minecraft 档案: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("该 Microsoft 账户没有 Minecraft: Java Edition")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 Minecraft 档案: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var p minecraftProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("解析 Minecraft 档案: %w", err)
	}
	if p.ID == "" || p.Name == "" {
		return nil, fmt.Errorf("Minecraft 档案缺少名称或 UUID")
	}
	return &p, nil
}

func postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := microsoftHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

func postJSON(ctx context.Context, endpoint string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := microsoftHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func tokenError(code, desc string) error {
	switch code {
	case "authorization_declined":
		return fmt.Errorf("Microsoft 登录被拒绝")
	case "expired_token", "bad_verification_code":
		return fmt.Errorf("Microsoft 登录代码已过期，请重试")
	case "slow_down":
		return fmt.Errorf("Microsoft 请求过于频繁，请稍后重试")
	case "invalid_client":
		return fmt.Errorf("Microsoft 客户端 ID 无效")
	default:
		if desc != "" {
			return fmt.Errorf("Microsoft 登录失败: %s", desc)
		}
		if code != "" {
			return fmt.Errorf("Microsoft 登录失败: %s", code)
		}
		return fmt.Errorf("Microsoft 登录失败")
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type HomebridgeClient struct {
	endpoint       string
	username       string
	password       string
	otp            string
	client         *http.Client
	tokenCachePath string
	token          *TokenCache
	showAll        bool
}

type TokenCache struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type Accessory struct {
	UniqueID               string                 `json:"uniqueId"`
	ServiceName            string                 `json:"serviceName"`
	HumanType              string                 `json:"humanType"`
	ServiceCharacteristics []Characteristic       `json:"serviceCharacteristics"`
	AccessoryInformation   map[string]string      `json:"accessoryInformation"`
	Values                 map[string]interface{} `json:"values"`
}

type Characteristic struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
	CanRead     bool        `json:"canRead"`
	CanWrite    bool        `json:"canWrite"`
}

type ToggleAccessory struct {
	UniqueID           string
	Name               string
	HumanType          string
	On                 bool
	OnKnown            bool
	Toggleable         bool
	CharacteristicType string
}

func NewHomebridgeClient(cfg Config, tokenCachePath string) *HomebridgeClient {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	timeout := 10 * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	transport := &http.Transport{
		Proxy:                 nil, // bypass environment proxy settings (common issue on embedded devices)
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     false,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	return &HomebridgeClient{
		endpoint:       endpoint,
		username:       cfg.Username,
		password:       cfg.Password,
		otp:            cfg.OTP,
		client:         &http.Client{Timeout: timeout, Transport: transport},
		tokenCachePath: tokenCachePath,
		showAll:        cfg.ShowAll,
	}
}

func (c *HomebridgeClient) EnsureToken(ctx context.Context) error {
	if c.token == nil {
		if cached, err := c.loadTokenCache(); err == nil {
			c.token = cached
		}
	}

	if c.token != nil && time.Now().Add(30*time.Second).Before(c.token.ExpiresAt) {
		return nil
	}

	return c.login(ctx)
}

func (c *HomebridgeClient) login(ctx context.Context) error {
	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	if c.otp != "" {
		payload["otp"] = c.otp
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("post %s/api/auth/login: %w", c.endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed: %s", strings.TrimSpace(string(data)))
	}

	var auth authResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return err
	}
	if auth.AccessToken == "" {
		return errors.New("auth response missing access_token")
	}

	expiresAt := time.Now().Add(time.Duration(auth.ExpiresIn) * time.Second)
	c.token = &TokenCache{AccessToken: auth.AccessToken, TokenType: auth.TokenType, ExpiresAt: expiresAt}
	return c.saveTokenCache()
}

func (c *HomebridgeClient) authHeader() string {
	if c.token == nil {
		return ""
	}
	if c.token.TokenType == "" {
		return "Bearer " + c.token.AccessToken
	}
	return c.token.TokenType + " " + c.token.AccessToken
}

func (c *HomebridgeClient) GetAccessories(ctx context.Context) ([]ToggleAccessory, error) {
	if err := c.EnsureToken(ctx); err != nil {
		return nil, err
	}

	accessories, err := c.getAccessoriesFrom(ctx, "/api/accessories")
	if err != nil {
		return nil, err
	}
	var toggles []ToggleAccessory
	for _, acc := range accessories {
		name := ""
		if v, ok := acc.AccessoryInformation["Name"]; ok && v != "" {
			name = v
		}
		if name == "" {
			name = acc.ServiceName
		}
		if name == "" {
			name = acc.HumanType
		}

		item := ToggleAccessory{
			UniqueID:  acc.UniqueID,
			Name:      name,
			HumanType: acc.HumanType,
		}
		for _, ch := range acc.ServiceCharacteristics {
			if ch.Type != "On" {
				continue
			}
			item.OnKnown = true
			item.Toggleable = ch.CanWrite
			item.CharacteristicType = "On"
			item.On = valueToBool(ch.Value)
			if v, ok := acc.Values["On"]; ok {
				item.On = valueToBool(v)
			}
			break
		}
		if item.Toggleable || c.showAll {
			toggles = append(toggles, item)
		}
	}

	return toggles, nil
}

func (c *HomebridgeClient) SetAccessoryOn(ctx context.Context, uniqueID string, on bool) error {
	if err := c.EnsureToken(ctx); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"characteristicType": "On",
		"value":              on,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/accessories/%s", c.endpoint, uniqueID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("put %s/api/accessories/%s: %w", c.endpoint, uniqueID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set accessory failed: %s", strings.TrimSpace(string(data)))
	}
	return nil
}

type accessoriesDump struct {
	Timestamp        string          `json:"timestamp"`
	Endpoint         string          `json:"endpoint"`
	Accessories      json.RawMessage `json:"accessories,omitempty"`
	AccessoriesError string          `json:"accessories_error,omitempty"`
}

func (c *HomebridgeClient) DumpAccessories(ctx context.Context, path string) error {
	if err := c.EnsureToken(ctx); err != nil {
		return err
	}

	dump := accessoriesDump{
		Timestamp: time.Now().Format(time.RFC3339),
		Endpoint:  c.endpoint,
	}

	if raw, err := c.getRaw(ctx, "/api/accessories"); err != nil {
		dump.AccessoriesError = err.Error()
	} else {
		dump.Accessories = raw
	}

	data, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *HomebridgeClient) loadTokenCache() (*TokenCache, error) {
	data, err := os.ReadFile(c.tokenCachePath)
	if err != nil {
		return nil, err
	}
	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache.AccessToken == "" {
		return nil, errors.New("token cache missing access_token")
	}
	return &cache, nil
}

func (c *HomebridgeClient) saveTokenCache() error {
	if c.token == nil {
		return nil
	}
	data, err := json.MarshalIndent(c.token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.tokenCachePath, data, 0o600)
}

func (c *HomebridgeClient) getAccessoriesFrom(ctx context.Context, path string) ([]Accessory, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s%s: %w", c.endpoint, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", path, strings.TrimSpace(string(data)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var accessories []Accessory
	if err := json.Unmarshal(data, &accessories); err == nil {
		return accessories, nil
	}

	var wrapped struct {
		Accessories []Accessory `json:"accessories"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Accessories) > 0 {
		return wrapped.Accessories, nil
	}
	return nil, errors.New("unrecognized accessories payload")
}

func (c *HomebridgeClient) getRaw(ctx context.Context, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s%s: %w", c.endpoint, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", path, strings.TrimSpace(string(data)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func valueToBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		return t != "" && t != "0" && t != "false"
	default:
		return false
	}
}

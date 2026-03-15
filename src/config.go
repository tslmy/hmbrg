package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Endpoint        string `json:"endpoint"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	OTP             string `json:"otp,omitempty"`
	ShowAll         bool   `json:"show_all,omitempty"`
	DumpAccessories bool   `json:"dump_accessories,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
}

type RuntimeConfig struct {
	ConfigPath      string
	TokenCachePath  string
	Config          Config
	DumpAccessories bool
}

func LoadConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Endpoint == "" || cfg.Username == "" || cfg.Password == "" {
		return Config{}, errors.New("config must include endpoint, username, and password")
	}
	return cfg, nil
}

func ResolveRuntimeConfig() (RuntimeConfig, error) {
	configPath := flag.String("config", "config.toml", "Path to config TOML")
	overrideEndpoint := flag.String("endpoint", "", "Override Homebridge endpoint")
	overrideUsername := flag.String("username", "", "Override Homebridge username")
	overridePassword := flag.String("password", "", "Override Homebridge password")
	overrideOTP := flag.String("otp", "", "Override Homebridge OTP (if required)")
	overrideShowAll := flag.Bool("show-all", false, "Show non-toggleable accessories")
	overrideDump := flag.Bool("dump-accessories", false, "Dump raw accessories JSON and exit")
	flag.Parse()

	absConfigPath, err := filepath.Abs(*configPath)
	if err != nil {
		return RuntimeConfig{}, err
	}

	cfg, err := LoadConfigFromFile(absConfigPath)
	if err != nil {
		return RuntimeConfig{}, err
	}

	if *overrideEndpoint != "" {
		cfg.Endpoint = *overrideEndpoint
	}
	if *overrideUsername != "" {
		cfg.Username = *overrideUsername
	}
	if *overridePassword != "" {
		cfg.Password = *overridePassword
	}
	if *overrideOTP != "" {
		cfg.OTP = *overrideOTP
	}
	if *overrideShowAll {
		cfg.ShowAll = true
	}
	if *overrideDump {
		cfg.DumpAccessories = true
	}

	tokenCachePath := filepath.Join(filepath.Dir(absConfigPath), "token.json")
	return RuntimeConfig{ConfigPath: absConfigPath, TokenCachePath: tokenCachePath, Config: cfg, DumpAccessories: cfg.DumpAccessories}, nil
}

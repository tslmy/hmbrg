package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	runtimeCfg, err := ResolveRuntimeConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		fmt.Fprintln(os.Stderr, "expected config.toml with endpoint, username, password")
		os.Exit(1)
	}

	client := NewHomebridgeClient(runtimeCfg.Config, runtimeCfg.TokenCachePath)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if runtimeCfg.DumpAccessories {
		dumpPath := filepath.Join(filepath.Dir(runtimeCfg.ConfigPath), "accessories_dump.json")
		if err := client.DumpAccessories(ctx, dumpPath); err != nil {
			fmt.Fprintln(os.Stderr, "dump error:", err)
			os.Exit(1)
		}
		fmt.Println("wrote", dumpPath)
		return
	}

	ui := NewUI(client, 750, 560)
	if err := ui.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "runtime error:", err)
		os.Exit(1)
	}
}

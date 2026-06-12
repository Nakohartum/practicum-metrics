package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/Nakohartum/practicum-metrics/internal/agent"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	fmt.Printf("Build version: %s\n", buildValue(buildVersion))
	fmt.Printf("Build date: %s\n", buildValue(buildDate))
	fmt.Printf("Build commit: %s\n", buildValue(buildCommit))
	if err := run(); err != nil {
		slog.Error("application stopped with error", "error", err)
	}
}

func run() error {
	if err := parseFlags(); err != nil {
		return fmt.Errorf("initialize configuration: %w", err)
	}
	a, err := agent.NewAgentMetrics(configData.pollInterval, configData.reportInterval, int(configData.rateLimit), configData.secretKey, configData.cryptoKey)
	if err != nil {
		return fmt.Errorf("initialize agent: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	if err := a.Run(ctx, configData.address.String()); err != nil {
		return fmt.Errorf("run agent: %w", err)
	}
	return nil
}

func buildValue(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}

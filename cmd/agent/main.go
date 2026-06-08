package main

import (
	"context"
	"fmt"

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
	parseFlags()
	var a = agent.NewAgentMetrics(int(configData.pollInterval), int(configData.reportInterval), int(configData.rateLimit), configData.secretKey, configData.CryptoKey)
	a.Run(context.Background(), configData.address.String())
}

func buildValue(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}

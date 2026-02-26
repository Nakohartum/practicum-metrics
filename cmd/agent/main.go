package main

import (
	"context"

	"github.com/Nakohartum/practicum-metrics/internal/agent"
)

func main() {
	parseFlags()
	var a = agent.NewAgentMetrics(int(configData.pollInterval), int(configData.reportInterval))
	a.Run(context.Background(), configData.address.String())
}
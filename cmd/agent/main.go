package main

import (
	"github.com/Nakohartum/practicum-metrics/internal/agent"
)

func main() {
	parseFlags()
	var a = agent.NewAgentMetrics(int(pollInterval), int(reportInterval))
	a.Run(address.String())
}

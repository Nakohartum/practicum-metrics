package main

import "github.com/Nakohartum/practicum-metrics/internal/agent"

func main() {
	var a = agent.NewAgentMetrics(2, 10)
	a.Run()
}

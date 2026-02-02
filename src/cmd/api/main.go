package main

import (
	"log"

	"b2b-diagnostic-aggregator/apis/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

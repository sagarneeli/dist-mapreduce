package main

import (
	"os"

	"github.com/sagarneeli/dist-mapreduce/internal/worker"
)

func main() {
	coordinatorHost := os.Getenv("COORDINATOR_HOST")
	if coordinatorHost == "" {
		coordinatorHost = "localhost"
	}
	worker.Worker(coordinatorHost)
}

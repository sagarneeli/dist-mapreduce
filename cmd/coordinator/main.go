package main

import (
	"fmt"
	"os"


	"github.com/sagarneeli/dist-mapreduce/internal/api"
	"github.com/sagarneeli/dist-mapreduce/internal/coordinator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: coordinator <file1> <file2> ...")
		// For docker demo, we can default to looking into a data directory
	}

	// Simple simulation of reading input files from a directory or args
	// In a real scenario, we might scan a directory or use standard input
	files := os.Args[1:]
	if len(files) == 0 {
		// Default to reading from the mounted data directory in Docker
		files = []string{"/app/data/input/test1.txt", "/app/data/input/test2.txt"}
	}

	c := coordinator.NewCoordinator()
	c.Start()

	// Submit the initial job from command line args
	c.SubmitJob(files, 10)

	// Start REST API
	apiServer := api.NewServer(c)
	go func() {
		if err := apiServer.Start("8080"); err != nil {
			fmt.Printf("API Server failed: %v\n", err)
		}
	}()

	// Keep main thread alive
	select {}
}

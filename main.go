package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
)

func run(args ...string) error {
	cmd := exec.Command("terraform", args...)

	// Create pipes for capturing stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "cmd.StdoutPipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "cmd.StderrPipe")
	}

	// Create a WaitGroup to wait for goroutines to finish
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println(m)
		}
	}()

	// Start the Terraform command
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "cmd.Start")
	}

	// Capture and process stdout
	if err := runPrettier(bufio.NewScanner(stdout)); err != nil {
		return errors.Wrap(err, "runPrettier")
	}

	// Wait for the Terraform command to finish and wait for the stderr goroutine to finish
	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "cmd.Wait")
	}

	wg.Wait() // Wait for the stderr goroutine to finish

	return nil
}

func main() {
	if err := run(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}

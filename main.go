package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func run() error {
	cmd := exec.Command("terraform", os.Args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "cmd.StdoutPipe")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "cmd.StderrPipe")
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println(m)
		}
	}()

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "cmd.Start")
	}

	scanner := bufio.NewScanner(stdout)
	if err := runPrettier(scanner); err != nil {
		return errors.Wrap(err, "runPrettier")
	}

	return cmd.Wait()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

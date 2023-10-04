package main

import (
	"bufio"
	_ "embed"
	"strings"
	"testing"
)

//go:embed tests/up.txt
var testUp string

//go:embed tests/down.txt
var testDown string

//go:embed tests/ebs-up.txt
var testEbsUp string

//go:embed tests/ebs-down.txt
var testEbsDown string

var tests = []string{
	testUp,
	testDown,
	testEbsUp,
	testEbsDown,
}

func TestMain(t *testing.T) {
	for _, test := range tests {
		reader := strings.NewReader(test)
		scanner := bufio.NewScanner(reader)
		err := runPrettier(scanner)
		if err != nil {
			t.Fatal(err)
		}
	}
}

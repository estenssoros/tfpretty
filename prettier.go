package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrDone             = errors.New("done")
	ErrResourceNotFound = errors.New("resource not found")
)

type Prettier struct {
	Resources map[string]*Status
}

func (p Prettier) String() string {
	data := map[string][]*Status{}

	for _, v := range p.Resources {
		data[v.Op] = append(data[v.Op], v)
	}

	order := []string{}

	for k, list := range data {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Done && !list[j].Done {
				return true
			}
			if !list[i].Done && list[j].Done {
				return false
			}
			if list[i].Started && !list[j].Started {
				return true
			}
			if !list[i].Started && list[j].Started {
				return false
			}
			return list[i].Resource < list[j].Resource
		})
		data[k] = list
		order = append(order, k)
	}
	sort.Strings(order)

	var b strings.Builder
	for _, k := range order {
		list := data[k]
		done, inProgress, notStarted := statusStats(list)
		b.WriteString(
			fmt.Sprintf(
				"\033[1m%s (%d) done: (%d) in progress: (%d) not started: (%d)\033[0m\n",
				k,
				len(list),
				done,
				inProgress,
				notStarted,
			))
		for _, status := range list {
			b.WriteString(fmt.Sprintf("  %s%s%s\n", status.symbol(), status.Resource, status.dur()))
		}
	}

	return b.String()
}

func statusStats(list []*Status) (int, int, int) {
	var done, inProgress, notStarted int
	for _, status := range list {
		if status.Done {
			done++
		} else if status.Started {
			inProgress++
		} else {
			notStarted++
		}
	}
	return done, inProgress, notStarted
}

func (p *Prettier) Watch(scanner *bufio.Scanner) error {
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if err := p.Process(line); err != nil {
			if errors.Is(err, ErrDone) {
				return nil
			}
			return errors.Wrap(err, "p.Process")
		}
		time.Sleep(100 * time.Millisecond)
		clearScreen()
		fmt.Println(p)
	}
	return nil
}

func (p *Prettier) Process(line string) error {
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, "Apply complete!") || strings.HasPrefix(line, "Destroy complete!") {
		return ErrDone
	}
	resource, after, found := strings.Cut(line, ":")
	if !found {
		return errors.Errorf("bad line: [%s]", line)
	}

	status, ok := p.Resources[resource]
	if !ok {
		return errors.Wrapf(ErrResourceNotFound, "[%s]", resource)
	}
	if err := status.Update(strings.TrimSpace(after)); err != nil {
		return errors.Wrapf(err, "status.Update: [%s]", after)
	}
	return nil
}

var (
	startCode = "Terraform will perform the following actions:"
)

func runPrettier(scanner *bufio.Scanner) error {
	if err := skipStart(scanner); err != nil {
		return errors.Wrap(err, "skipStart")
	}
	resources, err := findResources(scanner)
	if err != nil {
		return errors.Wrap(err, "findResources")
	}
	firstLine, err := skipChangesToOutputs(scanner)
	if err != nil {
		return errors.Wrap(err, "skipChangesToOutputs")
	}
	p := &Prettier{
		Resources: resources,
	}
	clearScreen()
	fmt.Println(p)
	if err := p.Process(firstLine); err != nil {
		return errors.Wrap(err, "p.Process")
	}
	return p.Watch(scanner)
}

func findResources(scanner *bufio.Scanner) (map[string]*Status, error) {
	resources := map[string]*Status{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Plan:") {
			break
		}
		if !strings.HasPrefix(line, "#") {
			continue
		}
		if line[2] == '(' {
			continue
		}
		resource, action, err := scanResourceLine(line)
		if err != nil {
			return nil, errors.Wrap(err, "scanResourceLine")
		}
		var op string
		switch action {
		case "created":
			op = opCreating
		case "read":
			op = opReading
		case "replaced":
			op = opReplacing
		case "updated":
			op = opUpdating
		case "destroyed":
			op = opDestroying
		default:
			return nil, errors.Errorf("unknown action: [%s]", action)
		}

		resources[resource] = &Status{Resource: resource, Op: op}

	}
	return resources, nil
}

func scanResourceLine(line string) (string, string, error) {
	var (
		resource string
		action   string
	)
	_, err := fmt.Sscanf(line, "# %s will be %s", &resource, &action)
	if err == nil {
		return resource, action, nil
	}
	_, err = fmt.Sscanf(line, "# %s must be %s", &resource, &action)
	if err == nil {
		return resource, action, nil
	}
	return "", "", errors.Errorf("could not find resource and action: [%s]", line)
}

func skipStart(scanner *bufio.Scanner) error {
	for scanner.Scan() {
		m := strings.TrimSpace(scanner.Text())
		if m == startCode {
			return nil
		}
	}
	return errors.New("failed to initialize prettier")
}

func skipChangesToOutputs(scanner *bufio.Scanner) (string, error) {
	var seenChangesToOutput bool
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "Changes to Outputs:" {
			seenChangesToOutput = true
			continue
		}
		if seenChangesToOutput && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) {
			continue
		}
		if seenChangesToOutput {
			return line, nil
		}

	}
	return "", nil
}

func clearScreen() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

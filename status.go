package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrBadUpdate = errors.New("bad update")
)

var (
	opCreating   = "creating"
	opDestroying = "destroying"
	opReplacing  = "replacing"
	opUpdating   = "updating"
	opReading    = "reading"
)

type Status struct {
	Resource string
	Op       string
	Started  bool
	Duration string
	Done     bool
	ID       string
}

func (s Status) symbol() string {
	if s.Done {
		return "✅ "
	}
	if s.Started {
		return "⏳ "
	}
	return "   "
}

func (s *Status) dur() string {
	if s.Done {
		return fmt.Sprintf(": %s", s.Duration)
	}
	if s.Started {
		return fmt.Sprintf(": %s", s.Duration)
	}
	return ""
}

var (
	creationElapsedRe    = regexp.MustCompile(`\[(.*?) elapsed\]`)
	destructionElapsedRe = regexp.MustCompile(`(\w+)\s+elapsed`)
	completedRe          = regexp.MustCompile(`([^\s]+)\s+\[id=([^\]]+)\]`)
	destroyedRe          = regexp.MustCompile(`\b(\w+)\b$`)
)

func (s *Status) Update(update string) error {
	if op, starting := isStartUpdate(update); starting {
		s.Op = op
		s.Started = true
		s.Done = false
		return nil
	}
	switch s.Op {
	case opCreating:
		return errors.Wrap(s.updateCreate(update), "updateCreate")
	case opDestroying:
		return errors.Wrap(s.updateDestroy(update), "updateDestroy")
	case opUpdating:
		return errors.Wrap(s.updateCreate(update), "updateUpdated")
	case opReading:
		return errors.Wrap(s.updateCreate(update), "updateReading")
	}
	return errors.Errorf("unknown op: %s", s.Op)
}

func (s *Status) updateDestroy(update string) error {
	if strings.HasPrefix(update, "Still") {
		match := destructionElapsedRe.FindStringSubmatch(update)
		if len(match) != 2 {
			return errors.Errorf("destructionElapsedRe: [%s]", update)
		}
		dur, err := time.ParseDuration(match[1])
		if err != nil {
			return errors.Wrap(err, "time.ParseDuration")
		}
		_ = dur
		s.Duration = match[1]
		return nil
	}
	if strings.Contains(update, "complete") {
		matches := destroyedRe.FindStringSubmatch(update)
		if len(matches) != 2 {
			return errors.Errorf("destroyedRe: [%s]", update)
		}
		dur, err := time.ParseDuration(matches[1])
		if err != nil {
			return errors.Wrap(err, "time.ParseDuration")
		}
		_ = dur
		s.Duration = matches[1]
		s.Done = true
		return nil
	}
	return errors.Wrap(ErrBadUpdate, update)
}

func (s *Status) updateCreate(update string) error {
	if strings.HasPrefix(update, "Still") {
		matches := creationElapsedRe.FindStringSubmatch(update)
		if len(matches) != 2 {
			return errors.Errorf("creationElapsedRe: [%s]", update)
		}
		dur, err := time.ParseDuration(matches[1])
		if err != nil {
			return errors.Wrap(err, "time.ParseDuration")
		}
		_ = dur
		s.Duration = matches[1]
		return nil
	}
	if strings.Contains(update, "complete") {
		matches := completedRe.FindStringSubmatch(update)
		if len(matches) != 3 {
			return errors.Errorf("completedRe: [%s]", update)
		}
		dur, err := time.ParseDuration(matches[1])
		if err != nil {
			return errors.Wrap(err, "time.ParseDuration")
		}
		_ = dur
		s.Duration = matches[1]
		s.ID = matches[2]
		s.Done = true
		return nil
	}

	return errors.Wrap(ErrBadUpdate, update)
}

func isStartUpdate(s string) (string, bool) {
	if s == "Creating..." {
		return opCreating, true
	}
	if strings.HasPrefix(s, "Destroying...") {
		return opDestroying, true
	}
	if strings.HasPrefix(s, "Modifying...") {
		return opUpdating, true
	}
	if s == "Reading..." {
		return opReading, true
	}
	return "", false
}

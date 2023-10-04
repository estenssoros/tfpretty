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
	if s.Done || s.Started {
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
		s.Op, s.Started, s.Done = op, true, false
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

func (s *Status) genericUpdate(update string, elapsedRe, completedRe *regexp.Regexp) error {
	if strings.HasPrefix(update, "Still") {
		matches := elapsedRe.FindStringSubmatch(update)
		if len(matches) < 2 {
			return errors.Errorf("elapsedRe: [%s]", update)
		}
		_, err := time.ParseDuration(matches[1])
		if err != nil {
			return errors.Wrap(err, "time.ParseDuration")
		}
		s.Duration = matches[1]
		return nil
	}
	if strings.Contains(update, "complete") {
		matches := completedRe.FindStringSubmatch(update)
		if len(matches) < 2 {
			return errors.Errorf("completedRe: [%s]", update)
		}
		_, err := time.ParseDuration(matches[1])
		if err != nil {
			return errors.Wrap(err, "time.ParseDuration")
		}
		s.Duration = matches[1]
		s.Done = true
		return nil
	}
	return errors.Wrap(ErrBadUpdate, update)
}

func (s *Status) updateDestroy(update string) error {
	return s.genericUpdate(update, destructionElapsedRe, destroyedRe)
}

func (s *Status) updateCreate(update string) error {
	return s.genericUpdate(update, creationElapsedRe, completedRe)
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

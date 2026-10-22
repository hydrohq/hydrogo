// Package cron parses five-field cron expressions ("minute hour day month
// weekday") and answers whether a minute matches. Fields take numbers,
// ranges (1-5), lists (1,15), steps (*/10, 0-30/5) and *; weekdays are 0-6
// (Sunday 0, 7 also means Sunday). Times are matched in UTC. When both the
// day of month and the weekday are restricted, either may match, as in Vixie
// cron.
package cron

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Spec is a parsed expression.
type Spec struct {
	minute, hour, dom, month, dow uint64
	domAny, dowAny                bool
}

var bounds = [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}

// Parse reads a five-field expression.
func Parse(expression string) (Spec, error) {
	fields := strings.Fields(expression)
	if len(fields) != 5 {
		return Spec{}, errors.New("a schedule needs five fields: minute hour day month weekday")
	}
	var masks [5]uint64
	for i, field := range fields {
		mask, err := parseField(field, bounds[i][0], bounds[i][1])
		if err != nil {
			return Spec{}, fmt.Errorf("field %d (%q): %w", i+1, field, err)
		}
		masks[i] = mask
	}
	if masks[4]&(1<<7) != 0 { // 7 is Sunday too
		masks[4] |= 1
	}
	return Spec{minute: masks[0], hour: masks[1], dom: masks[2], month: masks[3], dow: masks[4] &^ (1 << 7),
		domAny: fields[2] == "*" || strings.HasPrefix(fields[2], "*/"), dowAny: fields[4] == "*" || strings.HasPrefix(fields[4], "*/")}, nil
}

func parseField(field string, low, high int) (uint64, error) {
	var mask uint64
	for _, part := range strings.Split(field, ",") {
		rangePart, stepText, hasStep := strings.Cut(part, "/")
		step := 1
		if hasStep {
			parsed, err := strconv.Atoi(stepText)
			if err != nil || parsed < 1 {
				return 0, errors.New("step must be a positive number")
			}
			step = parsed
		}
		start, end := low, high
		if rangePart != "*" {
			from, to, isRange := strings.Cut(rangePart, "-")
			first, err := strconv.Atoi(from)
			if err != nil {
				return 0, errors.New("not a number")
			}
			start, end = first, first
			if isRange {
				if end, err = strconv.Atoi(to); err != nil {
					return 0, errors.New("not a number")
				}
			} else if hasStep {
				end = high
			}
		}
		if start < low || end > high || start > end {
			return 0, fmt.Errorf("must be within %d-%d", low, high)
		}
		for value := start; value <= end; value += step {
			mask |= 1 << uint(value)
		}
	}
	return mask, nil
}

// Matches reports whether the minute containing t (UTC) is selected.
func (s Spec) Matches(t time.Time) bool {
	t = t.UTC()
	if s.minute&(1<<uint(t.Minute())) == 0 || s.hour&(1<<uint(t.Hour())) == 0 || s.month&(1<<uint(t.Month())) == 0 {
		return false
	}
	dom := s.dom&(1<<uint(t.Day())) != 0
	dow := s.dow&(1<<uint(t.Weekday())) != 0
	if !s.domAny && !s.dowAny {
		return dom || dow
	}
	return dom && dow
}

// Package cron parses cron expressions.
//
// This package works with the format:
//   ┌───────────── minute (0 - 59)
//   │ ┌───────────── hour (0 - 23)
//   │ │ ┌───────────── day of the month (1 - 31)
//   │ │ │ ┌───────────── month (1 - 12)
//   │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday)
//   │ │ │ │ │              or
//   │ │ │ │ │              (1 - 7) (Monday to Sunday)
//   │ │ │ │ │
//   │ │ │ │ │
//   * * * * * <command to execute>
//
// Supported values for each field:
// - Unrestricted: * will match every value
// - Digit: e.g 30 will match a single value
// - List: e.g 10,20 minute will match at 10 and 20 (ranges are also supported)
// - Range: e.g 1-5 day of week will match Monday-Friday and not weekends (inclusive)
// - Range with step: e.g */5 minute will match every five minutes
//                    or 0-23/2 for every other hour
//
// Names can also be used for month and day of month fields, the first three
// characters of the day/month case in-sensitive e.g sun. Ranges of lists of
// names are not supported.
package cron

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
)

const (
	minute uint8 = iota
	hour
	dayOfMonth
	month
	dayOfWeek
	command
)

var (
	validRanges = map[uint8][2]uint8{
		minute:     {0, 59},
		hour:       {0, 23},
		dayOfMonth: {1, 31},
		month:      {1, 12},
		dayOfWeek:  {0, 7},
	}

	validNames = map[uint8]map[string]uint8{
		month: {
			"jan": 1,
			"feb": 2,
			"mar": 3,
			"apr": 4,
			"may": 5,
			"jun": 6,
			"jul": 7,
			"aug": 8,
			"sep": 9,
			"oct": 10,
			"nov": 11,
			"dec": 12,
		},
		dayOfWeek: {
			"mon": 1,
			"tue": 2,
			"wed": 3,
			"thu": 4,
			"fri": 5,
			"sat": 6,
			"sun": 7,
		},
	}
)

// Expression is an expanded cron expression.
type Expression struct {
	Minute     []uint8
	Hour       []uint8
	DayOfMonth []uint8
	Month      []uint8
	DayOfWeek  []uint8
	Command    string
}

// Parse cron string s into Expression.
func Parse(s string) (Expression, error) {
	parts := strings.SplitN(s, " ", 6)
	if len(parts) != 6 {
		return Expression{}, errors.New("invalid expression")
	}

	expanded := make([][]uint8, 5)
	for i := uint8(0); i < 5; i++ {
		min, max := validRanges[i][0], validRanges[i][1]
		v, err := expand(parts[i], i, min, max)
		if err != nil {
			return Expression{}, fmt.Errorf("%d: %w", i, err)
		}
		expanded[i] = v
	}

	return Expression{
		Minute:     expanded[minute],
		Hour:       expanded[hour],
		DayOfMonth: expanded[dayOfMonth],
		Month:      expanded[month],
		DayOfWeek:  expanded[dayOfWeek],
		Command:    parts[command],
	}, nil
}

// String returns a pretty printed table of the receiver.
func (e Expression) String() string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 8, 0, '\t', 0)
	join := func(xs []uint8) string {
		var b strings.Builder
		for i, x := range xs {
			if i > 0 {
				fmt.Fprintf(&b, " %d", x)
			} else {
				fmt.Fprintf(&b, "%d", x)
			}

		}
		return b.String()
	}
	fmt.Fprintf(w, "minute\t%v\n", join(e.Minute))
	fmt.Fprintf(w, "hour\t%v\n", join(e.Hour))
	fmt.Fprintf(w, "day of month\t%v\n", join(e.DayOfMonth))
	fmt.Fprintf(w, "month\t%v\n", join(e.Month))
	fmt.Fprintf(w, "day of week\t%v\n", join(e.DayOfWeek))
	fmt.Fprintf(w, "command\t%v\n", e.Command)

	w.Flush()
	return b.String()
}

// expand field into possible values.
func expand(s string, field, min, max uint8) ([]uint8, error) {
	if parts := strings.Split(s, ","); len(parts) > 1 { // list
		var ret []uint8
		for _, ss := range parts {
			exp, err := expand(ss, field, min, max)
			if err != nil {
				return nil, err
			}
			ret = append(ret, exp...)
		}
		return ret, nil
	}

	switch {
	case s == "*":
		return expandAny(field, min, max)
	case strings.HasPrefix(s, "*/"):
		return expandAnyStepRange(s, min, max)
	case strings.ContainsRune(s, '-'):
		return expandRange(s, min, max)
	default:
		return expandSingle(s, field, min, max)
	}
}

func expandAny(field, min, max uint8) ([]uint8, error) {
	if field == dayOfWeek {
		min++ // avoid duplicating sunday as 0 and 7 are supported
	}
	return stepRange(min, max, 1), nil
}

func expandAnyStepRange(s string, min, max uint8) ([]uint8, error) {
	step, err := atoi(s[2:]) // trim "*/"
	if err != nil {
		return nil, fmt.Errorf("invalid step range: %v", s)
	}
	return stepRange(min, max, step), nil
}

func expandRange(s string, min, max uint8) ([]uint8, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range: %v", s)
	}

	start, err := atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid range start: %v", parts[0])
	}

	endStr, step := parts[1], uint8(1)
	if strings.Contains(endStr, "/") { // range with step
		parts := strings.Split(endStr, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid step range: %v", s)
		}
		endStr = parts[0]
		step, err = atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid step range: %v", s)
		}
	}
	end, err := atoi(endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid range end: %v", endStr)
	}

	if start > end {
		return nil, fmt.Errorf("invalid range %d > %d", start, end)
	}
	if (min > start || start > max) || (min > end || end > max) {
		return nil, fmt.Errorf("outside of range: %d-%d", min, max)
	}
	return stepRange(start, end, step), nil
}

func expandSingle(s string, field, min, max uint8) ([]uint8, error) {
	if names, ok := validNames[field]; ok {
		if x, ok := names[strings.ToLower(s)]; ok {
			return []uint8{x}, nil
		}
	}
	if x, err := atoi(s); err == nil {
		if min > x || x > max {
			return nil, fmt.Errorf("outside of range: %d-%d", min, max)
		}
		return []uint8{x}, nil
	}
	return nil, fmt.Errorf("invalid value: %v", s)
}

func stepRange(start, end, step uint8) []uint8 {
	ret := make([]uint8, 0, end-start)
	for x := start; x <= end; x += step {
		ret = append(ret, x)
	}
	return ret
}

func atoi(s string) (uint8, error) {
	x, err := strconv.ParseInt(s, 10, 8)
	return uint8(x), err
}

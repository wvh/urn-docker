// Package interval implements an interval parser for a format that allows daily or weekly repeating events.
package interval

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type weekday int

const (
	_ weekday = iota
	monday
	tuesday
	wednesday
	thursday
	friday
	saturday
	sunday
)

var (
	weekdays   = []string{"", "mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	timeFields = [3]string{"hour", "minute", "second"}
)

// weekdayFromAbbrev translates a 3-letter weekday to a weekday-typed int. Invalid weekday strings return 0.
func weekdayFromAbbrev(s string) weekday {
	if len(s) != 3 {
		return weekday(0)
	}

	for i := range weekdays {
		if s == weekdays[i] {
			return weekday(i)
		}
	}
	return weekday(0)
}

// Interval defines a time interval consisting of a weekday for weekly repeating jobs,
// H, M, S for time of day, and a informational tag. Weeday and Tag are optional.
type Interval struct {
	Weekday weekday
	H       int
	M       int
	S       int
	Tag     string
}

// String gives a string representation of a time interval.
func (ival *Interval) String() string {
	if ival.Tag != "" {
		return fmt.Sprintf("%s@%02d:%02d %s", weekdays[ival.Weekday], ival.H, ival.M, ival.Tag)
	}
	return fmt.Sprintf("%s@%02d:%02d", weekdays[ival.Weekday], ival.H, ival.M)
}

// Parser holds configuration fields for the time spec parser.
type Parser struct {
	re          *regexp.Regexp
	validateTag func(string) bool
}

// NewParser creates a time spec parser. It takes a list of tag strings to validate against.
// Tags are alphanumeric words that can be used to relay a informational message to the process controlled by the time spec.
func NewParser(tags ...string) (*Parser, error) {
	return &Parser{
		re: regexp.MustCompile(`^(?:([a-zA-Z]{3})?@)?([0-9]{1,2}(?::[0-9]{1,2}){1,2})(?:\s([0-9A-Za-z_-]{1,24}))?$`),
		validateTag: func(tag string) bool {
			for i := range tags {
				if tag == tags[i] {
					return true
				}
			}
			return false
		},
	}, nil
}

// Parse takes a time spec and tries to parse it into an Interval.
//
// Time specs have the form "day@time tag". The day of the week is in three-letter form and optional.
// The time of day is in 24 clock format H:M:S; M and S are optional.
// The optional tag consists of alphanumeric characters A-Z and numbers 0-9, dash and underscore.
// Tags are checked against a fixed list by the interval parser; non-existing tags are invalid.
//
// For example, "mon@10" runs a job on Monday at 10:00, and "12:12:12 partial" runs a job at 12:12:12 with tag "partial"
// (assuming that tag is defined).
func (parser *Parser) Parse(str string) (*Interval, error) {
	if !parser.re.MatchString(str) {
		return nil, errors.New("invalid time spec")
	}

	matches := parser.re.FindStringSubmatch(str)
	if len(matches) < 1 {
		return nil, errors.New("no matches in time spec")
	}

	// 4 matches: (full,) weekday, h:m:s, tag
	if len(matches) != 4 {
		return nil, errors.New("invalid time spec: wrong number of matches")
	}

	// handle tag first
	if matches[3] != "" && !parser.validateTag(matches[3]) {
		return nil, errors.New("invalid time spec: unknown tag")
	}

	var wd weekday
	if matches[1] != "" {
		wd = weekdayFromAbbrev(matches[1])
		if wd == 0 {
			return nil, errors.New("invalid time spec: invalid weekday")
		}
	}

	h, m, s, err := parseTime(matches[2])
	if err != nil {
		return nil, errors.New("invalid time spec: " + err.Error())
	}

	return &Interval{
		Weekday: wd,
		H:       h,
		M:       m,
		S:       s,
		Tag:     matches[3],
	}, nil
}

// ParseMany takes a comma-separated list of time specs and calls Parse on each of them.
// Extra spaces will be trimmed. An empty input string is valid and will return an empty list.
func (parser *Parser) ParseMany(many string) ([]*Interval, error) {
	var ivals []*Interval

	if many == "" {
		return nil, nil
	}

	specs := strings.Split(many, ",")
	for i, spec := range specs {
		ival, err := parser.Parse(strings.TrimSpace(spec))
		if err != nil {
			return nil, fmt.Errorf("interval %d: %v", i, err)
		}
		ivals = append(ivals, ival)
	}
	return ivals, nil
}

// parseTime parses a time string with the form H:M:S into integer H, M and S.
// Neither Seconds nor Minutes are mandatory; missing fields will be interpreted as zero.
// Examples of valid times are 5, 5:5, 05:05:05 or even 001:002:003;
// while some of these more outrageous forms could be caught by a preceeding regexp,
// this function does not depend on prior validation.
func parseTime(s string) (int, int, int, error) {
	hms := strings.SplitN(s, ":", 3)
	if hms == nil {
		return 0, 0, 0, errors.New("can't parse time")
	}

	var ihms [3]int

	for i := range hms {
		val, err := strconv.Atoi(hms[i])
		if err != nil {
			return 0, 0, 0, errors.New("not a number for field " + timeFields[i])
		}
		if (i == 0 && (val < 0 || val > 23)) || (i > 0 && (val < 0 || val > 59)) {
			return 0, 0, 0, errors.New("out of range for field " + timeFields[i])
		}
		ihms[i] = val
	}

	return ihms[0], ihms[1], ihms[2], nil
}

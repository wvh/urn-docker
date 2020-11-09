package interval

import (
	"reflect"
	"testing"
)

// This function allows some input values (like 001:002:003) that would nevertheless be caught by a preceding regexp;
//
func TestParseTime(t *testing.T) {
	tests := []struct {
		tstr    string
		h, m, s int
		isError bool
	}{
		{
			tstr: "12",
			h:    12, m: 0, s: 0,
			isError: false,
		},
		{
			tstr: "12:12",
			h:    12, m: 12, s: 0,
			isError: false,
		},
		{
			tstr: "12:12:12",
			h:    12, m: 12, s: 12,
			isError: false,
		},
		{
			tstr: "1:2:3",
			h:    1, m: 2, s: 3,
			isError: false,
		},
		{
			tstr: "0",
			h:    0, m: 0, s: 0,
			isError: false,
		},
		{
			tstr: "001:002:003",
			h:    1, m: 2, s: 3,
			isError: false,
		},
		{
			tstr: "24:0:0",
			h:    0, m: 0, s: 0,
			isError: true,
		},
		{
			tstr: "12:12:60",
			h:    0, m: 0, s: 0,
			isError: true,
		},
		{
			tstr: "12:12:12:12",
			h:    0, m: 0, s: 0,
			isError: true,
		},
		{
			tstr: "12:12:a",
			h:    0, m: 0, s: 0,
			isError: true,
		},
		{
			tstr: "",
			h:    0, m: 0, s: 0,
			isError: true,
		},
		{
			tstr: "abc",
			h:    0, m: 0, s: 0,
			isError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.tstr, func(t *testing.T) {
			h, m, s, err := parseTime(test.tstr)
			//t.Log("actual error:", err)
			if (err != nil) != test.isError {
				t.Errorf("unexpected error result: expected: %v, got: %v", test.isError, err == nil)
			}
			if h != test.h || m != test.m || s != test.s {
				t.Errorf("mismatched time: expected: [%d,%d,%d], got: [%d,%d,%d]",
					test.h, test.m, test.s,
					h, m, s,
				)
			}
		})
	}
}

func TestIntervals(t *testing.T) {
	passTests := map[string]string{
		"12:12":         "@12:12",
		"12:12:12":      "@12:12",
		"mon@23:59":     "mon@23:59",
		"12:34 goodtag": "@12:34 goodtag",
	}
	failTests := map[string]string{
		// these are erroneous
		"mon@:23:59":    "",
		"someday@23:59": "",
		"23:59:":        "",
		// these have invalid tags
		"12:12 badtag":         "",
		"12:12 goodtag badtag": "",
		// invalid utf8
		"12:12 a\xc5z": "",
		"12:12 \x00":   "",
	}

	parser, _ := NewParser("goodtag")

	for in, res := range passTests {
		t.Run(in, func(t *testing.T) {
			ival, err := parser.Parse(in)
			if err != nil {
				t.Errorf("can't parse interval %q: %s", in, err)
				return
			}

			if ival.String() != res {
				t.Errorf("error parsing interval: expected %q, got %q", res, ival.String())
			}
		})
	}

	for in := range failTests {
		t.Run(in, func(t *testing.T) {
			ival, err := parser.Parse(in)
			if err == nil {
				t.Errorf("test should fail for time spec %q, but got result: %s", in, ival.String())
			}
		})
	}
}

func TestParseMany(t *testing.T) {
	passTests := map[string][]*Interval{
		"12:12, 12:12:12": {
			&Interval{
				Weekday: weekday(0),
				H:       12,
				M:       12,
				S:       0,
				Tag:     "",
			},
			&Interval{
				Weekday: weekday(0),
				H:       12,
				M:       12,
				S:       12,
				Tag:     "",
			},
		},
		"mon@23:59, 12:34 goodtag": {
			&Interval{
				Weekday: weekday(1),
				H:       23,
				M:       59,
				S:       0,
				Tag:     "",
			},
			&Interval{
				Weekday: weekday(0),
				H:       12,
				M:       34,
				S:       0,
				Tag:     "goodtag",
			},
		},
		"": nil,
	}
	failTests := map[string]struct{}{
		" ":                          {},
		",":                          {},
		"12:12,":                     {},
		"12:12,, 12:12:12":           {},
		"mon@23:59, a@12:34 goodtag": {},
		",mon@23:59,12:34 goodtag":   {},
	}

	t.Logf("%+v", passTests)
	parser, _ := NewParser("goodtag")

	for in, res := range passTests {
		t.Run(in, func(t *testing.T) {
			ivals, err := parser.ParseMany(in)
			if err != nil {
				t.Errorf("can't parse intervals %q: %s", in, err)
				return
			}

			if len(ivals) != len(res) {
				t.Errorf("wrong amount of intervals: expected %d, got %d", len(res), len(ivals))
			}

			if !reflect.DeepEqual(ivals, res) {
				t.Errorf("intervals are not equal: expected %+v, got %+v", res, ivals)
			}
		})
	}

	for in := range failTests {
		t.Run(in, func(t *testing.T) {
			ivals, err := parser.ParseMany(in)
			t.Log(err)
			if err == nil {
				t.Errorf("test should fail for time specs %q, but passed", in)
			}
			if len(ivals) != 0 {
				t.Errorf("invalid time spec list should return no intervals, expected: %d, got: %d", 0, len(ivals))
			}
		})
	}
}

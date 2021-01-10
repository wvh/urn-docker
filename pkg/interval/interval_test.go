package interval

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const testTimeFormat = "Mon Jan _2 15:04:05"

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
			if err == nil {
				t.Errorf("test should fail for time specs %q, but passed", in)
			}
			if len(ivals) != 0 {
				t.Errorf("invalid time spec list should return no intervals, expected: %d, got: %d", 0, len(ivals))
			}
		})
	}
}

func TestIntervalNext(t *testing.T) {
	// reference time is 2009-11-10 23:00:00 +0000 UTC / 2009-11-11 01:00:00 +0200 EET;
	// 2009-11-10 is a Tuesday.
	// time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)
	nowFunc := func() time.Time { return time.Unix(1257894000, 0) }
	now := nowFunc()
	tests := []struct {
		interval *Interval
		t        time.Time
	}{
		{
			// time is future
			interval: &Interval{
				Weekday: weekday(0),
				H:       12,
				M:       12,
				S:       0,
				Tag:     "",
				now:     nowFunc,
			},
			t: time.Date(2009, 11, 11, 12, 12, 0, 0, now.Location()),
		},
		{
			// time is passed
			interval: &Interval{
				Weekday: weekday(0),
				H:       0,
				M:       15,
				S:       15,
				Tag:     "",
				now:     nowFunc,
			},
			t: time.Date(2009, 11, 12, 0, 15, 15, 0, now.Location()),
		},
		{
			// weekday is future
			interval: &Interval{
				Weekday: weekday(time.Wednesday),
				H:       12,
				M:       12,
				S:       0,
				Tag:     "",
				now:     nowFunc,
			},
			t: time.Date(2009, 11, 11, 12, 12, 0, 0, now.Location()),
		},
		{
			// weekday is past
			interval: &Interval{
				Weekday: weekday(time.Tuesday),
				H:       0,
				M:       15,
				S:       15,
				Tag:     "",
				now:     nowFunc,
			},
			t: time.Date(2009, 11, 17, 0, 15, 15, 0, now.Location()),
		},
		{
			// time is past and weekday is past
			interval: &Interval{
				Weekday: weekday(time.Wednesday),
				H:       0,
				M:       15,
				S:       15,
				Tag:     "",
				now:     nowFunc,
			},
			t: time.Date(2009, 11, 18, 0, 15, 15, 0, now.Location()),
		},
	}

	t.Log("reference time is", now.UTC())

	for _, test := range tests {
		t.Run(test.interval.String(), func(t *testing.T) {
			next := test.interval.Next()
			//t.Logf("%s --> %s", test.interval.String(), next.Format("Mon Jan _2 15:04:05"))
			if !next.Equal(test.t) {
				t.Errorf("wrong next time, want: %v, got: %v", test.t.Format(testTimeFormat), next.Format(testTimeFormat))
			}
		})
	}
}

func TestIntervalsNext(t *testing.T) {
	nowFunc := func() time.Time { return time.Unix(1257894000, 0) }
	now := nowFunc()
	tests := []struct {
		intervals Intervals
		t         time.Time
	}{
		{
			intervals: Intervals{
				// time is future
				&Interval{
					Weekday: weekday(0),
					H:       12,
					M:       12,
					S:       0,
					Tag:     "",
					now:     nowFunc,
				},
				// time is passed
				&Interval{
					Weekday: weekday(0),
					H:       0,
					M:       15,
					S:       15,
					Tag:     "",
					now:     nowFunc,
				},
				// weekday is future
				&Interval{
					Weekday: weekday(time.Wednesday),
					H:       12,
					M:       12,
					S:       0,
					Tag:     "",
					now:     nowFunc,
				},
				// weekday is past
				&Interval{
					Weekday: weekday(time.Tuesday),
					H:       0,
					M:       15,
					S:       15,
					Tag:     "",
					now:     nowFunc,
				},
				// time is past and weekday is past
				&Interval{
					Weekday: weekday(time.Wednesday),
					H:       0,
					M:       15,
					S:       15,
					Tag:     "",
					now:     nowFunc,
				},
			},
			t: time.Date(2009, 11, 11, 12, 12, 0, 0, now.Location()),
		},
		{
			intervals: Intervals{
				// time and weekday are upcoming (Friday, yay)
				&Interval{
					Weekday: weekday(time.Friday),
					H:       16,
					M:       00,
					S:       00,
					Tag:     "",
					now:     nowFunc,
				},
			},
			t: time.Date(2009, 11, 13, 16, 0, 0, 0, now.Location()),
		},
		{
			intervals: Intervals{},
			t:         time.Time{},
		},
	}

	for _, test := range tests {
		//t.Run(test.t.Format(testTimeFormat), func(t *testing.T) {
		t.Run(test.intervals.String(), func(t *testing.T) {
			t.Logf("%+v", test.intervals)
			next := test.intervals.Next()
			if !next.Equal(test.t) {
				t.Errorf("wrong next time, want: %v, got: %v", test.t.Format(testTimeFormat), next.Format(testTimeFormat))
			}
		})
	}
}

// Fastest way to create a string representation from a list... out of curiosity.
func BenchmarkStringer(b *testing.B) {
	var (
		once     sync.Once
		memoised string
	)

	nowFunc := time.Now
	intervals := Intervals{
		// time is future
		&Interval{
			Weekday: weekday(0),
			H:       12,
			M:       12,
			S:       0,
			Tag:     "",
			now:     nowFunc,
		},
		// time is passed
		&Interval{
			Weekday: weekday(0),
			H:       0,
			M:       15,
			S:       15,
			Tag:     "",
			now:     nowFunc,
		},
		// weekday is future
		&Interval{
			Weekday: weekday(time.Wednesday),
			H:       12,
			M:       12,
			S:       0,
			Tag:     "",
			now:     nowFunc,
		},
		// weekday is past
		&Interval{
			Weekday: weekday(time.Tuesday),
			H:       0,
			M:       15,
			S:       15,
			Tag:     "",
			now:     nowFunc,
		},
		// time is past and weekday is past
		&Interval{
			Weekday: weekday(time.Wednesday),
			H:       0,
			M:       15,
			S:       15,
			Tag:     "",
			now:     nowFunc,
		},
	}

	tests := []struct {
		name string
		f    func() string
	}{
		{
			name: "strings.Join",
			f: func() string {
				out := make([]string, len(intervals))
				for i, v := range intervals {
					out[i] = v.String()
				}
				return "[" + strings.Join(out, " ") + "]"
			},
		},
		{
			name: "strings.Join (append)",
			f: func() string {
				out := make([]string, 0, len(intervals))
				for _, v := range intervals {
					out = append(out, v.String())
				}
				return "[" + strings.Join(out, " ") + "]"
			},
		},
		{
			name: "sync.Once",
			f: func() string {
				once.Do(func() {
					out := make([]string, len(intervals))
					for i, v := range intervals {
						out[i] = v.String()
					}
					memoised = "[" + strings.Join(out, " ") + "]"
				})
				return memoised
			},
		},
		{
			name: "cast",
			f: func() string {
				return fmt.Sprintf("%+v", []*Interval(intervals))
			},
		},
		{
			name: "strings.Builder",
			f: func() string {
				var s strings.Builder
				s.WriteByte('[')
				for i, v := range intervals {
					if i > 0 {
						s.WriteByte(' ')
					}
					s.WriteString(v.String())
				}
				s.WriteByte(']')
				return s.String()
			},
		},
	}

	b.Log(intervals)

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = test.f()
			}
		})
	}
}

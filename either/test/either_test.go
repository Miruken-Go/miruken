package test

import (
	"fmt"
	"github.com/miruken-go/miruken/either"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func tryParseDate(
	candidate string,
	layouts   ...string,
) either.Monad[string, time.Time] {
	for _, layout := range layouts {
		if t, err := time.Parse(layout, candidate); err == nil {
			return either.Right(t)
		}
	}
	return either.Left(candidate)
}

func tryParseDuration(
	candidate string,
) either.Monad[string, time.Duration] {
	if d, err := time.ParseDuration(candidate); err == nil {
		return either.Right(d)
	}
	return either.Left(candidate)
}

func daysForward(d time.Duration) either.Monad[string, float64] {
	if d < 0 {
		return either.Left(fmt.Sprintf("negative duration not allowed: %v.", d))
	}
	return either.Right(d.Hours()/24)
}

func nat(d float64) either.Monad[string, int] {
	if d != float64(int64(d)){
		return either.Left(fmt.Sprintf("non-integers not allowed: {%v}.", d))
	}
	if d < 1 {
		return either.Left(fmt.Sprintf("non-positive numbers not allowed: {%v}.", d))
	}
	return either.Right(int(d))
}

func Test_Map(t *testing.T) {
	t.Run("Left", func (t *testing.T) {
		dt     := tryParseDate("ABC", "2006-01-02")
		result := either.Fold(dt,
			func (c string) time.Time { return time.Time{} },
			func (t time.Time) time.Time { panic("unexpected") })
		assert.True(t, result.IsZero())
	})

	t.Run("Right", func (t *testing.T) {
		dt := tryParseDate("2022-07-19", "2006-01-02")
		ts := tryParseDuration("2h")
		var nested = either.Map[string, time.Time](dt, func (t time.Time) either.Monad[string, time.Time]{
			return either.Map[string, time.Duration, time.Time](ts, func (d time.Duration) time.Time {
				return t.Add(d)
			})
		})
		flattened := either.Fold(nested,
			func (c string) either.Monad[string, time.Time] {
				return either.Left[string](c)
			},
			func (e either.Monad[string, time.Time]) either.Monad[string, time.Time] {
				return e
			})

		result := either.Fold(flattened,
			func (c string) time.Time { panic("unexpected")},
			func (t time.Time) time.Time { return t })

		assert.Equal(t, time.Date(2022, 7, 19, 2, 0, 0, 0, time.UTC), result)
	})
}

func Test_FlatMap(t *testing.T) {
	t.Run("Right", func (t *testing.T) {
		dt := tryParseDate("2022-07-19", "2006-01-02")
		ts := tryParseDuration("2h")
		var flattened = either.FlatMap(dt, func (t time.Time) either.Monad[string, time.Time]{
			return either.Map[string, time.Duration, time.Time](ts, func (d time.Duration) time.Time {
				return t.Add(d)
			})
		})
		result := either.Fold(flattened,
			func (c string) time.Time { panic("unexpected")},
			func (t time.Time) time.Time { return t })

		assert.Equal(t, time.Date(2022, 7, 19, 2, 0, 0, 0, time.UTC), result)
	})
}

func Test_Laws(t *testing.T) {
	t.Run("Left Identity", func (t *testing.T) {
		testCases := []string{
			"3h",
			"5h30m40s",
			"foo",
		}
		ret := func(s string) either.Monad[string, string] {
			return either.Right(s)
		}
		h := tryParseDuration
		for _, test := range testCases{
			if either.FlatMap(ret(test), h) != h(test) {
				t.Errorf("left identity failed for %q", test)
			}
		}
	})

	t.Run("Right Identity", func (t *testing.T) {
		testCases := []string{
			"02 Jan 06 15:04 MST",
			"19 Jul 19 04:30 CST",
			"bar",
		}
		f := func(s string) either.Monad[string, time.Time] {
			return tryParseDate(s, time.RFC822)
		}
		ret := func(t time.Time) either.Monad[string, time.Time] {
			return either.Right(t)
		}
		for _, test := range testCases{
			m := f(test)
			if either.FlatMap(m, ret) != m {
				t.Errorf("Right identity failed for %q", test)
			}
		}
	})

	t.Run("Associativity", func (t *testing.T) {
		testCases := []string{
			"2h",
			"-5h30m40s",
			"-4h5m30s",
			"0",
			"foo",
		}
		f := tryParseDuration
		g := daysForward
		h := nat
		for _, test := range testCases{
			m := f(test)
			if either.FlatMap(either.FlatMap(m, g), h) !=
				either.FlatMap(m, func(x time.Duration) either.Monad[string, int] {
					return either.FlatMap(g(x), h)
				}) {
				t.Errorf("Associativity failed for %q", test)
			}
		}
	})
}
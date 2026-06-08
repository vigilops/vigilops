package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultWindow    = 30 * 24 * time.Hour
	defaultListLimit = 25
	maxListLimit     = 1000
	maxStepsLimit    = 1000
)

type listParams struct {
	From   time.Time
	To     time.Time
	Limit  int
	Offset int
}

func parseListParams(r *http.Request) (listParams, error) {
	q := r.URL.Query()
	to, err := parseTimeOr(q.Get("to"), time.Now())
	if err != nil {
		return listParams{}, err
	}
	from, err := parseTimeOr(q.Get("from"), to.Add(-defaultWindow))
	if err != nil {
		return listParams{}, err
	}
	limit, err := parseIntInRange(q.Get("limit"), defaultListLimit, 1, maxListLimit)
	if err != nil {
		return listParams{}, err
	}
	offset, err := parseIntInRange(q.Get("offset"), 0, 0, 1_000_000)
	if err != nil {
		return listParams{}, err
	}
	return listParams{From: from, To: to, Limit: limit, Offset: offset}, nil
}

func parseAtWindow(r *http.Request) (from, to time.Time, err error) {
	q := r.URL.Query()
	at := q.Get("at")
	if at == "" {
		to = time.Now()
		from = to.Add(-defaultWindow)
		return
	}
	t, perr := parseRFC3339Tolerant(at)
	if perr != nil {
		err = perr
		return
	}
	from = t.Add(-time.Second)
	to = t.Add(time.Second)
	return
}

// parseRFC3339Tolerant handles the common case where a caller pastes a
// raw timestamp like "2026-06-08T12:30:00+08:00" into a URL without
// encoding +, which the server then sees as " " after URL-decoding.
func parseRFC3339Tolerant(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if i := strings.Index(s, " "); i > 0 {
		return time.Parse(time.RFC3339Nano, s[:i]+"+"+s[i+1:])
	}
	return time.Parse(time.RFC3339Nano, s)
}

func parseTimeOr(s string, fallback time.Time) (time.Time, error) {
	if s == "" {
		return fallback, nil
	}
	return time.Parse(time.RFC3339Nano, s)
}

func parseIntInRange(s string, def, min, max int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < min {
		n = min
	}
	if n > max {
		n = max
	}
	return n, nil
}

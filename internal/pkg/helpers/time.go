package helpers

import "time"

const DateLayout = "2006-01-02"

func DateString(t time.Time) string {
	return t.Format(DateLayout)
}

func ParseDate(value string) (time.Time, error) {
	return time.Parse(DateLayout, value)
}

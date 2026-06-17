package helpers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Timestamptz wraps a time as a non-null pgtype.Timestamptz.
func Timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// TimestamptzNow returns the current UTC time as a non-null pgtype.Timestamptz.
func TimestamptzNow() pgtype.Timestamptz {
	return Timestamptz(time.Now().UTC())
}

// DateOf wraps a calendar date as a non-null pgtype.Date.
func DateOf(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

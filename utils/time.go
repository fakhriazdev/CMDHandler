package utils

import "time"

func ToSQLDateOnlyString(t time.Time) string {
	return t.Format("2006-01-02")
}

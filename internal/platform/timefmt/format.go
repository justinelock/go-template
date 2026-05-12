package timefmt

import (
	"fmt"
	"time"
)

// DateTimeSecond formats time as yyyy-MM-dd HH:mm:ss.
func DateTimeSecond(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf(
		"%04d-%02d-%02d %02d:%02d:%02d",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
	)
}

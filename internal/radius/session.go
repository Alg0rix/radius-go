package radius

import "time"

// timeNowPtr returns a pointer to the current time.
func timeNowPtr() *time.Time {
	t := time.Now()
	return &t
}
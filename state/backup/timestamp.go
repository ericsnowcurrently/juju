// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"fmt"
	"regexp"
	"time"
)

// DefaultTimestamp returns "now" as a string formatted as
// "YYYYMMDD-hhmmss".
func DefaultTimestamp(now time.Time) string {
	// Unfortunately time.Time.Format() is not smart enough for us.
	Y, M, D := now.Date()
	h, m, s := now.Clock()
	return fmt.Sprintf(TimestampFormat, Y, M, D, h, m, s)
}

// DefaultFilename returns a filename to use for a backup.  The name is
// derived from the current time and date.
func DefaultFilename() string {
	formattedDate := DefaultTimestamp(time.Now().UTC())
	return fmt.Sprintf(FilenameTemplate, formattedDate)
}

// TimestampFromDefaultFilename extracts the timestamp from the filename.
func TimestampFromDefaultFilename(filename string) (time.Time, error) {
	// Unfortunately we can't just use time.Parse().
	re, err := regexp.Compile(`-\d{8}-\d{6}\.`)
	if err != nil {
		return time.Time{}, err
	}
	match := re.FindString(filename)
	match = match[1:len(match)]

	var Y, M, D, h, m, s int
	fmt.Sscanf(match, TimestampFormat, &Y, &M, &D, &h, &m, &s)
	return time.Date(Y, time.Month(M), D, h, m, s, 0, time.UTC), nil
}

package version

// Do not change anything in this file,
// as it is overwritten during each build.

import (
	"fmt"
	"time"
)

// Date represents the date and time the app was built.
var Date = time.Date(2026, 10, 23, 9, 56, 41, 0, time.Local)

// Major is the first segment of the version: 1.2.3.4
var Major int32 = 1

// Minor is the second segment of the version: 1.2.3.4
var Minor int32 = 5

// Build is the third segment of the version: 1.2.3.4
var Build int64 = 261023

// Revision is the fourth segment of the version: 1.2.3.4
var Revision int64 = 956

// ShortString returns the full version: 'major.minor.date.time'.
func ShortString() string {
	return fmt.Sprintf("%d.%02d", Major, Minor)
}

// FullString returns the full version: 'major.minor.date.time'.
func FullString() string {
	return fmt.Sprintf("%d.%02d.%06d.%04d", Major, Minor, Build, Revision)
}

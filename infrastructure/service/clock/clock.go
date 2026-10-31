package clock

import "time"

// System supplies canonical UTC wall-clock instants at millisecond precision.
type System struct{}

// Now returns the current UTC instant truncated to the accepted system precision.
func (System) Now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

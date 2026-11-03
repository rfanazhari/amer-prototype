package interviewvoice

import "errors"

// Shutdown detaches and closes every owned socket after the HTTP owner's request drain.
// It performs no provider I/O and cannot undo durable session records.
func (d *Driver) Shutdown() error {
	d.mu.Lock()
	conns := d.conns
	d.conns = nil
	d.mu.Unlock()
	var result error
	for _, conn := range conns {
		result = errors.Join(result, conn.Close())
	}
	return result
}

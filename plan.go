package fsync

import "time"

// A Plan defines where a file should be stored, fetched from, and how frequently
// it should update
type Plan interface {
	// The remote path to pull the file from
	RemotePath() string
	// The local path to store the file at
	LocalPath() string
	// How frequently to update the file
	UpdateInterval() time.Duration
}

// BasicPlan is a simplistic plan which implements the Plan interface
type BasicPlan struct {
	// Remote is the remote path to pull the file from
	Remote string
	// Local is the local path where the file will be stored
	Local string
	// UpdateEvery is the duration that the file will be re-copied
	UpdateEvery time.Duration
}

// RemotePath implements the Plan interface for BasicPlan
func (b *BasicPlan) RemotePath() string {
	return b.Remote
}

// LocalPath implements the Plan interface for BasicPlan
func (b *BasicPlan) LocalPath() string {
	return b.Local
}

// UpdateInterval implements the Plan interface for BasicPlan
func (b *BasicPlan) UpdateInterval() time.Duration {
	return b.UpdateEvery
}

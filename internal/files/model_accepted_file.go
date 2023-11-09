package files

import "time"

type AcceptedFile struct {
	FileID   string
	ShardKey string

	Hostname string

	AcceptedAt time.Time
	CanceledAt time.Time
}

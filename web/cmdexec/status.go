package cmdexec

import "github.com/caofujiang/winchaos/data"

const (
	Created   = "Created"
	Success   = "Success"
	Running   = "Running"
	Error     = "Error"
	Destroyed = "Destroyed"
	Revoked   = "Revoked"
)

type StatusCommand struct {
	commandType string
	target      string
	action      string
	flag        string
	uid         string
	limit       string
	status      string
	asc         bool
}

// sqlite
var ds data.SourceI

// GetDS returns dataSource
func GetDS() data.SourceI {
	if ds == nil {
		ds = data.GetSource()
	}
	return ds
}

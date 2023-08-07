package build

import "time"

var (
	Commit  = "dev"
	Date    = time.Now().UTC().String()
	Version = "v0.0.0"
)

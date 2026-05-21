package migrations

import _ "embed"

//go:embed 0001_init.sql
var Init0001 string

//go:embed 0002_link_strength.sql
var Init0002 string

//go:embed 0003_snapshot_stats.sql
var Init0003 string

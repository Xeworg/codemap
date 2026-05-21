package cli

import (
	"context"
	"flag"
	"io"

	"codrut/packages/coding-agent/codemap/store"
)

// RunMigrate runs the "migrate" command and returns an exit code.
func RunMigrate(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	_ = fs.Bool("json", false, "Output JSON envelope (default)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "migrate", err.Error(), EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "migrate", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "migrate", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	versionBefore, _ := store.NewMigrationRunner(db.DB).CurrentSchemaVersion(ctx)
	if err := store.NewMigrationRunner(db.DB).Migrate(ctx); err != nil {
		WriteErrorEnvelope(w, "migrate", "migrate: "+err.Error(), EmptyMeta())
		return 1
	}
	versionAfter, _ := store.NewMigrationRunner(db.DB).CurrentSchemaVersion(ctx)

	applied := versionBefore == "none" || versionAfter != versionBefore
	migrateData := MigrateData{
		MigrationsApplied: applied,
		VersionBefore:     versionBefore,
		VersionAfter:      versionAfter,
	}

	envelope := NewEnvelope("migrate", true, migrateData, nil, EmptyMeta())
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

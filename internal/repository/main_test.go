package repository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// testDB is shared across the whole package: one real Postgres container,
// migrated once, tables truncated between tests for isolation. These tests
// exist specifically to exercise the actual SQL (hospital_id scoping, ILIKE
// bilingual matching, blind-index hash equality, unique-constraint errors)
// that mocked repository tests elsewhere in the codebase can't verify.
var testDB *gorm.DB

func TestMain(m *testing.M) {
	// Run in a separate function, not directly in TestMain, so that deferred
	// cleanup (container.Terminate) actually executes — os.Exit called directly
	// in TestMain would skip every defer registered in this function's scope.
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("agnos_test"),
		tcpostgres.WithUsername("agnos_test"),
		tcpostgres.WithPassword("agnos_test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "internal/repository tests require Docker — could not start postgres container:", err)
		return 1
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "get connection string:", err)
		return 1
	}

	// PreferSimpleProtocol lets a single Exec run the whole multi-statement
	// migration file as-is, instead of needing to split it into statements.
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect to test db:", err)
		return 1
	}

	migrationSQL, err := os.ReadFile("../../migrations/000001_init.up.sql")
	if err != nil {
		fmt.Fprintln(os.Stderr, "read migration file:", err)
		return 1
	}
	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		fmt.Fprintln(os.Stderr, "apply migration:", err)
		return 1
	}

	testDB = db
	return m.Run()
}

// truncateAll resets all tables before a test runs, keeping tests independent
// without needing a fresh container per test (which would be far slower).
func truncateAll(t *testing.T) {
	t.Helper()
	if err := testDB.Exec("TRUNCATE TABLE refresh_tokens, patients, staff, hospitals CASCADE").Error; err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

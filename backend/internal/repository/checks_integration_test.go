
package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"url-security-analyzer/backend/internal/database"
	"url-security-analyzer/backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	t.Cleanup(db.Close)

	if err := database.Migrate(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	// Очищаем таблицу только в выделенной тестовой БД.
	if _, err := db.Exec(
		ctx,
		`TRUNCATE TABLE url_checks RESTART IDENTITY`,
	); err != nil {
		t.Fatalf("clear test table: %v", err)
	}

	return db
}

func TestChecksRepositoryCreateAndList(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewChecksRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	input := model.Check{
		URL:            "https://example.com",
		Available:      true,
		StatusCode:     200,
		ResponseTimeMS: 123,
	}

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if created.ID <= 0 {
		t.Fatalf("expected positive ID, got %d", created.ID)
	}
	if created.URL != input.URL {
		t.Errorf("URL = %q, want %q", created.URL, input.URL)
	}
	if !created.Available {
		t.Error("expected Available=true")
	}
	if created.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", created.StatusCode)
	}
	if created.ResponseTimeMS != 123 {
		t.Errorf(
			"ResponseTimeMS = %d, want 123",
			created.ResponseTimeMS,
		)
	}
	if created.CheckedAt.IsZero() {
		t.Error("expected CheckedAt to be populated by PostgreSQL")
	}

	_, err = repo.Create(ctx, model.Check{
		URL:            "https://unavailable.example",
		Available:      false,
		ResponseTimeMS: 50,
		ErrorMessage:   "connection refused",
	})
	if err != nil {
		t.Fatalf("Create() second record error: %v", err)
	}

	checks, err := repo.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(checks) != 2 {
		t.Fatalf("List() returned %d records, want 2", len(checks))
	}

	foundUnavailable := false

	for _, check := range checks {
		if check.URL != "https://unavailable.example" {
			continue
		}

		foundUnavailable = true

		if check.Available {
			t.Error("expected unavailable record")
		}
		if check.StatusCode != 0 {
			t.Errorf(
				"StatusCode = %d, want 0 for NULL status",
				check.StatusCode,
			)
		}
		if check.ErrorMessage != "connection refused" {
			t.Errorf(
				"ErrorMessage = %q, want %q",
				check.ErrorMessage,
				"connection refused",
			)
		}
	}

	if !foundUnavailable {
		t.Error("unavailable record was not found in List()")
	}
}

func TestChecksRepositoryListLimit(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewChecksRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		_, err := repo.Create(ctx, model.Check{
			URL:            "https://example.com",
			Available:      true,
			StatusCode:     200,
			ResponseTimeMS: int64(i),
		})
		if err != nil {
			t.Fatalf("Create() record %d: %v", i, err)
		}
	}

	checks, err := repo.List(ctx, 2)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(checks) != 2 {
		t.Fatalf("List(2) returned %d records, want 2", len(checks))
	}
}

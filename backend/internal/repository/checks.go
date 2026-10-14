package repository

import (
	"context"
	"fmt"

	"url-security-analyzer/backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChecksRepository struct {
	DB *pgxpool.Pool
}

func NewChecksRepository(db *pgxpool.Pool) *ChecksRepository {
	return &ChecksRepository{DB: db}
}

func (r *ChecksRepository) Create(
	ctx context.Context,
	check model.Check,
) (model.Check, error) {
	err := r.DB.QueryRow(ctx, `
		INSERT INTO url_checks (
			url, available, status_code, response_time_ms, error_message
		)
		VALUES ($1, $2, NULLIF($3, 0), $4, NULLIF($5, ''))
		RETURNING id, checked_at
	`,
		check.URL,
		check.Available,
		check.StatusCode,
		check.ResponseTimeMS,
		check.ErrorMessage,
	).Scan(&check.ID, &check.CheckedAt)

	if err != nil {
		return model.Check{}, fmt.Errorf("create URL check: %w", err)
	}

	return check, nil
}

func (r *ChecksRepository) List(
	ctx context.Context,
	limit int,
) ([]model.Check, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT
			id,
			url,
			available,
			COALESCE(status_code, 0),
			response_time_ms,
			COALESCE(error_message, ''),
			checked_at
		FROM url_checks
		ORDER BY checked_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list URL checks: %w", err)
	}
	defer rows.Close()

	checks := make([]model.Check, 0)

	for rows.Next() {
		var check model.Check

		if err := rows.Scan(
			&check.ID,
			&check.URL,
			&check.Available,
			&check.StatusCode,
			&check.ResponseTimeMS,
			&check.ErrorMessage,
			&check.CheckedAt,
		); err != nil {
			return nil, fmt.Errorf("scan URL check: %w", err)
		}

		checks = append(checks, check)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate URL checks: %w", err)
	}

	return checks, nil
}

package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/june20516/orbithall/internal/models"
)

// CreateUser는 새로운 사용자를 생성합니다
// RETURNING 절을 사용하여 생성된 ID와 타임스탬프를 user 포인터에 설정합니다
func CreateUser(ctx context.Context, db DBTX, user *models.User) error {
	query := `
		INSERT INTO users (email, name, picture_url, google_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := db.QueryRowContext(ctx, query,
		user.Email,
		user.Name,
		user.PictureURL,
		user.GoogleID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByGoogleID는 Google ID로 사용자를 조회합니다
// 사용자를 찾지 못한 경우 nil을 반환합니다
func GetUserByGoogleID(ctx context.Context, db DBTX, googleID string) (*models.User, error) {
	query := `
		SELECT id, email, name, picture_url, google_id, created_at, updated_at
		FROM users
		WHERE google_id = $1
	`

	var user models.User
	err := db.QueryRowContext(ctx, query, googleID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PictureURL,
		&user.GoogleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 사용자를 찾지 못한 경우 nil 반환
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by google_id: %w", err)
	}

	return &user, nil
}

// GetUserByEmail은 이메일로 사용자를 조회합니다
// 사용자를 찾지 못한 경우 nil을 반환합니다
func GetUserByEmail(ctx context.Context, db DBTX, email string) (*models.User, error) {
	query := `
		SELECT id, email, name, picture_url, google_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PictureURL,
		&user.GoogleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 사용자를 찾지 못한 경우 nil 반환
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetUserByID는 ID로 사용자를 조회합니다
// 사용자를 찾지 못한 경우 nil을 반환합니다
func GetUserByID(ctx context.Context, db DBTX, id int64) (*models.User, error) {
	query := `
		SELECT id, email, name, picture_url, google_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PictureURL,
		&user.GoogleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 사용자를 찾지 못한 경우 nil 반환
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

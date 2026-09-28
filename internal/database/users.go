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

// GetOrCreateUserByGoogleID는 Google ID로 사용자를 조회하고, 없으면 user 정보로 생성합니다
// 이미 있는 사용자는 user 정보로 갱신하지 않고 저장된 값 그대로 반환합니다
//
// 같은 Google ID로 첫 로그인이 동시에 들어와도 UNIQUE 제약 위반이 나지 않도록 ON CONFLICT DO NOTHING + 재조회 패턴을 씁니다
// (먼저 INSERT한 트랜잭션이 끝날 때까지 다른 트랜잭션의 INSERT가 기다린 뒤, 충돌이면 아무것도 하지 않습니다)
// 다른 Google ID의 사용자가 이미 같은 이메일을 쓰고 있으면 INSERT가 무시되어 재조회 결과가 없으므로 ErrEmailTaken을 반환합니다
func GetOrCreateUserByGoogleID(ctx context.Context, db DBTX, user *models.User) (*models.User, error) {
	// 1단계: INSERT 시도 (google_id나 email이 이미 있으면 무시)
	// users에는 google_id 외에 email에도 UNIQUE 제약이 있습니다
	// ON CONFLICT (google_id)처럼 대상을 지정하면 google_id 충돌만 무시하는데,
	// 같은 사용자의 INSERT가 동시에 들어오면 email 인덱스에서 먼저 충돌이 검사되는 경우가 있어 unique_violation 에러가 납니다
	// 그래서 대상을 지정하지 않아 모든 UNIQUE 제약의 충돌을 무시하고, 결과는 2단계 재조회로 판단합니다
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (email, name, picture_url, google_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, user.Email, user.Name, user.PictureURL, user.GoogleID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	// 2단계: 반드시 Google ID로 재조회 (INSERT가 성공했거나 같은 Google ID 사용자가 있었다면 이 시점에는 존재함)
	stored, err := GetUserByGoogleID(ctx, db, user.GoogleID)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		// Google ID 사용자가 없는데 INSERT가 무시되었다면 email 충돌, 즉 다른 Google ID가 이 이메일을 쓰고 있는 경우입니다
		return nil, ErrEmailTaken
	}

	return stored, nil
}

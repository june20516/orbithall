package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/june20516/orbithall/internal/models"
)

// GetPostBySlug는 사이트 ID와 slug로 포스트를 조회합니다
// site_id와 slug 조합은 유니크하므로 정확히 하나의 포스트를 반환합니다
func GetPostBySlug(ctx context.Context, db DBTX, siteID int64, slug string) (*models.Post, error) {
	query := `
		SELECT id, site_id, slug, title, comment_count, created_at, updated_at
		FROM posts
		WHERE site_id = $1 AND slug = $2
	`

	var post models.Post
	err := db.QueryRowContext(ctx, query, siteID, slug).Scan(
		&post.ID,
		&post.SiteID,
		&post.Slug,
		&post.Title,
		&post.CommentCount,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 포스트를 찾지 못한 경우 nil 반환
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post by slug: %w", err)
	}

	return &post, nil
}

// GetPostByID는 ID로 포스트를 조회합니다
func GetPostByID(ctx context.Context, db DBTX, id int64) (*models.Post, error) {
	query := `
		SELECT id, site_id, slug, title, comment_count, created_at, updated_at
		FROM posts
		WHERE id = $1
	`

	var post models.Post
	err := db.QueryRowContext(ctx, query, id).Scan(
		&post.ID,
		&post.SiteID,
		&post.Slug,
		&post.Title,
		&post.CommentCount,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 포스트를 찾지 못한 경우 nil 반환
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post by id: %w", err)
	}

	return &post, nil
}

// GetOrCreatePost는 포스트를 조회하고, 없으면 생성합니다
// Next.js 블로그에는 존재하지만 DB에는 없는 포스트를 처음 댓글 작성 시 자동 생성합니다
// Race condition 방지를 위해 ON CONFLICT DO NOTHING + 재조회 패턴 사용
func GetOrCreatePost(ctx context.Context, db DBTX, siteID int64, slug, title string) (*models.Post, error) {
	// 1단계: INSERT 시도 (중복 시 무시)
	// 동시에 여러 요청이 들어와도 unique constraint에 의해 하나만 생성됨
	_, err := db.ExecContext(ctx, `
		INSERT INTO posts (site_id, slug, title, comment_count)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (site_id, slug) DO NOTHING
	`, siteID, slug, title)

	if err != nil {
		return nil, fmt.Errorf("failed to insert post: %w", err)
	}

	// 2단계: 반드시 재조회 (INSERT가 성공했든 충돌했든 확실히 존재함)
	post, err := GetPostBySlug(ctx, db, siteID, slug)
	if err != nil {
		return nil, err
	}

	// 이 시점에서 post는 항상 존재해야 함
	if post == nil {
		return nil, fmt.Errorf("post should exist after insert but not found")
	}

	return post, nil
}

// IncrementCommentCount는 포스트의 댓글 수를 1 증가시킵니다
func IncrementCommentCount(ctx context.Context, db DBTX, postID int64) error {
	query := `
		UPDATE posts
		SET comment_count = comment_count + 1,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, postID)
	if err != nil {
		return fmt.Errorf("failed to increment comment count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// DecrementCommentCount는 포스트의 댓글 수를 1 감소시킵니다
// 댓글 삭제 시 사용하며, 0 이하로 내려가지 않도록 제한합니다
func DecrementCommentCount(ctx context.Context, db DBTX, postID int64) error {
	query := `
		UPDATE posts
		SET comment_count = GREATEST(comment_count - 1, 0),
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, postID)
	if err != nil {
		return fmt.Errorf("failed to decrement comment count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

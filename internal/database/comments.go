package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/june20516/orbithall/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// scanComment는 데이터베이스 row를 Comment 모델로 변환합니다
// database/sql의 Scan 메서드를 활용하여 12개 필드를 매핑합니다
func scanComment(row *sql.Row) (*models.Comment, error) {
	var comment models.Comment
	err := row.Scan(
		&comment.ID,
		&comment.PostID,
		&comment.ParentID,
		&comment.AuthorName,
		&comment.AuthorPassword,
		&comment.Content,
		&comment.IPAddress,
		&comment.UserAgent,
		&comment.IsDeleted,
		&comment.CreatedAt,
		&comment.UpdatedAt,
		&comment.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// CreateComment는 새로운 댓글을 생성합니다
// 비밀번호는 bcrypt로 해싱하여 저장하고, 대댓글의 depth를 검증합니다 (1 depth만 허용)
func CreateComment(ctx context.Context, db DBTX, postID int64, parentID *int64, authorName, password, content, ipAddress, userAgent string) (*models.Comment, error) {
	// 1단계: 부모 댓글이 있으면 depth 검증 (2depth 금지)
	if parentID != nil {
		var parentParentID sql.NullInt64
		err := db.QueryRowContext(ctx, `
			SELECT parent_id
			FROM comments
			WHERE id = $1
		`, *parentID).Scan(&parentParentID)

		if err == sql.ErrNoRows {
			return nil, ErrParentCommentNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query parent comment: %w", err)
		}

		// 부모 댓글이 이미 대댓글이면 (parent_id가 null이 아니면) 2depth이므로 거부
		if parentParentID.Valid {
			return nil, ErrNestedReplyNotAllowed
		}
	}

	// 2단계: 비밀번호 해싱 (bcrypt cost 12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 3단계: 댓글 INSERT 및 RETURNING으로 생성된 레코드 조회
	query := `
		INSERT INTO comments (post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE)
		RETURNING id, post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at, deleted_at
	`

	row := db.QueryRowContext(ctx, query, postID, parentID, authorName, string(hashedPassword), content, ipAddress, userAgent)
	comment, err := scanComment(row)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return comment, nil
}

// GetCommentByID는 ID로 댓글을 조회합니다
// 삭제된 댓글(is_deleted=true)도 조회됩니다
func GetCommentByID(ctx context.Context, db DBTX, commentID int64) (*models.Comment, error) {
	query := `
		SELECT id, post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at, deleted_at
		FROM comments
		WHERE id = $1
	`

	row := db.QueryRowContext(ctx, query, commentID)
	comment, err := scanComment(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get comment by id: %w", err)
	}

	return comment, nil
}

// UpdateComment는 댓글의 content, ip_address, user_agent를 수정합니다
// 삭제된 댓글(is_deleted=true)은 수정할 수 없습니다
func UpdateComment(ctx context.Context, db DBTX, commentID int64, content, ipAddress, userAgent string) error {
	query := `
		UPDATE comments
		SET content = $1,
			ip_address = $2,
			user_agent = $3,
			updated_at = CLOCK_TIMESTAMP()
		WHERE id = $4 AND is_deleted = FALSE
	`

	result, err := db.ExecContext(ctx, query, content, ipAddress, userAgent, commentID)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found or already deleted")
	}

	return nil
}

// DeleteComment는 댓글을 soft delete 처리합니다
// is_deleted를 TRUE로 설정하고 deleted_at에 현재 시각을 기록합니다
// 이미 삭제된 댓글은 다시 삭제할 수 없습니다
func DeleteComment(ctx context.Context, db DBTX, commentID int64) error {
	query := `
		UPDATE comments
		SET is_deleted = TRUE,
			deleted_at = CLOCK_TIMESTAMP()
		WHERE id = $1 AND is_deleted = FALSE
	`

	result, err := db.ExecContext(ctx, query, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found or already deleted")
	}

	return nil
}

// ListComments는 포스트의 댓글 목록을 2-level 계층 구조로 조회합니다
// 최상위 댓글(parent_id IS NULL)을 페이지네이션하고, 각 댓글의 대댓글을 함께 조회합니다
// created_at ASC 순으로 정렬됩니다
func ListComments(ctx context.Context, db DBTX, postID int64, limit, offset int) ([]*models.Comment, int, error) {
	// 1단계: 최상위 댓글 총 개수 조회
	var total int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM comments
		WHERE post_id = $1 AND parent_id IS NULL
	`, postID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments: %w", err)
	}

	// 2단계: 최상위 댓글 조회 (페이지네이션)
	query := `
		SELECT id, post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at, deleted_at
		FROM comments
		WHERE post_id = $1 AND parent_id IS NULL
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.ParentID,
			&comment.AuthorName,
			&comment.AuthorPassword,
			&comment.Content,
			&comment.IPAddress,
			&comment.UserAgent,
			&comment.IsDeleted,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&comment.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	// 3단계: 각 최상위 댓글의 대댓글 조회
	for _, comment := range comments {
		replies, err := getReplies(ctx, db, comment.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get replies for comment %d: %w", comment.ID, err)
		}
		comment.Replies = replies
	}

	return comments, total, nil
}

// getReplies는 특정 댓글의 대댓글 목록을 조회합니다 (비공개 헬퍼 함수)
// created_at ASC, id ASC 순으로 정렬됩니다
func getReplies(ctx context.Context, db DBTX, parentID int64) ([]*models.Comment, error) {
	query := `
		SELECT id, post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at, deleted_at
		FROM comments
		WHERE parent_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query replies: %w", err)
	}
	defer rows.Close()

	var replies []*models.Comment
	for rows.Next() {
		var reply models.Comment
		err := rows.Scan(
			&reply.ID,
			&reply.PostID,
			&reply.ParentID,
			&reply.AuthorName,
			&reply.AuthorPassword,
			&reply.Content,
			&reply.IPAddress,
			&reply.UserAgent,
			&reply.IsDeleted,
			&reply.CreatedAt,
			&reply.UpdatedAt,
			&reply.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reply: %w", err)
		}
		replies = append(replies, &reply)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return replies, nil
}

// GetAdminComments는 Admin용 댓글 조회 함수입니다
// 일반 ListComments와 달리 삭제된 댓글도 포함하며, IP 마스킹을 하지 않습니다
func GetAdminComments(ctx context.Context, db DBTX, postID int64, limit, offset int) ([]*models.Comment, int, error) {
	// 1단계: 최상위 댓글 총 개수 조회 (삭제된 것 포함)
	var total int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM comments
		WHERE post_id = $1 AND parent_id IS NULL
	`, postID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments: %w", err)
	}

	// 2단계: 최상위 댓글 조회 (페이지네이션, 삭제된 것 포함)
	query := `
		SELECT id, post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at, deleted_at
		FROM comments
		WHERE post_id = $1 AND parent_id IS NULL
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.ParentID,
			&comment.AuthorName,
			&comment.AuthorPassword,
			&comment.Content,
			&comment.IPAddress, // Admin은 전체 IP 확인 가능
			&comment.UserAgent,
			&comment.IsDeleted,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&comment.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	// 3단계: 각 최상위 댓글의 대댓글 조회 (삭제된 것 포함)
	for _, comment := range comments {
		replies, err := getAdminReplies(ctx, db, comment.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get replies for comment %d: %w", comment.ID, err)
		}
		comment.Replies = replies
	}

	return comments, total, nil
}

// getAdminReplies는 Admin용 대댓글 조회 (삭제된 것 포함, IP 마스킹 없음)
func getAdminReplies(ctx context.Context, db DBTX, parentID int64) ([]*models.Comment, error) {
	query := `
		SELECT id, post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at, deleted_at
		FROM comments
		WHERE parent_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query replies: %w", err)
	}
	defer rows.Close()

	var replies []*models.Comment
	for rows.Next() {
		var reply models.Comment
		err := rows.Scan(
			&reply.ID,
			&reply.PostID,
			&reply.ParentID,
			&reply.AuthorName,
			&reply.AuthorPassword,
			&reply.Content,
			&reply.IPAddress, // Admin은 전체 IP 확인 가능
			&reply.UserAgent,
			&reply.IsDeleted,
			&reply.CreatedAt,
			&reply.UpdatedAt,
			&reply.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reply: %w", err)
		}
		replies = append(replies, &reply)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return replies, nil
}


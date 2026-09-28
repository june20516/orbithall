package database

import (
	"context"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// refreshTokenTestTime은 저장소 테스트의 기준 시각입니다
var refreshTokenTestTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// createRefreshTokenTestUser는 Refresh Token 테스트용 사용자를 만듭니다
func createRefreshTokenTestUser(ctx context.Context, t *testing.T, tx DBTX) *models.User {
	t.Helper()
	user := &models.User{
		Email:    "refresh-token@example.com",
		Name:     "Refresh Token User",
		GoogleID: "google-refresh-token-user",
	}
	if err := CreateUser(ctx, tx, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// createTestRefreshTokenFamily는 hash 값으로 새 계열의 첫 토큰을 저장합니다
func createTestRefreshTokenFamily(ctx context.Context, t *testing.T, tx DBTX, userID int64, hash string, familyExpiresAt time.Time) *models.RefreshToken {
	t.Helper()
	token, err := CreateRefreshTokenFamily(ctx, tx, userID, []byte(hash), familyExpiresAt, familyExpiresAt)
	if err != nil {
		t.Fatalf("failed to create refresh token family: %v", err)
	}
	return token
}

// TestCreateRefreshTokenFamily는 로그인 시 첫 토큰 저장을 테스트합니다
func TestCreateRefreshTokenFamily(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("새 계열의 첫 토큰을 저장한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 만료 시각
		user := createRefreshTokenTestUser(ctx, t, tx)
		expiresAt := refreshTokenTestTime.Add(14 * 24 * time.Hour)
		familyExpiresAt := refreshTokenTestTime.Add(30 * 24 * time.Hour)

		// When: 저장
		token, err := CreateRefreshTokenFamily(ctx, tx, user.ID, []byte("hash-create"), expiresAt, familyExpiresAt)

		// Then: 계열 ID가 생기고 부모·사용·폐기 정보는 비어 있음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token.ID == 0 || token.FamilyID == "" {
			t.Errorf("expected ID and FamilyID, got %+v", token)
		}
		if token.UserID != user.ID {
			t.Errorf("UserID = %d, want %d", token.UserID, user.ID)
		}
		if token.ParentID != nil || token.UsedAt != nil || token.RevokedAt != nil {
			t.Errorf("expected nil parent/used/revoked, got %+v", token)
		}
		if !token.ExpiresAt.Equal(expiresAt) || !token.FamilyExpiresAt.Equal(familyExpiresAt) {
			t.Errorf("unexpected expiry: %v / %v", token.ExpiresAt, token.FamilyExpiresAt)
		}
	})

	t.Run("로그인마다 다른 계열 ID를 가진다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자
		user := createRefreshTokenTestUser(ctx, t, tx)
		familyExpiresAt := refreshTokenTestTime.Add(time.Hour)

		// When: 두 번 로그인
		first := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-family-1", familyExpiresAt)
		second := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-family-2", familyExpiresAt)

		// Then: 계열 ID가 다름
		if first.FamilyID == second.FamilyID {
			t.Error("expected different family IDs")
		}
	})
}

// TestGetRefreshTokenByHash는 해시로 토큰 조회를 테스트합니다
func TestGetRefreshTokenByHash(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("저장된 토큰을 찾는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 저장된 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		created := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-get", refreshTokenTestTime.Add(time.Hour))

		// When: 해시로 조회
		found, err := GetRefreshTokenByHash(ctx, tx, []byte("hash-get"))

		// Then: 같은 토큰
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if found == nil || found.ID != created.ID {
			t.Errorf("expected token %d, got %+v", created.ID, found)
		}
	})

	t.Run("없으면 nil을 반환한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 없는 해시로 조회
		found, err := GetRefreshTokenByHash(ctx, tx, []byte("hash-missing"))

		// Then: nil, nil
		if err != nil || found != nil {
			t.Errorf("expected nil, nil; got %+v, %v", found, err)
		}
	})
}

// TestRotateRefreshToken은 회전(이전 토큰 사용 처리 + 후속 토큰 저장)을 테스트합니다
func TestRotateRefreshToken(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("이전 토큰을 사용 처리하고 후속 토큰을 저장한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 첫 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		familyExpiresAt := refreshTokenTestTime.Add(30 * 24 * time.Hour)
		parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-parent", familyExpiresAt)
		childExpiresAt := refreshTokenTestTime.Add(14 * 24 * time.Hour)

		// When: 회전
		child, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-child"), childExpiresAt, refreshTokenTestTime)

		// Then: 후속 토큰이 같은 계열·절대 만료를 이어받고 부모를 가리킴
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if child == nil {
			t.Fatal("expected child token, got nil")
		}
		if child.ParentID == nil || *child.ParentID != parent.ID {
			t.Errorf("ParentID = %v, want %d", child.ParentID, parent.ID)
		}
		if child.FamilyID != parent.FamilyID || !child.FamilyExpiresAt.Equal(familyExpiresAt) {
			t.Errorf("expected same family, got %+v", child)
		}
		if !child.ExpiresAt.Equal(childExpiresAt) {
			t.Errorf("ExpiresAt = %v, want %v", child.ExpiresAt, childExpiresAt)
		}

		// Then: 부모는 사용 처리됨
		reloaded, _ := GetRefreshTokenByHash(ctx, tx, []byte("hash-parent"))
		if reloaded.UsedAt == nil || !reloaded.UsedAt.Equal(refreshTokenTestTime) {
			t.Errorf("parent UsedAt = %v, want %v", reloaded.UsedAt, refreshTokenTestTime)
		}
	})

	t.Run("이미 사용된 토큰은 다시 회전되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 번 회전한 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-used", refreshTokenTestTime.Add(time.Hour))
		if _, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-used-child"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime); err != nil {
			t.Fatalf("first rotation failed: %v", err)
		}

		// When: 같은 부모로 다시 회전
		second, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-used-child-2"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime)

		// Then: nil, nil
		if err != nil || second != nil {
			t.Errorf("expected nil, nil; got %+v, %v", second, err)
		}
	})

	t.Run("폐기된 토큰은 회전되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 폐기된 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-revoked", refreshTokenTestTime.Add(time.Hour))
		if err := RevokeRefreshTokenFamily(ctx, tx, parent.FamilyID, models.RefreshTokenRevokedByLogout, refreshTokenTestTime); err != nil {
			t.Fatalf("failed to revoke: %v", err)
		}

		// When: 회전
		child, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-revoked-child"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime)

		// Then: nil, nil
		if err != nil || child != nil {
			t.Errorf("expected nil, nil; got %+v, %v", child, err)
		}
	})
}

// TestRotateRefreshToken_FamilyRevoked는 계열 폐기를 빠져나간 토큰이 회전되지 않는지 테스트합니다
// 폐기와 동시에 진행된 회전이 만든 후속 토큰은 revoked_at이 비어 있을 수 있으므로, 같은 계열의 폐기 여부로 막아야 합니다
func TestRotateRefreshToken_FamilyRevoked(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 폐기된 계열과, 같은 계열에서 폐기되지 않고 남은 토큰(동시 회전으로 폐기를 빠져나간 상황)
	user := createRefreshTokenTestUser(ctx, t, tx)
	familyExpiresAt := refreshTokenTestTime.Add(time.Hour)
	first := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-family-first", familyExpiresAt)
	if err := RevokeRefreshTokenFamily(ctx, tx, first.FamilyID, models.RefreshTokenRevokedByReuse, refreshTokenTestTime); err != nil {
		t.Fatalf("failed to revoke: %v", err)
	}
	var escapedID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO refresh_tokens (user_id, family_id, parent_id, token_hash, expires_at, family_expires_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id
	`, user.ID, first.FamilyID, first.ID, []byte("hash-escaped"), familyExpiresAt).Scan(&escapedID)
	if err != nil {
		t.Fatalf("failed to insert escaped token: %v", err)
	}

	// When: 남은 토큰으로 회전
	child, err := RotateRefreshToken(ctx, tx, escapedID, []byte("hash-escaped-child"), familyExpiresAt, refreshTokenTestTime)

	// Then: nil, nil이고 남은 토큰은 사용 처리되지 않음
	if err != nil || child != nil {
		t.Errorf("expected nil, nil; got %+v, %v", child, err)
	}
	escaped, _ := GetRefreshTokenByHash(ctx, tx, []byte("hash-escaped"))
	if escaped.UsedAt != nil {
		t.Errorf("expected escaped token to stay unused, got UsedAt %v", escaped.UsedAt)
	}
}

// TestGetChildRefreshToken은 후속 토큰 조회를 테스트합니다
func TestGetChildRefreshToken(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 회전하지 않은 토큰과 회전한 토큰
	user := createRefreshTokenTestUser(ctx, t, tx)
	lonely := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-lonely", refreshTokenTestTime.Add(time.Hour))
	parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-with-child", refreshTokenTestTime.Add(time.Hour))
	child, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-the-child"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime)
	if err != nil {
		t.Fatalf("rotation failed: %v", err)
	}

	// When: 후속 토큰 조회
	foundChild, err := GetChildRefreshToken(ctx, tx, parent.ID)
	noChild, noChildErr := GetChildRefreshToken(ctx, tx, lonely.ID)

	// Then: 회전한 토큰은 후속 토큰을, 아닌 토큰은 nil을 반환
	if err != nil || foundChild == nil || foundChild.ID != child.ID {
		t.Errorf("expected child %d, got %+v, %v", child.ID, foundChild, err)
	}
	if noChildErr != nil || noChild != nil {
		t.Errorf("expected nil, nil; got %+v, %v", noChild, noChildErr)
	}
}

// TestRevokeRefreshTokenFamily는 계열 폐기를 테스트합니다
func TestRevokeRefreshTokenFamily(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 토큰 두 개로 이어진 계열 A와 별도 계열 B
	user := createRefreshTokenTestUser(ctx, t, tx)
	familyA := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-a1", refreshTokenTestTime.Add(time.Hour))
	if _, err := RotateRefreshToken(ctx, tx, familyA.ID, []byte("hash-a2"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime); err != nil {
		t.Fatalf("rotation failed: %v", err)
	}
	createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-b1", refreshTokenTestTime.Add(time.Hour))

	// When: 계열 A를 로그아웃으로 폐기한 뒤 재사용 사유로 다시 폐기
	revokedAt := refreshTokenTestTime.Add(time.Minute)
	if err := RevokeRefreshTokenFamily(ctx, tx, familyA.FamilyID, models.RefreshTokenRevokedByLogout, revokedAt); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := RevokeRefreshTokenFamily(ctx, tx, familyA.FamilyID, models.RefreshTokenRevokedByReuse, revokedAt.Add(time.Minute)); err != nil {
		t.Fatalf("expected no error on second revoke, got: %v", err)
	}

	// Then: 계열 A의 모든 토큰은 처음 폐기 정보를 유지
	for _, hash := range []string{"hash-a1", "hash-a2"} {
		token, _ := GetRefreshTokenByHash(ctx, tx, []byte(hash))
		if token.RevokedAt == nil || !token.RevokedAt.Equal(revokedAt) {
			t.Errorf("%s RevokedAt = %v, want %v", hash, token.RevokedAt, revokedAt)
		}
		if token.RevokedReason == nil || *token.RevokedReason != models.RefreshTokenRevokedByLogout {
			t.Errorf("%s RevokedReason = %v", hash, token.RevokedReason)
		}
	}

	// Then: 계열 B는 그대로
	other, _ := GetRefreshTokenByHash(ctx, tx, []byte("hash-b1"))
	if other.RevokedAt != nil {
		t.Errorf("expected family B untouched, got RevokedAt %v", other.RevokedAt)
	}
}

// TestDeleteStaleRefreshTokens는 오래된 토큰 정리를 테스트합니다
func TestDeleteStaleRefreshTokens(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 기준 시각 7일 전보다 오래 전에 절대 만료된 계열, 폐기된 계열, 유효한 계열
	user := createRefreshTokenTestUser(ctx, t, tx)
	cutoff := refreshTokenTestTime.Add(-7 * 24 * time.Hour)
	createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-expired", cutoff.Add(-time.Hour))
	revoked := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-revoked-old", refreshTokenTestTime.Add(time.Hour))
	if err := RevokeRefreshTokenFamily(ctx, tx, revoked.FamilyID, models.RefreshTokenRevokedByLogout, cutoff.Add(-time.Hour)); err != nil {
		t.Fatalf("failed to revoke: %v", err)
	}
	createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-alive", refreshTokenTestTime.Add(time.Hour))

	// When: 정리
	err := DeleteStaleRefreshTokens(ctx, tx, user.ID, cutoff)

	// Then: 만료·폐기된 계열만 삭제
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for hash, wantExists := range map[string]bool{"hash-expired": false, "hash-revoked-old": false, "hash-alive": true} {
		token, _ := GetRefreshTokenByHash(ctx, tx, []byte(hash))
		if (token != nil) != wantExists {
			t.Errorf("%s exists = %v, want %v", hash, token != nil, wantExists)
		}
	}
}

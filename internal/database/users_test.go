package database

import (
	"testing"

	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// TestCreateUser는 CreateUser 메서드를 테스트합니다
// 주의: 각 서브테스트마다 독립적인 트랜잭션을 사용합니다.
// 이유: 중복 체크 테스트에서 UNIQUE constraint 에러가 발생하면
// 트랜잭션이 abort 상태가 되어 이후 모든 쿼리가 실패하기 때문입니다.
func TestCreateUser(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("신규 사용자 생성 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()
		// Given: 새로운 사용자 정보
		user := &models.User{
			Email:      "test@example.com",
			Name:       "Test User",
			PictureURL: "https://example.com/pic.jpg",
			GoogleID:   "google-id-123",
		}

		// When: CreateUser 호출
		err := CreateUser(ctx, tx, user)

		// Then: 사용자 생성 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if user.ID == 0 {
			t.Error("expected non-zero user ID")
		}
		if user.CreatedAt.IsZero() {
			t.Error("expected non-zero created_at")
		}
		if user.UpdatedAt.IsZero() {
			t.Error("expected non-zero updated_at")
		}
	})

	t.Run("중복된 Google ID는 에러 반환", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 이미 존재하는 Google ID
		googleID := "google-id-duplicate"
		user1 := &models.User{
			Email:    "user1@example.com",
			Name:     "User 1",
			GoogleID: googleID,
		}

		// 첫 번째 사용자 생성
		err := CreateUser(ctx, tx, user1)
		if err != nil {
			t.Fatalf("failed to create first user: %v", err)
		}

		// When: 같은 Google ID로 두 번째 사용자 생성 시도
		user2 := &models.User{
			Email:    "user2@example.com",
			Name:     "User 2",
			GoogleID: googleID,
		}
		err = CreateUser(ctx, tx, user2)

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for duplicate google_id, got nil")
		}
	})

	t.Run("중복된 Email은 에러 반환", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 이미 존재하는 Email
		email := "duplicate@example.com"
		user1 := &models.User{
			Email:    email,
			Name:     "User 1",
			GoogleID: "google-id-1",
		}

		// 첫 번째 사용자 생성
		err := CreateUser(ctx, tx, user1)
		if err != nil {
			t.Fatalf("failed to create first user: %v", err)
		}

		// When: 같은 Email로 두 번째 사용자 생성 시도
		user2 := &models.User{
			Email:    email,
			Name:     "User 2",
			GoogleID: "google-id-2",
		}
		err = CreateUser(ctx, tx, user2)

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for duplicate email, got nil")
		}
	})
}

// TestGetUserByGoogleID는 GetUserByGoogleID 메서드를 테스트합니다
func TestGetUserByGoogleID(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("존재하는 사용자 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()
		// Given: 테스트 사용자 생성
		googleID := "google-id-exists"
		email := "exists@example.com"
		name := "Existing User"

		user := &models.User{
			Email:    email,
			Name:     name,
			GoogleID: googleID,
		}
		err := CreateUser(ctx, tx, user)
		if err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}

		// When: GetUserByGoogleID 호출
		foundUser, err := GetUserByGoogleID(ctx, tx, googleID)

		// Then: 사용자 조회 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if foundUser == nil {
			t.Fatal("expected user, got nil")
		}
		if foundUser.Email != email {
			t.Errorf("expected email=%s, got %s", email, foundUser.Email)
		}
		if foundUser.Name != name {
			t.Errorf("expected name=%s, got %s", name, foundUser.Name)
		}
		if foundUser.GoogleID != googleID {
			t.Errorf("expected google_id=%s, got %s", googleID, foundUser.GoogleID)
		}
	})

	t.Run("존재하지 않는 Google ID는 nil 반환", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 존재하지 않는 Google ID
		nonExistentGoogleID := "google-id-nonexistent"

		// When: GetUserByGoogleID 호출
		user, err := GetUserByGoogleID(ctx, tx, nonExistentGoogleID)

		// Then: nil 반환, 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if user != nil {
			t.Errorf("expected nil user, got: %+v", user)
		}
	})
}

// TestGetUserByEmail는 GetUserByEmail 메서드를 테스트합니다
func TestGetUserByEmail(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("존재하는 사용자 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()
		// Given: 테스트 사용자 생성
		email := "email@example.com"
		name := "Email User"
		googleID := "google-id-email"

		user := &models.User{
			Email:    email,
			Name:     name,
			GoogleID: googleID,
		}
		err := CreateUser(ctx, tx, user)
		if err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}

		// When: GetUserByEmail 호출
		foundUser, err := GetUserByEmail(ctx, tx, email)

		// Then: 사용자 조회 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if foundUser == nil {
			t.Fatal("expected user, got nil")
		}
		if foundUser.Email != email {
			t.Errorf("expected email=%s, got %s", email, foundUser.Email)
		}
		if foundUser.Name != name {
			t.Errorf("expected name=%s, got %s", name, foundUser.Name)
		}
	})

	t.Run("존재하지 않는 Email은 nil 반환", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 존재하지 않는 Email
		nonExistentEmail := "nonexistent@example.com"

		// When: GetUserByEmail 호출
		user, err := GetUserByEmail(ctx, tx, nonExistentEmail)

		// Then: nil 반환, 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if user != nil {
			t.Errorf("expected nil user, got: %+v", user)
		}
	})
}

// TestGetUserByID는 GetUserByID 메서드를 테스트합니다
func TestGetUserByID(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("존재하는 사용자 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()
		// Given: 테스트 사용자 생성
		email := "id@example.com"
		name := "ID User"

		user := &models.User{
			Email:    email,
			Name:     name,
			GoogleID: "google-id-for-id",
		}
		err := CreateUser(ctx, tx, user)
		if err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}
		userID := user.ID

		// When: GetUserByID 호출
		foundUser, err := GetUserByID(ctx, tx, userID)

		// Then: 사용자 조회 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if foundUser == nil {
			t.Fatal("expected user, got nil")
		}
		if foundUser.ID != userID {
			t.Errorf("expected id=%d, got %d", userID, foundUser.ID)
		}
		if foundUser.Email != email {
			t.Errorf("expected email=%s, got %s", email, foundUser.Email)
		}
		if foundUser.Name != name {
			t.Errorf("expected name=%s, got %s", name, foundUser.Name)
		}
	})

	t.Run("존재하지 않는 ID는 nil 반환", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 존재하지 않는 ID
		var nonExistentID int64 = 99999

		// When: GetUserByID 호출
		user, err := GetUserByID(ctx, tx, nonExistentID)

		// Then: nil 반환, 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if user != nil {
			t.Errorf("expected nil user, got: %+v", user)
		}
	})
}

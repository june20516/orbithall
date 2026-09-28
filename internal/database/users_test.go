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

// TestGetOrCreateUserByGoogleID는 Google ID 기준 사용자 조회·생성을 테스트합니다
// 이메일 충돌 테스트는 에러로 트랜잭션이 abort되므로 서브테스트마다 독립 트랜잭션을 씁니다
func TestGetOrCreateUserByGoogleID(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("없으면 새로 생성한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 아직 없는 Google ID의 사용자 정보
		input := &models.User{
			Email:      "get-or-create-new@example.com",
			Name:       "New User",
			PictureURL: "https://example.com/new.jpg",
			GoogleID:   "google-get-or-create-new",
		}

		// When: GetOrCreateUserByGoogleID 호출
		user, err := GetOrCreateUserByGoogleID(ctx, tx, input)

		// Then: 입력 값으로 생성되고 ID·타임스탬프가 채워짐
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if user.ID == 0 || user.CreatedAt.IsZero() {
			t.Errorf("expected persisted user, got %+v", user)
		}
		if user.Email != input.Email || user.Name != input.Name || user.PictureURL != input.PictureURL || user.GoogleID != input.GoogleID {
			t.Errorf("user = %+v, want values of %+v", user, input)
		}
	})

	t.Run("이미 있으면 기존 사용자를 그대로 반환한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 이미 저장된 사용자
		existing := &models.User{Email: "get-or-create-existing@example.com", Name: "Existing", GoogleID: "google-get-or-create-existing"}
		if err := CreateUser(ctx, tx, existing); err != nil {
			t.Fatalf("failed to create existing user: %v", err)
		}

		// When: 같은 Google ID, 다른 이름으로 호출
		user, err := GetOrCreateUserByGoogleID(ctx, tx, &models.User{
			Email:    "get-or-create-other@example.com",
			Name:     "Other Name",
			GoogleID: existing.GoogleID,
		})

		// Then: 기존 사용자가 바뀌지 않고 반환됨
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if user.ID != existing.ID || user.Email != existing.Email || user.Name != existing.Name {
			t.Errorf("user = %+v, want existing %+v", user, existing)
		}
	})

	t.Run("두 번 호출해도 한 행만 생긴다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 같은 사용자 정보
		input := models.User{Email: "get-or-create-twice@example.com", Name: "Twice", GoogleID: "google-get-or-create-twice"}

		// When: 두 번 호출
		first, err := GetOrCreateUserByGoogleID(ctx, tx, &input)
		if err != nil {
			t.Fatalf("first call failed: %v", err)
		}
		second, err := GetOrCreateUserByGoogleID(ctx, tx, &input)
		if err != nil {
			t.Fatalf("second call failed: %v", err)
		}

		// Then: 같은 ID이고 행은 하나
		if first.ID != second.ID {
			t.Errorf("IDs differ: %d vs %d", first.ID, second.ID)
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE google_id = $1`, input.GoogleID).Scan(&count); err != nil {
			t.Fatalf("failed to count users: %v", err)
		}
		if count != 1 {
			t.Errorf("users rows = %d, want 1", count)
		}
	})

	t.Run("다른 Google ID가 같은 이메일을 쓰면 에러를 반환한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 이메일을 이미 쓰는 사용자
		existing := &models.User{Email: "get-or-create-taken@example.com", Name: "Owner", GoogleID: "google-get-or-create-owner"}
		if err := CreateUser(ctx, tx, existing); err != nil {
			t.Fatalf("failed to create existing user: %v", err)
		}

		// When: 다른 Google ID로 같은 이메일 사용자 조회·생성
		user, err := GetOrCreateUserByGoogleID(ctx, tx, &models.User{Email: existing.Email, Name: "Other", GoogleID: "google-get-or-create-other"})

		// Then: 에러 반환
		if err == nil {
			t.Fatalf("expected error for duplicate email, got user %+v", user)
		}
	})
}

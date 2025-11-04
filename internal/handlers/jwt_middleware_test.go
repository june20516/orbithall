package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

func init() {
	// 테스트용 환경변수 설정
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-characters-long-12345")
	os.Setenv("JWT_EXPIRATION_HOURS", "168")
}

// TestJWTAuthMiddleware_ValidToken은 유효한 JWT 토큰으로 인증이 성공하는지 테스트합니다
func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	// DB 연결 및 트랜잭션 시작
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// 테스트 사용자 생성
	testUser := &models.User{
		Email:      "test@example.com",
		Name:       "Test User",
		PictureURL: "https://example.com/picture.jpg",
		GoogleID:   "google-test-id-123",
	}
	err := database.CreateUser(ctx, tx, testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// JWT 토큰 생성
	token, err := auth.GenerateJWT(testUser.ID, testUser.Email)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// 다음 핸들러 (미들웨어를 통과하면 호출됨)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Context에서 사용자 정보 추출
		user := GetUserFromContext(r.Context())
		if user == nil {
			t.Error("Expected user in context, got nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 사용자 정보 검증
		if user.ID != testUser.ID {
			t.Errorf("Expected user ID %d, got %d", testUser.ID, user.ID)
		}
		if user.Email != testUser.Email {
			t.Errorf("Expected email %s, got %s", testUser.Email, user.Email)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// 미들웨어 적용
	handler := JWTAuthMiddleware(tx)(nextHandler)

	// 요청 생성
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// 응답 기록
	rr := httptest.NewRecorder()

	// 핸들러 실행
	handler.ServeHTTP(rr, req)

	// 응답 검증
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", rr.Body.String())
	}
}

// TestJWTAuthMiddleware_MissingToken은 Authorization 헤더가 없을 때 401을 반환하는지 테스트합니다
func TestJWTAuthMiddleware_MissingToken(t *testing.T) {
	// DB 연결
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	// 다음 핸들러 (호출되면 안 됨)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when token is missing")
		w.WriteHeader(http.StatusOK)
	})

	// 미들웨어 적용
	handler := JWTAuthMiddleware(db)(nextHandler)

	// 요청 생성 (Authorization 헤더 없음)
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)

	// 응답 기록
	rr := httptest.NewRecorder()

	// 핸들러 실행
	handler.ServeHTTP(rr, req)

	// 응답 검증
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestJWTAuthMiddleware_InvalidFormat은 Bearer 형식이 아닐 때 401을 반환하는지 테스트합니다
func TestJWTAuthMiddleware_InvalidFormat(t *testing.T) {
	// DB 연결
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	// 다음 핸들러 (호출되면 안 됨)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when token format is invalid")
		w.WriteHeader(http.StatusOK)
	})

	// 미들웨어 적용
	handler := JWTAuthMiddleware(db)(nextHandler)

	// 요청 생성 (Bearer 없음)
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "InvalidToken")

	// 응답 기록
	rr := httptest.NewRecorder()

	// 핸들러 실행
	handler.ServeHTTP(rr, req)

	// 응답 검증
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestJWTAuthMiddleware_InvalidToken은 잘못된 JWT 토큰으로 401을 반환하는지 테스트합니다
func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	// DB 연결
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	// 다음 핸들러 (호출되면 안 됨)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when token is invalid")
		w.WriteHeader(http.StatusOK)
	})

	// 미들웨어 적용
	handler := JWTAuthMiddleware(db)(nextHandler)

	// 요청 생성 (잘못된 토큰)
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")

	// 응답 기록
	rr := httptest.NewRecorder()

	// 핸들러 실행
	handler.ServeHTTP(rr, req)

	// 응답 검증
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestJWTAuthMiddleware_UserNotFound는 존재하지 않는 사용자 ID일 때 401을 반환하는지 테스트합니다
func TestJWTAuthMiddleware_UserNotFound(t *testing.T) {
	// DB 연결
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	// 존재하지 않는 사용자 ID로 JWT 생성
	nonExistentUserID := int64(999999)
	token, err := auth.GenerateJWT(nonExistentUserID, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// 다음 핸들러 (호출되면 안 됨)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when user is not found")
		w.WriteHeader(http.StatusOK)
	})

	// 미들웨어 적용
	handler := JWTAuthMiddleware(db)(nextHandler)

	// 요청 생성
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// 응답 기록
	rr := httptest.NewRecorder()

	// 핸들러 실행
	handler.ServeHTTP(rr, req)

	// 응답 검증
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/handlers"
	"github.com/june20516/orbithall/internal/ratelimit"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/time/rate"

	_ "github.com/june20516/orbithall/docs" // swagger docs
)

// @title           Orbithall API
// @version         1.0
// @description     임베드형 댓글 시스템 API
// @termsOfService  https://orbithall.onrender.com/terms

// @contact.name   API Support
// @contact.email  june20516@gmail.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @BasePath  /
// @schemes   https http

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-Orbithall-API-Key
// @description 위젯 접근용 API Key (사이트별 발급)

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT 토큰 ("Bearer " 접두사 포함)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run은 애플리케이션의 메인 로직을 실행합니다
// 테스트 가능하도록 main()에서 분리되었습니다
func run() error {
	// ============================================
	// 환경변수 로드 (.env 파일, 로컬 개발용)
	// ============================================
	// production 환경에서는 .env 파일을 사용하지 않음
	// Railway 같은 배포 환경에서는 환경변수를 직접 설정
	if os.Getenv("ENV") != "production" {
		_ = godotenv.Load() // 에러 무시 (.env 파일 없어도 OK)
	}

	// ============================================
	// 데이터베이스 연결
	// ============================================
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := database.New(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close(db)

	log.Println("Database connected successfully")

	// ============================================
	// 핸들러 초기화
	// ============================================
	commentHandler := handlers.NewCommentHandler(db)
	authHandler := handlers.NewAuthHandler(db)
	adminHandler := handlers.NewAdminHandler(db)

	// ============================================
	// Rate Limiter 초기화
	// ============================================
	// 댓글 작성 제한: 10 req/min, burst 5
	// rate.Every()를 사용하여 분당 10개 = 6초당 1개로 설정
	createCommentLimiter := ratelimit.NewRateLimiter(rate.Every(time.Minute/10), 5)

	// ============================================
	// 라우터 설정
	// ============================================
	// Chi 라우터 생성
	// Chi는 가볍고 빠른 HTTP 라우터 라이브러리
	r := chi.NewRouter()

	// ============================================
	// 미들웨어 등록
	// ============================================
	// Logger: 모든 HTTP 요청을 로깅 (개발 시 디버깅 용이)
	r.Use(middleware.Logger)
	// Recoverer: panic 발생 시 서버가 죽지 않도록 복구
	r.Use(middleware.Recoverer)

	// ============================================
	// CORS(Cross-Origin Resource Sharing) 설정
	// ============================================
	// CORS는 사이트별 동적 검증을 사용합니다 (AuthMiddleware에서 처리)
	// 멀티 테넌시 환경이므로 글로벌 CORS 설정은 사용하지 않습니다
	// 각 사이트의 cors_origins 배열로 검증됩니다
	r.Use(cors.Handler(cors.Options{
		// 모든 origin 허용 (실제 검증은 AuthMiddleware에서 수행)
		AllowedOrigins: []string{"*"},
		// 허용할 HTTP 메서드
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// 허용할 요청 헤더
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Orbithall-API-Key", "Origin"},
		// 노출할 응답 헤더
		ExposedHeaders: []string{"Link"},
		// 쿠키 및 인증 정보 전송 허용
		AllowCredentials: false, // "*" origin 사용 시 false 필수
		// preflight 요청 캐시 시간 (초)
		MaxAge: 300,
	}))

	// ============================================
	// 라우트 정의
	// ============================================
	// 헬스체크 엔드포인트 (서버 상태 확인용)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// 응답 헤더에 JSON 타입 명시
		w.Header().Set("Content-Type", "application/json")
		// HTTP 상태 코드 200 (성공)
		w.WriteHeader(http.StatusOK)
		// JSON 형식으로 응답
		w.Write([]byte(`{"status":"ok","service":"orbithall"}`))
	})

	// Auth 라우트 그룹 (/auth 접두사, 인증 불필요)
	r.Route("/auth", func(r chi.Router) {
		// Google OAuth 검증 및 JWT 발급
		r.Post("/google/verify", authHandler.GoogleVerify)
	})

	// API 라우트 그룹 (/api 접두사)
	r.Route("/api", func(r chi.Router) {
		// 인증 미들웨어 적용 (모든 API 요청은 API 키 필요)
		r.Use(handlers.AuthMiddleware(db))

		// 댓글 CRUD 엔드포인트
		// 댓글 작성: Rate Limiting 적용 (10 req/min, burst 5)
		r.With(ratelimit.RateLimitMiddleware(createCommentLimiter)).Post("/posts/{slug}/comments", commentHandler.CreateComment)
		r.Get("/posts/{slug}/comments", commentHandler.ListComments)
		r.Put("/comments/{id}", commentHandler.UpdateComment)
		r.Delete("/comments/{id}", commentHandler.DeleteComment)
	})

	// Admin 라우트 그룹 (/admin 접두사, JWT 인증 필요)
	r.Route("/admin", func(r chi.Router) {
		// JWT 인증 미들웨어 적용 (모든 Admin 요청은 JWT 토큰 필요)
		r.Use(handlers.JWTAuthMiddleware(db))

		// 프로필 조회
		r.Get("/profile", adminHandler.GetProfile)

		// 사이트 관리
		r.Get("/sites", adminHandler.ListSites)
		r.Post("/sites", adminHandler.CreateSite)
		r.Get("/sites/{id}", adminHandler.GetSite)
		r.Put("/sites/{id}", adminHandler.UpdateSite)
		r.Delete("/sites/{id}", adminHandler.DeleteSite)

		// 사이트 통계 및 컨텐츠 조회 (016)
		r.Get("/sites/{id}/stats", adminHandler.GetSiteStats)
		r.Get("/sites/{id}/posts", adminHandler.ListSitePosts)
		r.Get("/posts/{slug}/comments", adminHandler.GetPostComments)
	})

	// ============================================
	// API 문서 (OpenAPI/Swagger)
	// ============================================
	// Swagger 2.0 문서 및 인터랙티브 API 테스트 UI
	// 환경에 따라 동적으로 URL 설정
	docsURL := os.Getenv("DOCS_URL")
	if docsURL == "" {
		// 기본값: 상대 경로 사용 (현재 호스트 기준)
		docsURL = "/docs/doc.json"
	}
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL(docsURL),
	))

	// ============================================
	// 서버 시작
	// ============================================
	// 환경변수에서 포트 읽기
	port := os.Getenv("PORT")
	if port == "" {
		// 환경변수가 없으면 기본 포트 사용
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	// HTTP 서버 시작 (블로킹)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

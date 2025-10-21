package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/handlers"
)

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
	// 다른 도메인(Next.js 블로그)에서 이 API를 호출할 수 있도록 허용
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		// 환경변수가 없으면 기본값 사용
		corsOrigin = "http://localhost:3000"
	}

	r.Use(cors.Handler(cors.Options{
		// 허용할 도메인 목록
		AllowedOrigins: []string{corsOrigin},
		// 허용할 HTTP 메서드
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// 허용할 요청 헤더
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Orbithall-API-Key", "Origin"},
		// 노출할 응답 헤더
		ExposedHeaders: []string{"Link"},
		// 쿠키 및 인증 정보 전송 허용
		AllowCredentials: true,
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

	// API 라우트 그룹 (/api 접두사)
	r.Route("/api", func(r chi.Router) {
		// 인증 미들웨어 적용 (모든 API 요청은 API 키 필요)
		r.Use(handlers.AuthMiddleware(db))

		// 댓글 CRUD 엔드포인트
		r.Post("/posts/{slug}/comments", commentHandler.CreateComment)
		r.Get("/posts/{slug}/comments", commentHandler.ListComments)
		r.Put("/comments/{id}", commentHandler.UpdateComment)
		r.Delete("/comments/{id}", commentHandler.DeleteComment)
	})

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

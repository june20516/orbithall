package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
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
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
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
		// 댓글 목록 조회 엔드포인트 (임시)
		r.Get("/comments", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"comments endpoint"}`))
		})
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
	// err가 nil이 아니면 치명적 오류로 종료
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

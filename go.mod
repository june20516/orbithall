// Go 모듈 이름 (패키지 import 시 사용될 경로)
module github.com/bran/orbithall

// Go 버전 (1.21 사용)
go 1.21

// 직접 의존하는 외부 패키지 목록
require (
	github.com/go-chi/chi/v5 v5.0.11  // HTTP 라우터 (경량, 빠름)
	github.com/go-chi/cors v1.2.1     // CORS 미들웨어 (도메인 간 요청 허용)
	github.com/lib/pq v1.10.9         // PostgreSQL 드라이버
)

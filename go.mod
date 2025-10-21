// Go 모듈 이름 (패키지 import 시 사용될 경로)
module github.com/june20516/orbithall

// Go 버전 (1.25 사용)
go 1.25

// 직접 의존하는 외부 패키지 목록
require (
	github.com/go-chi/chi/v5 v5.0.11 // HTTP 라우터 (경량, 빠름)
	github.com/go-chi/cors v1.2.1 // CORS 미들웨어 (도메인 간 요청 허용)
)

require github.com/lib/pq v1.10.9

require (
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/net v0.45.0 // indirect
)

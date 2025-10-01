# ============================================
# Development Stage (로컬 개발용)
# ============================================
FROM golang:1.25-alpine AS development

# 작업 디렉토리 설정
WORKDIR /app

# Air 설치 (Go 파일 변경 시 자동 재컴파일 및 재시작)
RUN go install github.com/air-verse/air@latest

# Go 모듈 파일 복사 (의존성 캐싱을 위해 소스 코드보다 먼저 복사)
# go.sum은 go mod download 실행 시 자동 생성됨
COPY go.mod ./

# 의존성 다운로드 (go.sum이 자동 생성됨)
RUN go mod download

# 전체 소스 코드 복사
COPY . .

# go.sum 파일 생성/업데이트 (실행 전 필수)
RUN go mod tidy

# API 서버 포트 노출
EXPOSE 8080

# 기본 실행 명령 (docker-compose에서 오버라이드됨)
CMD ["air", "-c", ".air.toml"]

# ============================================
# Production Build Stage (프로덕션 빌드용)
# ============================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Go 모듈 파일 복사
# go.sum은 go mod download 실행 시 자동 생성됨
COPY go.mod ./

# 의존성 다운로드
RUN go mod download

# 전체 소스 코드 복사
COPY . .

# go.sum 파일 생성/업데이트 (빌드 전 필수)
RUN go mod tidy

# 정적 바이너리 빌드 (CGO 비활성화로 다른 의존성 없이 실행 가능)
# GOOS=linux: Linux용 바이너리 생성
# -o: 출력 파일명
# ./cmd/api: 빌드할 패키지 경로
RUN CGO_ENABLED=0 GOOS=linux go build -o /orbithall ./cmd/api

# ============================================
# Production Final Stage (최종 프로덕션 이미지)
# ============================================
FROM alpine:latest AS production

# SSL/TLS 인증서 설치 (HTTPS 요청을 위해 필요)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 빌드된 바이너리만 복사 (이미지 크기 최소화)
COPY --from=builder /orbithall .

# API 서버 포트 노출
EXPOSE 8080

# 바이너리 실행
CMD ["./orbithall"]

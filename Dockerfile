# ============================================
# Development Stage (로컬 개발용)
# ============================================
FROM golang:1.21-alpine AS development

# 작업 디렉토리 설정
WORKDIR /app

# Air 설치 (Go 파일 변경 시 자동 재컴파일 및 재시작)
RUN go install github.com/cosmtrek/air@latest

# Go 모듈 파일 복사 (의존성 캐싱을 위해 소스 코드보다 먼저 복사)
# go.sum이 없어도 go mod download가 자동 생성하므로 go.mod만 필수
COPY go.mod ./
# go.sum이 있으면 복사 (없어도 무방)
COPY go.su[m] ./ 2>/dev/null || true

# 의존성 다운로드 (go.sum이 자동 생성됨)
RUN go mod download

# 전체 소스 코드 복사
COPY . .

# API 서버 포트 노출
EXPOSE 8080

# 기본 실행 명령 (docker-compose에서 오버라이드됨)
CMD ["air", "-c", ".air.toml"]

# ============================================
# Production Build Stage (프로덕션 빌드용)
# ============================================
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Go 모듈 파일 복사
COPY go.mod ./
COPY go.su[m] ./ 2>/dev/null || true

# 의존성 다운로드
RUN go mod download

# 전체 소스 코드 복사
COPY . .

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

# Orbithall

임베드형 댓글 시스템 (Disqus 스타일)

## 개요

Orbithall은 Next.js SSG 블로그를 위한 독립형 댓글 시스템입니다. 블로그에 임베드하여 사용할 수 있으며, 별도의 서버에서 실행됩니다.

## 기술 스택

- **Backend**: Go 1.25
- **Framework**: Chi (HTTP 라우터)
- **Database**: PostgreSQL 16
- **Infrastructure**: Docker + Docker Compose
- **Deploy Target**: Railway

## 프로젝트 구조

```
orbithall/
├── cmd/api/              # 애플리케이션 진입점
│   └── main.go          # 서버 시작 및 라우팅
├── internal/            # 내부 패키지 (외부에서 import 불가)
│   ├── handlers/        # HTTP 핸들러 (추후 사용)
│   ├── models/          # 데이터 모델 (추후 사용)
│   └── database/        # DB 연결 로직 (추후 사용)
├── .air.toml            # Air 설정 (hot reload)
├── .gitignore           # Git 제외 파일
├── docker-compose.yml   # 로컬 개발 환경 설정
├── Dockerfile           # 컨테이너 이미지 빌드 설정
├── go.mod               # Go 모듈 의존성
└── README.md            # 프로젝트 문서
```

## 사전 요구사항

- Docker Desktop (필수)

모든 개발 환경이 Docker 컨테이너에서 실행되므로 Go, PostgreSQL 등의 로컬 설치가 불필요합니다.

## 로컬 개발 환경 구성

### 1. 프로젝트 클론 또는 다운로드

```bash
cd /path/to/your/workspace
```

### 2. Docker Compose로 서버 실행

```bash
cd orbithall
docker-compose up
```

이 명령어는 다음을 자동으로 수행합니다:
- PostgreSQL 16 컨테이너 시작
- Go 의존성 다운로드 (go.sum 자동 생성)
- API 서버 빌드 및 실행
- 코드 변경 시 자동 재시작 (Air)

### 3. 서버 확인

```bash
# 헬스체크
curl http://localhost:8080/health

# 댓글 API (임시)
curl http://localhost:8080/api/comments
```

## API 엔드포인트

### 헬스체크
```
GET /health
```

응답 예시:
```json
{
  "status": "ok",
  "service": "orbithall"
}
```

### 댓글 목록 (임시)
```
GET /api/comments
```

응답 예시:
```json
{
  "message": "comments endpoint"
}
```

## 환경변수

| 변수명 | 설명 | 기본값 |
|--------|------|--------|
| `PORT` | API 서버 포트 | `8080` |
| `DATABASE_URL` | PostgreSQL 연결 문자열 | docker-compose에서 자동 설정 |
| `CORS_ORIGIN` | CORS 허용 도메인 | `http://localhost:3000` |
| `ENV` | 환경 (development/production) | `development` |

## 블로그와 연동

### 1. 블로그에서 API 호출

```typescript
// codeverse/components/Comments.tsx
const response = await fetch('http://localhost:8080/api/comments?postId=123');
const data = await response.json();
```

### 2. 개발 플로우

```bash
# 터미널 1: 블로그 실행
cd codeverse
yarn dev          # → http://localhost:3000

# 터미널 2: 댓글 API 실행
cd orbithall
docker-compose up # → http://localhost:8080
```

## 개발 팁

### 컨테이너 재시작
```bash
docker-compose restart
```

### 컨테이너 중지 및 제거
```bash
docker-compose down
```

### 데이터베이스 초기화 (볼륨 삭제)
```bash
docker-compose down -v
```

### 로그 확인
```bash
docker-compose logs -f api      # API 서버 로그
docker-compose logs -f postgres # DB 로그
```

### PostgreSQL 직접 접속
```bash
docker exec -it orbithall-db psql -U orbithall -d orbithall_db
```

## 다음 단계

- [ ] 데이터베이스 스키마 설계
- [ ] 댓글 CRUD API 구현
- [ ] 기본 보안 구현 (Rate Limiting, Input Validation)
- [ ] Railway 배포 설정
- [ ] 프로덕션 환경변수 관리

## 라이선스

MIT

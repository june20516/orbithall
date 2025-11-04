# Orbithall

임베드형 댓글 시스템

## 개요

Orbithall은 slug 기반 웹 콘텐츠를 위한 독립형 댓글 시스템입니다. 프레임워크나 빌드 방식에 관계없이 임베드하여 사용할 수 있으며, 별도의 서버에서 실행됩니다.

## 기술 스택

- **Backend**: Go 1.25
- **Framework**: Chi (HTTP 라우터)
- **Database**: PostgreSQL 16 (Supabase)
- **Infrastructure**: Docker + Docker Compose
- **Deployment**: Render (API) + Supabase (Database)

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
- Go 1.25 (권장, 테스트 실행용)

API 서버와 PostgreSQL은 Docker에서 실행되며, 테스트는 로컬 Go에서 실행합니다.

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

### Swagger 문서

- http://localhost:8080/swagger/index.html

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

### 댓글 생성

```
POST /api/posts/:slug/comments
Headers: X-Orbithall-API-Key
```

요청 예시:

```json
{
  "author_name": "작성자",
  "password": "1234",
  "content": "댓글 내용"
}
```

### 댓글 조회

```
GET /api/posts/:slug/comments
Headers: X-Orbithall-API-Key
```

응답 예시:

```json
{
  "comments": [...],
  "pagination": {
    "current_page": 1,
    "total_pages": 1,
    "total_comments": 10,
    "per_page": 50
  }
}
```

## 환경변수

| 변수명         | 설명                          | 기본값                       |
| -------------- | ----------------------------- | ---------------------------- |
| `PORT`         | API 서버 포트                 | `8080`                       |
| `DATABASE_URL` | PostgreSQL 연결 문자열        | docker-compose에서 자동 설정 |
| `ENV`          | 환경 (development/production) | `development`                |

**참고**: CORS는 사이트별 동적 검증 방식을 사용합니다. 각 사이트의 `cors_origins` 배열로 관리됩니다.

## Rate Limiting

API 남용 방지를 위해 IP 기반 요청 제한이 적용됩니다.

### 제한 정책

- **댓글 작성**: 10회/분 (burst: 5)
- **댓글 조회**: 제한 없음
- **댓글 수정/삭제**: 제한 없음 (30분 시간 제한으로 충분)

### 제한 초과 시

- **HTTP 상태 코드**: 429 Too Many Requests
- **응답 예시**:
  ```json
  {
    "error": "rate_limit_exceeded",
    "message": "Too many requests. Please try again later."
  }
  ```
- **Retry-After 헤더**: 재시도 대기 시간 (초 단위)

### IP 추출 방식

프록시 환경을 고려하여 다음 순서로 IP를 확인합니다:

1. `X-Forwarded-For` 헤더 (첫 번째 IP)
2. `X-Real-IP` 헤더
3. `RemoteAddr` (직접 연결)

## JS Widget

블로그나 웹사이트에 임베드할 수 있는 댓글 위젯을 제공합니다.

### 빠른 시작

```html
<link
  rel="stylesheet"
  href="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css"
/>

<div
  data-orb-container
  data-widget-type="comments"
  data-post-slug="your-post-id"
></div>

<script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js"></script>
<script>
  OrbitHall.init({ apiKey: "YOUR_API_KEY" });
</script>
```

**상세한 사용법은 [widget/README.md](widget/README.md)를 참고하세요.**

- Next.js, React 통합 가이드
- 스타일 커스터마이징
- 다국어 설정
- API 레퍼런스

## 웹사이트 연동 (직접 API 호출)

위젯을 사용하지 않고 직접 API를 호출하려면:

### API 호출 예시

```typescript
// 댓글 조회
const response = await fetch(
  "https://orbithall.onrender.com/api/posts/my-post-slug/comments",
  {
    headers: {
      "X-Orbithall-API-Key": "YOUR_API_KEY",
    },
  }
);
const data = await response.json();

// 댓글 작성
await fetch("https://orbithall.onrender.com/api/posts/my-post-slug/comments", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-Orbithall-API-Key": "YOUR_API_KEY",
  },
  body: JSON.stringify({
    author: "작성자",
    password: "1234",
    content: "댓글 내용",
    parentId: null, // 대댓글인 경우 부모 댓글 ID
  }),
});
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

### json/yaml을 직접 확인하려는 경우

```bash
  ~/go/bin/swag init -g cmd/api/main.go --output ./docs
```

## 테스트

### 테스트 실행

테스트는 PostgreSQL 컨테이너만 필요합니다 (API 서버는 불필요).

```bash
# PostgreSQL 컨테이너만 시작
docker-compose up -d postgres

# 테스트 실행
go test ./...

# 특정 패키지 테스트
go test ./internal/handlers/...
go test ./internal/database/...

# 상세 출력 (-v)
go test -v ./...

# 커버리지 확인
go test -cover ./...
```

### 테스트 데이터베이스

- **자동 생성**: `docker-compose up` 시 `test_orbithall_db` 자동 생성
- **자동 마이그레이션**: 테스트 실행 시 최신 스키마 자동 적용 (golang-migrate)
- **환경변수**: `TEST_DATABASE_URL` (`.env` 파일 또는 환경변수로 설정)
  ```bash
  TEST_DATABASE_URL=postgres://orbithall:dev_password@localhost:5432/test_orbithall_db?sslmode=disable
  ```

### 테스트 전략

- **Transaction-based Testing**: 모든 통합 테스트는 트랜잭션 내에서 실행 후 자동 롤백
- **격리성**: 각 테스트는 완전히 독립적이며, 데이터 누수 없음
- **자동 정리**: Cleanup 함수 불필요, `defer cleanup()` 한 줄로 자동 롤백

자세한 내용은 다음 문서를 참고하세요:

- `docs/adr/002-transaction-based-testing-strategy.md`

## 배포 정보

### 프로덕션 환경

- **API URL**: https://orbithall.onrender.com
- **Database**: Supabase (Seoul Region)
- **Auto-Deploy**: main 브랜치 푸시 시 자동 배포

자세한 배포 가이드는 `docs/tasks/completed/003-deployment.md`를 참고하세요.

## 다음 단계

- [x] 데이터베이스 스키마 설계
- [x] 댓글 CRUD API 구현
- [x] 프로덕션 배포 (Render + Supabase)
- [x] JS Widget 구현 (Preact + Bun)
- [x] jsDelivr CDN 배포
- [x] Rate Limiting 구현
- [ ] 반응(Reactions) 위젯 구현
- [ ] 실제 웹사이트 연동

## 라이선스

MIT

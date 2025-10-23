# Orbithall

임베드형 댓글 시스템 (Disqus 스타일)

## 개요

Orbithall은 Next.js SSG 블로그를 위한 독립형 댓글 시스템입니다. 블로그에 임베드하여 사용할 수 있으며, 별도의 서버에서 실행됩니다.

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

| 변수명 | 설명 | 기본값 |
|--------|------|--------|
| `PORT` | API 서버 포트 | `8080` |
| `DATABASE_URL` | PostgreSQL 연결 문자열 | docker-compose에서 자동 설정 |
| `ENV` | 환경 (development/production) | `development` |

**참고**: CORS는 사이트별 동적 검증 방식을 사용합니다. 각 사이트의 `cors_origins` 배열로 관리됩니다.

## JS Widget 사용 가이드

OrbitHall은 임베드형 JavaScript 위젯을 제공합니다. 블로그나 웹사이트에 간단하게 추가할 수 있습니다.

### 1. 기본 사용법

```html
<!DOCTYPE html>
<html>
<head>
  <!-- CSS 로드 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/june20516/orbithall@main/static/embed.css">
</head>
<body>
  <!-- 댓글 위젯 컨테이너 -->
  <div
    data-widget-type="comments"
    data-post-slug="my-first-post"
  ></div>

  <!-- JS 로드 및 초기화 -->
  <script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@main/static/embed.js"></script>
  <script>
    OrbitHall.init({
      apiKey: 'YOUR_API_KEY_HERE'
    });
  </script>
</body>
</html>
```

### 2. Next.js (App Router) 연동

```tsx
// app/layout.tsx - 레이아웃에서 한 번만 초기화
import Script from 'next/script';

export default function RootLayout({ children }) {
  return (
    <html>
      <head>
        <link
          rel="stylesheet"
          href="https://cdn.jsdelivr.net/gh/june20516/orbithall@main/static/embed.css"
        />
      </head>
      <body>
        {children}

        <Script
          src="https://cdn.jsdelivr.net/gh/june20516/orbithall@main/static/embed.js"
          onLoad={() => {
            window.OrbitHall.init({
              apiKey: process.env.NEXT_PUBLIC_ORBITHALL_API_KEY!
            });
          }}
        />
      </body>
    </html>
  );
}

// app/posts/[slug]/page.tsx - 각 포스트 페이지
export default function PostPage({ params }: { params: { slug: string } }) {
  return (
    <article>
      <h1>Post Title</h1>
      <div>{/* Post content */}</div>

      {/* 댓글 위젯 - data-post-slug만 변경하면 자동 리로드 */}
      <div
        data-widget-type="comments"
        data-post-slug={params.slug}
      />
    </article>
  );
}
```

### 3. WordPress 연동

```php
<!-- single.php 또는 테마 파일 -->
<div
  data-widget-type="comments"
  data-post-slug="<?php echo get_post_field('post_name'); ?>"
></div>

<?php wp_footer(); ?>
<script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@main/static/embed.js"></script>
<script>
  OrbitHall.init({
    apiKey: '<?php echo get_option('orbithall_api_key'); ?>'
  });
</script>
```

### 4. 스타일 커스터마이징

CSS 변수를 사용하여 위젯 스타일을 커스터마이징할 수 있습니다:

```css
:root {
  --orbithall-primary-color: #3b82f6;
  --orbithall-border-radius: 0.5rem;
  --orbithall-font-family: 'Your Font', sans-serif;
}
```

사용 가능한 CSS 변수 목록은 `widget/src/styles.css`를 참고하세요.

### 5. 개발 환경

로컬 개발 시 위젯은 자동으로 로컬 API를 사용합니다:
- **개발**: `http://localhost:8080/api`
- **프로덕션**: `https://orbithall.onrender.com/api`

```bash
# 위젯 개발 모드 (watch + hot reload)
cd widget
bun install
bun run dev

# 프로덕션 빌드
bun run build

# 브랜치별 배포
bun run publish:develop  # develop 브랜치로 배포 (베타)
bun run publish:main     # main 브랜치로 배포 (프로덕션)
```

## 블로그와 연동 (직접 API 호출)

위젯을 사용하지 않고 직접 API를 호출하려면:

### 1. API 호출 예시

```typescript
// 댓글 조회
const response = await fetch('https://orbithall.onrender.com/api/posts/my-post-slug/comments', {
  headers: {
    'X-Orbithall-API-Key': 'YOUR_API_KEY'
  }
});
const data = await response.json();

// 댓글 작성
await fetch('https://orbithall.onrender.com/api/posts/my-post-slug/comments', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Orbithall-API-Key': 'YOUR_API_KEY'
  },
  body: JSON.stringify({
    author: '작성자',
    password: '1234',
    content: '댓글 내용',
    parentId: null // 대댓글인 경우 부모 댓글 ID
  })
});
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
- [ ] Rate Limiting 구현
- [ ] 반응(Reactions) 위젯 구현
- [ ] 실제 블로그 연동

## 라이선스

MIT

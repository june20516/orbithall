# 데이터베이스 연결 및 초기 스키마 구현

## 작성일
2025-10-02

## 우선순위
- [x] 긴급
- [ ] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
PostgreSQL 데이터베이스 연결 패키지 구현 및 멀티 테넌시 지원 초기 스키마 마이그레이션 파일 작성

## 작업 목적
여러 사이트가 사용할 수 있는 댓글 시스템의 데이터 저장소를 구성하고, 안전하고 효율적인 데이터베이스 연결을 제공합니다. API 키 기반 인증과 사이트별 데이터 격리를 구현합니다.

## 작업 범위

### 포함 사항
- `internal/database/db.go` 파일 생성 (연결 관리)
- `internal/database/cache.go` 파일 생성 (사이트 정보 캐싱)
- `migrations/001_initial_schema.sql` 파일 생성 (스키마 정의)
- `internal/models/site.go` 파일 생성 (Site 구조체)
- `internal/models/post.go` 파일 생성 (Post 구조체)
- `internal/models/comment.go` 파일 생성 (Comment 구조체)
- `cmd/api/main.go` 수정 (DB 연결 통합)
- `go.mod` 업데이트 (PostgreSQL 드라이버 추가)

### 제외 사항
- 실제 댓글 CRUD API 구현 (다음 작업에서 진행)
- 읽기 복제 설정 (추후 도입)

## 기술적 접근

### 사용할 기술/라이브러리
- **github.com/lib/pq**: PostgreSQL 드라이버 (Go 표준, 안정적)
- **database/sql**: Go 표준 DB 인터페이스
- **bcrypt**: 비밀번호 해싱 (golang.org/x/crypto/bcrypt)

### 파일 구조
```
orbithall/
├── cmd/api/
│   └── main.go                          # DB 초기화 코드 추가
├── internal/
│   ├── database/
│   │   ├── db.go                        # DB 연결 및 Pool 관리
│   │   └── cache.go                     # 사이트 정보 캐싱 (TTL 1분)
│   └── models/
│       ├── site.go                      # Site 구조체
│       ├── post.go                      # Post 구조체
│       └── comment.go                   # Comment 구조체
├── migrations/
│   └── 001_initial_schema.sql           # 초기 스키마 (sites, posts, comments)
└── go.mod                               # 의존성 추가
```

### 구현 단계

#### 1. PostgreSQL 드라이버 추가
```bash
# go.mod에 자동 추가됨 (코드에서 import하면)
import _ "github.com/lib/pq"
```

#### 2. internal/database/db.go 생성
- `New(databaseURL string) (*sql.DB, error)` 함수
  - DATABASE_URL 파싱
  - sql.Open() 호출
  - Connection Pool 설정
  - Ping()으로 연결 확인
  - 에러 시 명확한 메시지
- `Close(db *sql.DB) error` 함수
  - graceful shutdown

#### 3. migrations/001_initial_schema.sql 생성
- sites 테이블 (API 키, CORS origins 포함)
- posts 테이블 (site_id 외래키 포함)
- comments 테이블
- 모든 인덱스
- 명세서의 스키마 그대로 구현

#### 4. internal/models/site.go 생성
```go
type Site struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    Domain      string    `json:"domain"`
    APIKey      string    `json:"api_key"`
    CORSOrigins []string  `json:"cors_origins"`  // PostgreSQL TEXT[]
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

#### 5. internal/models/post.go 생성
```go
type Post struct {
    ID           int64     `json:"id"`
    SiteID       int64     `json:"site_id"`
    Slug         string    `json:"slug"`
    Title        string    `json:"title"`
    CommentCount int       `json:"comment_count"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

#### 6. internal/models/comment.go 생성
```go
type Comment struct {
    ID             int64      `json:"id"`
    PostID         int64      `json:"post_id"`
    ParentID       *int64     `json:"parent_id,omitempty"`
    AuthorName     string     `json:"author_name"`
    AuthorPassword string     `json:"-"`  // 응답에 포함하지 않음
    Content        string     `json:"content"`
    IsDeleted      bool       `json:"is_deleted"`
    IPAddress      string     `json:"-"`  // 응답에 포함하지 않음
    UserAgent      string     `json:"-"`  // 응답에 포함하지 않음
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
```

#### 7. internal/database/cache.go 생성
- Site 정보 캐싱을 위한 구조체
- sync.Map 사용 (thread-safe)
- TTL 1분 구현
- `GetSiteByAPIKey(db *sql.DB, apiKey string) (*models.Site, error)` 함수
  - 캐시 히트: 캐시에서 반환
  - 캐시 미스: DB 조회 후 캐싱

#### 8. cmd/api/main.go 수정
- DATABASE_URL 환경변수 읽기
- database.New() 호출
- defer db.Close()
- 에러 시 서버 시작 중단
- 연결 성공 로그

## 검증 방법

### 테스트 케이스
1. **테스트 시나리오 1: DB 연결 성공**
   - 입력: `docker-compose up`
   - 예상 결과:
     ```
     Database connected successfully
     Server starting on port 8080
     ```

2. **테스트 시나리오 2: 스키마 생성**
   - 입력:
     ```bash
     docker exec -i orbithall-db psql -U orbithall -d orbithall_db < migrations/001_initial_schema.sql
     ```
   - 예상 결과: `CREATE TABLE`, `CREATE INDEX` 메시지

3. **테스트 시나리오 3: 테이블 확인**
   - 입력:
     ```bash
     docker exec -it orbithall-db psql -U orbithall -d orbithall_db -c "\dt"
     ```
   - 예상 결과: posts, comments 테이블 표시

4. **테스트 시나리오 4: 사이트 등록**
   - 입력:
     ```sql
     INSERT INTO sites (name, domain, cors_origins)
     VALUES ('코드버스', 'blog.codeverse.com', ARRAY['http://localhost:3000']);
     SELECT api_key FROM sites WHERE domain = 'blog.codeverse.com';
     ```
   - 예상 결과: UUID 형식 API 키 자동 생성

5. **테스트 시나리오 5: 한글 저장/조회**
   - 입력:
     ```sql
     INSERT INTO posts (site_id, slug, title) VALUES (1, 'test', '테스트 포스트');
     SELECT * FROM posts;
     ```
   - 예상 결과: 한글 깨짐 없이 조회

6. **테스트 시나리오 6: 데이터 격리**
   - 입력: 2개 사이트 등록, 각각 댓글 작성 후 site_id로 필터링
   - 예상 결과: 각 사이트의 댓글만 조회됨

### 수동 확인
- [ ] `go mod tidy` 실행 시 에러 없음
- [ ] 애플리케이션 시작 시 DB 연결 성공 로그 출력
- [ ] 마이그레이션 실행 후 sites, posts, comments 테이블 존재 확인
- [ ] 인덱스 생성 확인 (`\di` 명령어)
- [ ] 한글 데이터 정상 처리 확인
- [ ] UUID API 키 자동 생성 확인

## 의존성
- 선행 작업: 없음 (Docker 환경 이미 구성됨)
- 후속 작업: 댓글 CRUD API 구현

## 예상 소요 시간
- 예상: 1-2시간
- 실제: (완료 후 기록)

## 주의사항

### 코드 작성 시
- DATABASE_URL이 없을 때 명확한 에러 메시지
- sql.Open()은 실제 연결하지 않음 (Ping() 필수)
- Connection Pool 설정값은 명세서 참고
- 주석은 Go 초보자도 이해 가능하게

### 마이그레이션 작성 시
- UTF-8 인코딩 확인
- TIMESTAMPTZ 사용 (TIMESTAMP 아님)
- ON DELETE CASCADE 명시
- 인덱스는 실제 쿼리 패턴 고려

### 보안
- 비밀번호 필드는 `json:"-"` 태그
- IP 주소 등 민감정보는 응답에 포함 안함
- 환경변수 값은 로그에 출력하지 않음

## 참고 자료
- 명세서: `docs/specs/database-setup.md`
- Go database/sql: https://pkg.go.dev/database/sql
- lib/pq: https://github.com/lib/pq

## 상세 구현 가이드

### internal/database/db.go 구현 예시 구조
```go
package database

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/lib/pq"
)

// New는 PostgreSQL 데이터베이스 연결을 생성하고 설정합니다
func New(databaseURL string) (*sql.DB, error) {
    // 1. DATABASE_URL 검증
    if databaseURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }

    // 2. 데이터베이스 연결 (실제 연결은 아직 안 됨)
    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // 3. Connection Pool 설정
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(5 * time.Minute)

    // 4. 실제 연결 테스트
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return db, nil
}

// Close는 데이터베이스 연결을 종료합니다
func Close(db *sql.DB) error {
    if db != nil {
        return db.Close()
    }
    return nil
}
```

### cmd/api/main.go 수정 가이드
```go
// main 함수 시작 부분에 추가

// DATABASE_URL 환경변수 읽기
databaseURL := os.Getenv("DATABASE_URL")
if databaseURL == "" {
    log.Fatal("DATABASE_URL environment variable is required")
}

// 데이터베이스 연결
db, err := database.New(databaseURL)
if err != nil {
    log.Fatalf("Failed to connect to database: %v", err)
}
defer database.Close(db)

log.Println("Database connected successfully")

// 기존 라우터 설정 코드...
```

### migrations/001_initial_schema.sql 작성 가이드
```sql
-- 트랜잭션 시작 (전체 성공 또는 전체 실패)
BEGIN;

-- posts 테이블
CREATE TABLE posts (
    -- 명세서 내용 그대로
);

-- comments 테이블
CREATE TABLE comments (
    -- 명세서 내용 그대로
);

-- 인덱스들
CREATE INDEX idx_posts_slug ON posts(slug);
-- 나머지 인덱스들

-- 트랜잭션 커밋
COMMIT;
```

---

## 작업 이력

### [2025-10-02] 작업 문서 작성
- 명세서 기반 작업 문서 작성
- 구현 단계 및 검증 방법 정의

### [2025-10-02] 멀티 테넌시 구조로 업데이트
- sites 테이블 추가
- Site 모델 추가
- API 키 캐싱 로직 추가 (cache.go)
- 데이터 격리 테스트 추가

### [2025-10-02] 구현 완료
**구현된 파일 (11개):**
1. internal/models/site.go
2. internal/models/post.go
3. internal/models/comment.go
4. internal/database/db.go
5. internal/database/db_test.go
6. internal/database/cache.go
7. internal/database/cache_test.go
8. migrations/001_initial_schema.up.sql
9. migrations/001_initial_schema.down.sql
10. cmd/api/main.go (수정)
11. cmd/api/main_test.go

**자동 마이그레이션 추가 (계획 변경):**
- Dockerfile에 golang-migrate 설치 (development, production stage)
- entrypoint.sh 생성 (앱 시작 전 마이그레이션 자동 실행)
- ADR-001 작성 (데이터베이스 마이그레이션 전략 결정 문서화)
- 이유: 개발/프로덕션 환경 일관성, Railway 배포 최적화

**수정 사항:**
- 마이그레이션 파일명: `.up.sql`, `.down.sql` 규칙 적용 (golang-migrate 요구사항)
- docker-compose.yml healthcheck 수정: `-d orbithall_db` 추가
- go.mod 모듈명 수정: `github.com/june20516/orbithall`

**추가 문서:**
- docs/adr/README.md (ADR 사용 가이드)
- docs/adr/001-database-migration-strategy.md

**검증 완료:**
- ✅ Docker 환경 빌드 및 실행 성공
- ✅ 마이그레이션 자동 실행 확인
- ✅ DB 연결 성공 로그 확인
- ✅ 멱등성 보장 (재시작 시 중복 실행 안함)

**실제 소요 시간:** 약 3시간 (문서 작성 포함)

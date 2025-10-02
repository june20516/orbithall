# 데이터베이스 연결 및 스키마 명세서

## 작성일
2025-10-02

## 버전
v1.1

## 개요
PostgreSQL 데이터베이스 연결 설정 및 멀티 테넌시 댓글 시스템을 위한 초기 스키마를 구성합니다. 여러 사이트가 하나의 Orbithall 인스턴스를 공유할 수 있도록 설계하며, 보안, 성능, 한국어 지원을 고려한 프로덕션 레디 설정을 목표로 합니다.

## 목적 및 배경
- 멀티 테넌시 아키텍처 (여러 사이트 지원)
- 안전하고 효율적인 데이터베이스 연결 관리
- 한국어 댓글 저장 및 검색 최적화
- 확장 가능한 스키마 설계
- API 키 기반 사이트별 인증 및 데이터 격리
- 프로덕션 환경에서의 안정성 확보

## 사용자 스토리
```
AS A 블로그 운영자
I WANT 여러 사이트에서 하나의 Orbithall 인스턴스를 공유하며 댓글 시스템 사용
SO THAT 각 사이트의 댓글이 안전하게 격리되고 관리될 수 있다
```

## 기능 요구사항

### 필수 기능
1. **멀티 테넌시 지원**
   - 설명: 여러 사이트의 데이터를 안전하게 격리
   - 조건: 사이트별 API 키 기반 인증
   - 결과: 사이트 간 데이터 접근 불가

2. **데이터베이스 연결 관리**
   - 설명: Connection Pool을 사용한 효율적인 연결 관리
   - 조건: 환경변수를 통한 안전한 credential 관리
   - 결과: 재사용 가능한 DB 연결 패키지

3. **스키마 마이그레이션**
   - 설명: 버전 관리 가능한 스키마 생성
   - 조건: SQL 파일 기반, 수동 실행 방식
   - 결과: 초기 테이블 및 인덱스 생성 (sites, posts, comments)

4. **한국어 지원**
   - 설명: UTF-8 인코딩 및 적절한 collation 설정
   - 조건: 한글 검색 및 정렬 가능
   - 결과: 한국어 댓글 정상 처리

5. **동적 CORS 처리**
   - 설명: 사이트별 CORS 설정을 DB에서 조회
   - 조건: 메모리 캐싱 (TTL 1분)
   - 결과: 등록된 도메인에서만 접근 가능

### 선택 기능
1. **마이그레이션 도구**
   - 설명: golang-migrate 같은 도구 사용
   - 우선순위: 낮음 (추후 도입)

## 비기능 요구사항

### 성능
- Connection Pool: 최소 5, 최대 25 연결
- Idle Connection: 5분 후 자동 해제
- Connection Timeout: 5초
- Query Timeout: 10초

### 보안
- 환경변수로 credential 관리 (.env 사용)
- Prepared Statement 사용 (SQL Injection 방지)
- 최소 권한 원칙 (필요한 권한만 부여)
- SSL/TLS 연결 (프로덕션)

### 확장성
- 인덱스 전략: 조회 쿼리 최적화
- 파티셔닝 준비: created_at 기반 (추후)
- 읽기 복제 준비 (추후)

## PostgreSQL 설정

### 데이터베이스 설정
```sql
-- 데이터베이스 생성 (Docker에서 자동 생성되지만 설정 명시)
CREATE DATABASE orbithall_db
  ENCODING 'UTF8'
  LC_COLLATE 'C.UTF-8'      -- 정렬 규칙 (유니코드 기본)
  LC_CTYPE 'C.UTF-8'        -- 문자 분류 (유니코드 기본)
  TEMPLATE template0;        -- 깨끗한 템플릿 사용

-- 타임존 설정
ALTER DATABASE orbithall_db SET timezone TO 'Asia/Seoul';
```

### Connection Pool 설정 (Go)
```go
// MaxOpenConns: 동시 연결 최대 개수
db.SetMaxOpenConns(25)

// MaxIdleConns: 유휴 연결 유지 개수
db.SetMaxIdleConns(5)

// ConnMaxLifetime: 연결 최대 수명
db.SetConnMaxLifetime(5 * time.Minute)

// ConnMaxIdleTime: 유휴 연결 최대 시간
db.SetConnMaxIdleTime(5 * time.Minute)
```

### 문자 인코딩 관련
- **ENCODING**: UTF8 (모든 언어 지원)
- **LC_COLLATE**: C.UTF-8 (성능 우선, 유니코드 정렬)
- **LC_CTYPE**: C.UTF-8 (문자 분류)
- **이유**:
  - ko_KR.UTF-8은 Alpine Linux에서 기본 제공 안됨
  - C.UTF-8은 로케일 독립적이며 성능 우수
  - 한글 저장/조회에 문제 없음

## 데이터 모델

### 1. sites 테이블
Orbithall을 사용하는 사이트 정보 (멀티 테넌시)

```sql
CREATE TABLE sites (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,                -- 사이트 이름 (예: "코드버스 블로그")
    domain VARCHAR(255) NOT NULL UNIQUE,       -- 도메인 (예: "blog.codeverse.com")
    api_key UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),  -- API 인증 키
    cors_origins TEXT[] NOT NULL,              -- CORS 허용 도메인 배열
    is_active BOOLEAN DEFAULT TRUE,            -- 활성화 여부
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 인덱스
CREATE INDEX idx_sites_api_key ON sites(api_key);
CREATE INDEX idx_sites_domain ON sites(domain);
CREATE INDEX idx_sites_is_active ON sites(is_active) WHERE is_active = TRUE;
```

### 2. posts 테이블
블로그 포스트 메타데이터 (실제 내용은 Next.js에 있음)

```sql
CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    slug VARCHAR(255) NOT NULL,                -- 블로그 포스트 URL slug
    title VARCHAR(500) NOT NULL,               -- 포스트 제목
    comment_count INTEGER DEFAULT 0,           -- 댓글 수 (캐시)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- slug는 사이트 내에서만 unique
    UNIQUE(site_id, slug)
);

-- 인덱스
CREATE INDEX idx_posts_site_id ON posts(site_id);
CREATE INDEX idx_posts_site_slug ON posts(site_id, slug);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
```

### 3. comments 테이블
댓글 데이터

```sql
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES comments(id) ON DELETE CASCADE,  -- 대댓글용

    -- 작성자 정보 (간단한 인증)
    author_name VARCHAR(100) NOT NULL,     -- 이름
    author_password VARCHAR(255) NOT NULL, -- bcrypt 해시

    -- 댓글 내용
    content TEXT NOT NULL,                 -- 댓글 본문

    -- 메타데이터
    is_deleted BOOLEAN DEFAULT FALSE,      -- 소프트 삭제
    ip_address INET,                       -- IP 주소 (스팸 방지용)
    user_agent TEXT,                       -- User Agent (스팸 방지용)

    -- 타임스탬프
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 인덱스
CREATE INDEX idx_comments_post_id ON comments(post_id, created_at DESC);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);
CREATE INDEX idx_comments_is_deleted ON comments(is_deleted) WHERE is_deleted = FALSE;
```

### 관계도
```
sites (1) --- (*) posts
posts (1) --- (*) comments
comments (*) --- (1) comments (대댓글)
```

## API 영향도

### 데이터베이스 연결이 필요한 API
모든 API는 `X-Orbithall-API-Key` 헤더로 사이트 인증 필요

- `POST /api/posts/:slug/comments` - 댓글 작성 (site_id로 post 조회)
- `GET /api/posts/:slug/comments` - 댓글 목록 조회 (site_id로 필터링)
- `PUT /api/comments/:id` - 댓글 수정 (site_id 검증)
- `DELETE /api/comments/:id` - 댓글 삭제 (site_id 검증)

### 인증 플로우
```
1. 클라이언트 요청 → X-Orbithall-API-Key 헤더 포함
2. 서버: API 키로 sites 테이블 조회 (캐싱)
3. 서버: CORS origin 검증
4. 서버: site_id로 데이터 격리
```

## 환경변수

### 개발 환경
```env
DATABASE_URL=postgres://orbithall:dev_password@localhost:5432/orbithall_db?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

### 프로덕션 환경
```env
DATABASE_URL=postgres://orbithall:STRONG_PASSWORD@hostname:5432/orbithall_db?sslmode=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

## 제약사항 및 가정

### 제약사항
- Alpine Linux 사용으로 일부 로케일 제한
- 자동 마이그레이션 없음 (수동 실행)
- 현재는 단일 데이터베이스만 지원

### 가정
- PostgreSQL 16 이상 사용
- Docker 환경에서 실행
- 초당 100 요청 미만 (초기 단계)

## 의존성

### 외부 라이브러리
- `github.com/lib/pq`: PostgreSQL 드라이버 (Go 표준)
- 또는 `github.com/jackc/pgx/v5`: 고성능 드라이버 (선택)

### 내부 모듈
- `internal/database`: DB 연결 관리
- `internal/models`: 데이터 구조체

## 보안 고려사항

### 1. API 키 관리
- UUID v4 기반 API 키 자동 생성
- 환경변수로 관리 (클라이언트 측)
- 서버: 메모리 캐싱 (TTL 1분)으로 성능 최적화
- API 키 노출 시: DB에서 재발급 → 최대 1분 후 무효화

### 2. 데이터 격리
- 모든 쿼리에 site_id 필터링 필수
- API 키로 식별된 사이트의 데이터만 접근 가능
- Cross-site 접근 원천 차단

### 3. CORS 동적 검증
- 사이트별 cors_origins 배열 DB 저장
- 요청 Origin과 비교하여 허용/차단
- 캐싱으로 매 요청마다 DB 조회 방지

### 4. Credential 관리
- 환경변수로만 관리
- .env 파일은 .gitignore에 포함
- 프로덕션에서는 Railway Secrets 사용

### 5. SQL Injection 방지
- 모든 쿼리는 Prepared Statement 사용
- 사용자 입력은 절대 직접 SQL에 삽입하지 않음

### 6. 비밀번호 저장
- bcrypt 해시 (cost 12)
- 평문 비밀번호는 메모리에만 존재

### 7. IP 주소 저장
- GDPR/개인정보보호법 고려
- 스팸 방지 목적으로만 사용
- 해시 또는 일부만 저장 고려 (추후)

## 마이그레이션 전략

### 초기 스키마 생성
```bash
# SQL 파일 실행
docker exec -i orbithall-db psql -U orbithall -d orbithall_db < migrations/001_initial_schema.sql
```

### 향후 스키마 변경
- 버전 관리: `migrations/XXX_description.sql` 형식
- Up/Down 스크립트 분리 (추후)
- 프로덕션 적용 전 백업 필수

## 테스트 시나리오

### 정상 시나리오
1. **시나리오 1: DB 연결 성공**
   - 전제 조건: PostgreSQL 서비스 실행 중
   - 실행 단계: 애플리케이션 시작
   - 예상 결과: 연결 성공 로그, 에러 없음

2. **시나리오 2: 테이블 생성**
   - 전제 조건: 마이그레이션 SQL 파일 존재
   - 실행 단계: SQL 파일 실행
   - 예상 결과: sites, posts, comments 테이블 및 인덱스 생성

3. **시나리오 3: 사이트 등록**
   - 전제 조건: sites 테이블 생성 완료
   - 실행 단계: 사이트 정보 INSERT, API 키 자동 생성 확인
   - 예상 결과: UUID 형식 API 키 생성, UNIQUE 제약조건 작동

4. **시나리오 4: 한글 저장/조회**
   - 전제 조건: 테이블 생성 완료
   - 실행 단계: 한글 댓글 INSERT 후 SELECT
   - 예상 결과: 한글 깨짐 없이 정상 조회

5. **시나리오 5: 데이터 격리**
   - 전제 조건: 2개 사이트 등록, 각각 댓글 작성
   - 실행 단계: site_id로 필터링하여 조회
   - 예상 결과: 각 사이트의 댓글만 조회됨

### 예외 시나리오
1. **시나리오 1: DB 연결 실패**
   - 전제 조건: 잘못된 DATABASE_URL
   - 실행 단계: 애플리케이션 시작
   - 예상 결과: 명확한 에러 메시지, graceful 종료

2. **시나리오 2: 연결 풀 고갈**
   - 전제 조건: 모든 연결 사용 중
   - 실행 단계: 새로운 쿼리 실행
   - 예상 결과: Timeout 에러 또는 대기 후 실행

### 엣지 케이스
1. **케이스 1: 긴 댓글 (10KB+)**
   - 상황: TEXT 타입 한계 테스트
   - 처리 방법: 애플리케이션 레벨에서 길이 제한 (10KB)

2. **케이스 2: 특수문자 및 이모지**
   - 상황: UTF-8 4바이트 문자 (🔥 등)
   - 처리 방법: UTF-8 인코딩으로 정상 처리

## 마일스톤
- [x] Phase 1: 명세서 작성 (2025-10-02)
- [ ] Phase 2: DB 연결 패키지 구현 (예상일: 2025-10-02)
- [ ] Phase 3: 마이그레이션 SQL 작성 (예상일: 2025-10-02)
- [ ] Phase 4: 검증 및 테스트 (예상일: 2025-10-02)

## 참고 자료
- PostgreSQL Collation: https://www.postgresql.org/docs/16/collation.html
- Go database/sql: https://pkg.go.dev/database/sql
- pgx 드라이버: https://github.com/jackc/pgx
- bcrypt: https://pkg.go.dev/golang.org/x/crypto/bcrypt

## 변경 이력
| 날짜 | 버전 | 변경 내용 | 작성자 |
|------|------|-----------|--------|
| 2025-10-02 | v1.0 | 초안 작성 | Claude |
| 2025-10-02 | v1.1 | 멀티 테넌시 구조 추가 (sites 테이블, API 키 인증, CORS 동적 처리) | Claude |

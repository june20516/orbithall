# Site-User 연결 로직 구현

## 작성일
2025-10-29

## 우선순위
- [ ] 긴급
- [x] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
User와 Site의 다대다 관계를 관리하는 Repository 구현. 사용자가 여러 사이트를 소유하고 관리할 수 있도록 `user_sites` 테이블 CRUD 로직을 TDD 방식으로 개발.

## 작업 목적
- Admin 사용자가 자신이 소유한 사이트 목록을 조회할 수 있도록 함
- 사이트 생성 시 자동으로 사용자와 연결 (owner 권한)
- 향후 사이트 공유 기능 확장 가능하도록 기반 마련

## 작업 범위

### 포함 사항
- `user_sites` 테이블 CRUD Repository (TDD)
- Site 생성 로직 확장 (User와 자동 연결)
- 테스트 (연결/해제, 권한 확인, 중복 방지)

### 제외 사항
- Admin API 엔드포인트 (작업 012)
- 권한 레벨 세분화 (`owner` 외는 추후)
- Site CRUD API (작업 012)

## 기술적 접근

### 사용할 기술/라이브러리
- **database/sql**: 기존 Raw SQL 방식 유지
- **PostgreSQL**: user_sites 테이블 (작업 009에서 마이그레이션 완료)
- **lib/pq**: PostgreSQL Array 타입 처리

### 파일 구조
```
orbithall/
├── internal/
│   └── database/
│       ├── user_sites.go
│       ├── user_sites_test.go
│       ├── sites.go (CreateSiteForUser 추가)
│       └── sites_test.go
```

## 구현 단계

### 1. User-Site Repository 구현 (TDD)

**파일**: `internal/database/user_sites.go`, `internal/database/user_sites_test.go`

**구현 내용**:
- `AddUserToSite()`: 사용자를 사이트에 연결
  - INSERT INTO user_sites (user_id, site_id, role)
  - 복합 PK로 중복 방지
- `GetUserSites()`: 사용자의 사이트 목록 조회
  - JOIN sites ON user_sites.site_id = sites.id
  - ORDER BY created_at DESC
- `GetSiteUsers()`: 사이트의 사용자 목록 조회
  - JOIN users ON user_sites.user_id = users.id
- `RemoveUserFromSite()`: 사용자-사이트 연결 해제
  - DELETE FROM user_sites WHERE ...
- `IsUserSiteOwner()`: 사용자가 사이트 소유자인지 확인
  - SELECT EXISTS(...) WHERE role = 'owner'

**주요 로직**:
- 복합 PK (user_id, site_id)로 중복 연결 방지
- JOIN 쿼리로 N+1 문제 회피
- role 기본값: `owner`

**테스트 시나리오**:
- AddUserToSite 성공
- 중복 연결 시도 → PK 위반 에러
- GetUserSites 조회 (3개 사이트)
- GetSiteUsers 조회 (2명 사용자)
- RemoveUserFromSite 후 빈 목록 확인
- IsUserSiteOwner true/false 케이스
- 트랜잭션 기반 테스트

---

### 2. Site 생성 로직 확장

**파일**: `internal/database/sites.go`, `internal/database/sites_test.go`

**구현 내용**:
- `CreateSiteForUser()`: 사이트 생성 및 사용자 연결
  - `models.GenerateAPIKey("orb_live_")` 호출
  - INSERT INTO sites (name, domain, api_key, cors_origins, is_active)
  - `AddUserToSite(user_id, site_id, "owner")` 호출
  - 트랜잭션 내에서 두 작업 수행

**주요 로직**:
- 트랜잭션으로 원자성 보장
- RETURNING으로 Site ID 획득
- API Key 자동 생성
- owner 권한으로 자동 연결

**테스트 시나리오**:
- CreateSiteForUser 호출
- Site ID 설정 확인
- GetUserSites로 사이트 목록 확인
- IsUserSiteOwner로 소유자 확인

---

## 검증 방법

### 1. 테스트 실행
```bash
go test ./internal/database -run TestAddUserToSite
go test ./internal/database -run TestGetUserSites
go test ./internal/database -run TestGetSiteUsers
go test ./internal/database -run TestRemoveUserFromSite
go test ./internal/database -run TestIsUserSiteOwner
go test ./internal/database -run TestCreateSiteForUser
go test ./...
```

**예상 결과**: 모든 테스트 PASS

---

### 2. DB 검증
```sql
-- 사용자-사이트 연결 확인
SELECT * FROM user_sites;

-- 사용자가 소유한 사이트 목록 (JOIN)
SELECT s.name, s.domain, us.role
FROM sites s
INNER JOIN user_sites us ON s.id = us.site_id
WHERE us.user_id = 1;

-- 중복 연결 방지 확인
INSERT INTO user_sites (user_id, site_id, role) VALUES (1, 1, 'owner');
-- 예상: duplicate key 에러
```

---

### 3. 통합 시나리오 (작업 012 완료 후)
```bash
# 1. Google 로그인 (작업 009)
# 2. JWT로 내 사이트 목록 조회 (작업 012)
# 3. 사이트 생성 (작업 012)
#    → CreateSiteForUser 호출 → user_sites 자동 연결
```

---

## 의존성
- 선행 작업:
  - 009: User 모델 및 JWT
  - 010: JWT 미들웨어
- 후속 작업:
  - 012: Admin API 엔드포인트 (`CreateSiteForUser` 사용)

## 예상 소요 시간
- 예상: 2-3시간
- 실제: (완료 후 기록)

## 주의사항

### TDD 원칙
- ✅ 테스트 먼저 작성
- ✅ 트랜잭션 기반 테스트

### 데이터 무결성
- ✅ 복합 PK (user_id, site_id)로 중복 방지
- ✅ ON DELETE CASCADE
- ✅ role 기본값: `owner`

### 권한 관리
- ✅ 현재는 `owner` 권한만 사용
- ✅ 향후 `admin`, `editor` 등 확장 가능

### Repository 패턴
- ✅ 트랜잭션을 파라미터로 받음
- ✅ JOIN 쿼리로 N+1 문제 방지
- ✅ `pq.Array()`로 PostgreSQL TEXT[] 처리

## 참고 자료
- PostgreSQL Array: https://www.postgresql.org/docs/16/arrays.html
- lib/pq Array: https://pkg.go.dev/github.com/lib/pq#Array

---

## 작업 이력

### [2025-10-29] 작업 문서 작성
- User-Site 다대다 관계 Repository 설계
- Site 생성 시 자동 연결 로직

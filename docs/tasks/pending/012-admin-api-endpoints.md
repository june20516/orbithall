# Admin API 엔드포인트 구현

## 작성일
2025-10-29

## 우선순위
- [x] 긴급
- [ ] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
Admin 페이지에서 사용할 Site 관리 API 엔드포인트 구현. JWT 인증된 사용자가 자신의 사이트를 조회/수정/삭제할 수 있도록 TDD 방식으로 개발.

## 작업 목적
- Admin 사용자가 자신이 소유한 사이트를 관리할 수 있는 API 제공
- 사이트 생성, 조회, 수정, 삭제 기능
- 사이트 소유권 검증을 통한 안전한 접근 제어

## 작업 범위

### 포함 사항
- Admin 핸들러 구현 (TDD)
  - `GET /admin/sites`: 내 사이트 목록
  - `GET /admin/sites/:id`: 사이트 상세
  - `POST /admin/sites`: 사이트 생성
  - `PUT /admin/sites/:id`: 사이트 수정
  - `DELETE /admin/sites/:id`: 사이트 삭제
  - `GET /admin/profile`: 내 프로필
- 소유권 검증 로직
- 입력 검증 (name, domain, cors_origins)

### 제외 사항
- 사이트 공유 기능 (추후)
- 사이트 통계/분석 (추후)
- 사용자 프로필 수정 (추후)

## 기술적 접근

### 사용할 기술/라이브러리
- **Chi 라우터**: 이미 사용 중
- **database/sql**: 기존 방식 유지
- **작업 010의 JWTAuthMiddleware**: Context에서 사용자 추출

### 파일 구조
```
orbithall/
├── internal/
│   ├── handlers/
│   │   ├── admin.go
│   │   └── admin_test.go
│   └── validators/
│       ├── site.go
│       └── site_test.go
└── cmd/api/
    └── main.go
```

## API 명세

### 공통 사항
- **인증**: Authorization: Bearer {JWT}
- **에러 코드**:
  - `UNAUTHORIZED`: 인증 실패
  - `FORBIDDEN`: 권한 없음
  - `NOT_FOUND`: 사이트 없음
  - `INVALID_INPUT`: 입력 검증 실패

---

### 1. 내 사이트 목록 조회

#### `GET /admin/sites`

**응답 (200 OK)**
```json
{
  "sites": [
    {
      "id": 1,
      "name": "My Blog",
      "domain": "blog.example.com",
      "api_key": "orb_live_abc123...",
      "cors_origins": ["https://blog.example.com"],
      "is_active": true,
      "created_at": "2025-10-29T10:00:00Z",
      "updated_at": "2025-10-29T10:00:00Z"
    }
  ]
}
```

---

### 2. 사이트 상세 조회

#### `GET /admin/sites/:id`

**응답 (200 OK)**: 동일한 Site 객체

**에러 응답**:
- `404`: 사이트 없음 또는 소유하지 않음

---

### 3. 사이트 생성

#### `POST /admin/sites`

**요청 본문**
```json
{
  "name": "My New Blog",
  "domain": "newblog.example.com",
  "cors_origins": ["https://newblog.example.com"]
}
```

**응답 (201 Created)**: 생성된 Site 객체

**에러 응답**:
- `400`: 입력 검증 실패 (name, domain 필수, CORS origins 배열)

---

### 4. 사이트 수정

#### `PUT /admin/sites/:id`

**요청 본문**
```json
{
  "name": "Updated Name",
  "cors_origins": ["https://newdomain.com", "http://localhost:3000"],
  "is_active": false
}
```

**응답 (200 OK)**: 수정된 Site 객체

**에러 응답**:
- `403`: 소유하지 않은 사이트
- `404`: 사이트 없음

**주의**: `domain`, `api_key`는 수정 불가

---

### 5. 사이트 삭제

#### `DELETE /admin/sites/:id`

**응답 (204 No Content)**: 빈 응답

**에러 응답**:
- `403`: 소유하지 않은 사이트
- `404`: 사이트 없음

**주의**: 사이트 삭제 시 연결된 posts, comments도 CASCADE 삭제됨

---

### 6. 내 프로필 조회

#### `GET /admin/profile`

**응답 (200 OK)**
```json
{
  "id": 1,
  "email": "user@example.com",
  "name": "홍길동",
  "picture_url": "https://lh3.googleusercontent.com/...",
  "created_at": "2025-10-29T10:00:00Z"
}
```

---

## 구현 단계

### 1. Site 입력 검증 구현 (TDD)

**파일**: `internal/validators/site.go`, `internal/validators/site_test.go`

**구현 내용**:
- `SiteCreateInput`, `SiteUpdateInput` 구조체
- `ValidateSiteCreate()`: name(필수, 1-100자), domain(필수), cors_origins(배열, 각 URL 형식)
- `ValidateSiteUpdate()`: name(선택, 1-100자), cors_origins(선택), is_active(선택)

**테스트 시나리오**:
- name 누락 시 에러
- domain 누락 시 에러
- CORS origins 형식 오류 시 에러

---

### 2. Admin 핸들러 구현 (TDD)

**파일**: `internal/handlers/admin.go`, `internal/handlers/admin_test.go`

**구현 내용**:
- `AdminHandler` 구조체, `NewAdminHandler()` 생성자
- `ListSites()`:
  - Context에서 사용자 추출 (`GetUserFromContext`)
  - `database.GetUserSites()` 호출
  - 200 OK 응답
- `GetSite()`:
  - URL 파라미터에서 site_id 추출
  - `database.GetSiteByID()` 호출
  - `database.IsUserSiteOwner()` 확인
  - 소유자 아니면 404 반환
  - 200 OK 응답
- `CreateSite()`:
  - JSON 요청 본문 파싱
  - 입력 검증 (`validators.ValidateSiteCreate`)
  - `database.CreateSiteForUser()` 호출 (작업 011에서 구현)
  - 201 Created 응답
- `UpdateSite()`:
  - site_id 추출, 입력 검증
  - 소유권 확인 (`IsUserSiteOwner`)
  - `database.UpdateSite()` 호출
  - 200 OK 응답
- `DeleteSite()`:
  - site_id 추출, 소유권 확인
  - `database.DeleteSite()` 호출
  - 204 No Content 응답
- `GetProfile()`:
  - Context에서 사용자 추출
  - 사용자 정보 반환

**주요 로직**:
- 모든 핸들러는 JWT 미들웨어 통과 후 호출
- Context에서 사용자 정보 추출
- 소유권 검증 필수 (수정/삭제 시)
- 트랜잭션 관리

**테스트 시나리오**:
- ListSites 성공 (3개 사이트)
- GetSite 성공 (소유자)
- GetSite 실패 (소유하지 않음) → 404
- CreateSite 성공 → 201
- CreateSite 입력 오류 → 400
- UpdateSite 성공 → 200
- UpdateSite 소유자 아님 → 403
- DeleteSite 성공 → 204
- DeleteSite 소유자 아님 → 403
- GetProfile 성공 → 200

---

### 3. Database 메서드 추가

**파일**: `internal/database/sites.go`

**구현 내용**:
- `GetSiteByID()`: ID로 사이트 조회 (이미 있을 수 있음, 확인 필요)
- `UpdateSite()`: name, cors_origins, is_active 수정
- `DeleteSite()`: 사이트 삭제 (CASCADE로 연결 데이터 자동 삭제)

---

### 4. 라우팅 설정

**파일**: `cmd/api/main.go`

**추가 내용**:
- `adminHandler := handlers.NewAdminHandler(db)` 초기화
- `/admin` 그룹에 엔드포인트 등록:
  - `GET /sites`
  - `GET /sites/:id`
  - `POST /sites`
  - `PUT /sites/:id`
  - `DELETE /sites/:id`
  - `GET /profile`

---

## 검증 방법

### 1. 테스트 실행
```bash
go test ./internal/validators -run TestValidateSiteCreate
go test ./internal/handlers -run TestListSites
go test ./internal/handlers -run TestCreateSite
go test ./internal/handlers -run TestUpdateSite
go test ./internal/handlers -run TestDeleteSite
go test ./...
```

**예상 결과**: 모든 테스트 PASS

---

### 2. API 수동 테스트

**사전 준비**: JWT 발급
```bash
RESPONSE=$(curl -s -X POST http://localhost:8080/auth/google/verify \
  -H "Content-Type: application/json" \
  -d '{"id_token": "실제_Token", "email": "test@example.com", "name": "Test"}')
JWT=$(echo $RESPONSE | jq -r '.token')
```

**시나리오 1: 내 사이트 목록 조회**
```bash
curl -X GET http://localhost:8080/admin/sites \
  -H "Authorization: Bearer $JWT"
# 예상: 200 OK, 사이트 목록
```

**시나리오 2: 사이트 생성**
```bash
curl -X POST http://localhost:8080/admin/sites \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My New Blog",
    "domain": "newblog.example.com",
    "cors_origins": ["https://newblog.example.com"]
  }'
# 예상: 201 Created, API Key 포함
```

**시나리오 3: 사이트 수정**
```bash
curl -X PUT http://localhost:8080/admin/sites/1 \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Name",
    "cors_origins": ["https://newdomain.com"],
    "is_active": false
  }'
# 예상: 200 OK
```

**시나리오 4: 사이트 삭제**
```bash
curl -X DELETE http://localhost:8080/admin/sites/1 \
  -H "Authorization: Bearer $JWT"
# 예상: 204 No Content
```

**시나리오 5: 프로필 조회**
```bash
curl -X GET http://localhost:8080/admin/profile \
  -H "Authorization: Bearer $JWT"
# 예상: 200 OK, 사용자 정보
```

**시나리오 6: 소유하지 않은 사이트 접근 → 403**
```bash
curl -X PUT http://localhost:8080/admin/sites/999 \
  -H "Authorization: Bearer $JWT" \
  -d '{}'
# 예상: 404 Not Found (또는 403 Forbidden)
```

---

## 의존성
- 선행 작업:
  - 009: User, JWT, Google OAuth
  - 010: JWT 미들웨어
  - 011: Site-User 연결 로직
- 후속 작업: 없음 (이 작업으로 Admin 기능 완성)

## 예상 소요 시간
- 예상: 4-5시간
- 실제: (완료 후 기록)

## 주의사항

### TDD 원칙
- ✅ 테스트 먼저 작성
- ✅ 트랜잭션 기반 테스트

### 권한 검증
- ✅ 모든 수정/삭제는 소유권 확인 필수
- ✅ `IsUserSiteOwner()` 사용
- ✅ 소유하지 않은 사이트 접근 시 404 또는 403

### 입력 검증
- ✅ name, domain 필수
- ✅ CORS origins URL 형식 확인
- ✅ domain, api_key는 수정 불가

### CASCADE 삭제
- ✅ Site 삭제 시 posts, comments 자동 삭제
- ✅ user_sites 레코드도 자동 삭제

### 보안
- ✅ API Key는 응답에 포함 (사이트 소유자만 볼 수 있음)
- ✅ JWT 인증 필수 (미들웨어에서 처리)

## 참고 자료
- Chi URL Params: https://github.com/go-chi/chi#url-parameters
- Context Pattern: https://pkg.go.dev/context

---

## 작업 이력

### [2025-10-29] 작업 문서 작성
- Admin API 엔드포인트 명세
- Site 관리 CRUD 기능 정의
- 소유권 검증 로직 설계

# [WIP] 댓글 CRUD API 구현

## 작성일
2025-10-14

## 우선순위
- [x] 긴급
- [ ] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
댓글 시스템의 핵심 기능인 댓글 생성, 조회, 수정, 삭제 API 엔드포인트 구현

## 작업 목적
블로그 사이트에서 댓글을 작성하고 관리할 수 있는 REST API 제공. API 키 기반 인증으로 멀티 테넌시 지원.

## 작업 범위

### 포함 사항
- 댓글 생성 API (POST /api/comments)
- 댓글 목록 조회 API (GET /api/comments)
- 댓글 수정 API (PUT /api/comments/:id)
- 댓글 삭제 API (DELETE /api/comments/:id)
- API 키 인증 미들웨어
- CORS 검증
- 입력 검증 (길이 제한, 필수 필드 등)
- 대댓글(중첩 댓글) 지원

### 제외 사항
- 고급 스팸 필터링 (추후)
- 실시간 알림 (추후)
- 파일 첨부 (추후)

## 기술적 접근

### 사용할 기술/라이브러리
- Chi 라우터 (이미 사용 중)
- database/sql (이미 설정됨)
- bcrypt (비밀번호 해싱)
- bluemonday (HTML sanitization, XSS 방어)
- 기존 models, database 패키지 활용

### 파일 구조
```
orbithall/
├── internal/
│   ├── handlers/
│   │   ├── comments.go          # 댓글 CRUD 핸들러
│   │   └── middleware.go        # API 키 인증 미들웨어
│   ├── validators/
│   │   └── comment.go           # 입력 검증
│   └── sanitizer/
│       └── html.go              # HTML sanitization (XSS 방어)
└── cmd/api/
    └── main.go                   # 라우팅 추가
```

## API 명세

### 공통 사항

#### 인증 헤더
모든 API 요청에 필수:
```
X-Orbithall-API-Key: {site의 api_key UUID}
```

#### 공통 에러 응답 포맷
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {} // 선택적, 추가 정보
  }
}
```

#### HTTP 상태 코드
- `200 OK`: 성공 (조회, 수정, 삭제)
- `201 Created`: 생성 성공
- `400 Bad Request`: 잘못된 요청 (입력 검증 실패)
- `401 Unauthorized`: 인증 정보 없음 (API 키 누락)
- `403 Forbidden`: 권한 없음 (잘못된 API 키, 비활성 사이트, 비밀번호 불일치)
- `404 Not Found`: 리소스 없음
- `429 Too Many Requests`: Rate limit 초과
- `500 Internal Server Error`: 서버 오류

#### 에러 코드 정의
```go
const (
    ErrMissingAPIKey      = "MISSING_API_KEY"       // API 키 헤더 없음
    ErrInvalidAPIKey      = "INVALID_API_KEY"       // API 키 형식 오류 또는 존재하지 않음
    ErrSiteInactive       = "SITE_INACTIVE"         // 비활성화된 사이트
    ErrInvalidInput       = "INVALID_INPUT"         // 입력 검증 실패
    ErrPostNotFound       = "POST_NOT_FOUND"        // 포스트 없음
    ErrCommentNotFound    = "COMMENT_NOT_FOUND"     // 댓글 없음
    ErrWrongPassword      = "WRONG_PASSWORD"        // 비밀번호 불일치
    ErrEditTimeExpired    = "EDIT_TIME_EXPIRED"     // 수정 가능 시간 초과
    ErrRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"   // Rate limit 초과
    ErrInternalServer     = "INTERNAL_SERVER_ERROR" // 서버 내부 오류
)
```

---

### 1. 댓글 생성

#### `POST /api/posts/:slug/comments`

**요청 헤더**
```
X-Orbithall-API-Key: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Content-Type: application/json
```

**요청 본문**
```json
{
  "author_name": "홍길동",
  "password": "mypassword123",
  "content": "좋은 글 잘 읽었습니다!",
  "parent_id": null  // 선택, 대댓글인 경우 부모 댓글 ID
}
```

**요청 필드 검증**
- `author_name`: 필수, 1-100자, 공백 제외
- `password`: 필수, 4-50자
- `content`: 필수, 1-10000자 (약 10KB)
- `parent_id`: 선택, 양의 정수, 존재하는 댓글 ID

**성공 응답 (201 Created)**
```json
{
  "id": 123,
  "post_id": 456,
  "parent_id": null,
  "author_name": "홍길동",
  "content": "좋은 글 잘 읽었습니다!",
  "ip_address_masked": "192.168.***.***",
  "is_deleted": false,
  "created_at": "2025-10-15T14:30:00Z",
  "updated_at": "2025-10-15T14:30:00Z",
  "deleted_at": null
}
```

**실패 응답 예시**
```json
// 400: 입력 검증 실패
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Validation failed",
    "details": {
      "author_name": "이름은 1-100자여야 합니다",
      "content": "내용은 필수입니다"
    }
  }
}

// 404: 포스트 없음
{
  "error": {
    "code": "POST_NOT_FOUND",
    "message": "Post with slug 'my-post' not found"
  }
}
```

---

### 2. 댓글 목록 조회

#### `GET /api/posts/:slug/comments`

**요청 헤더**
```
X-Orbithall-API-Key: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

**쿼리 파라미터**
- `page`: 페이지 번호 (기본값: 1, 최소: 1)
- `limit`: 페이지당 개수 (기본값: 50, 최대: 100)

**성공 응답 (200 OK)**
```json
{
  "comments": [
    {
      "id": 123,
      "post_id": 456,
      "parent_id": null,
      "author_name": "홍길동",
      "content": "좋은 글 잘 읽었습니다!",
      "ip_address_masked": "192.168.***.***",
      "is_deleted": false,
      "created_at": "2025-10-15T14:30:00Z",
      "updated_at": "2025-10-15T14:30:00Z",
      "deleted_at": null,
      "replies": [
        {
          "id": 124,
          "post_id": 456,
          "parent_id": 123,
          "author_name": "작성자",
          "content": "감사합니다!",
          "ip_address_masked": "203.0.113.***.***",
          "is_deleted": false,
          "created_at": "2025-10-15T15:00:00Z",
          "updated_at": "2025-10-15T15:00:00Z",
          "deleted_at": null,
          "replies": []
        }
      ]
    },
    {
      "id": 125,
      "post_id": 456,
      "parent_id": null,
      "author_name": "[삭제됨]",
      "content": "삭제된 댓글입니다",
      "ip_address_masked": "10.0.***.***",
      "is_deleted": true,
      "created_at": "2025-10-15T16:00:00Z",
      "updated_at": "2025-10-15T16:00:00Z",
      "deleted_at": "2025-10-15T17:00:00Z",
      "replies": []
    }
  ],
  "pagination": {
    "current_page": 1,
    "total_pages": 3,
    "total_comments": 120,
    "per_page": 50
  }
}
```

**삭제된 댓글 처리 규칙**
- `is_deleted = true`인 댓글:
  - 대댓글이 있는 경우: 구조 유지, `author_name = "[삭제됨]"`, `content = "삭제된 댓글입니다"`
  - 대댓글이 없는 경우: 완전 숨김 (목록에서 제외)

---

### 3. 댓글 수정

#### `PUT /api/comments/:id`

**요청 헤더**
```
X-Orbithall-API-Key: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Content-Type: application/json
```

**요청 본문**
```json
{
  "password": "mypassword123",
  "content": "수정된 내용입니다"
}
```

**요청 필드 검증**
- `password`: 필수
- `content`: 필수, 1-10000자

**수정 제한 사항**
- 작성 후 30분 이내만 수정 가능
- `content`만 수정 가능 (`author_name` 불가)
- 삭제된 댓글은 수정 불가

**수정 시 추가 처리**
- 수정 요청 시점의 IP 주소가 작성 시와 다르면 IP 주소 업데이트
- User-Agent도 함께 업데이트
- 동일 사용자가 다른 환경(모바일/PC)에서 수정 가능

**성공 응답 (200 OK)**
```json
{
  "id": 123,
  "post_id": 456,
  "parent_id": null,
  "author_name": "홍길동",
  "content": "수정된 내용입니다",
  "ip_address_masked": "192.168.***.***",
  "is_deleted": false,
  "created_at": "2025-10-15T14:30:00Z",
  "updated_at": "2025-10-15T14:45:00Z",
  "deleted_at": null
}
```

**실패 응답 예시**
```json
// 403: 비밀번호 불일치
{
  "error": {
    "code": "WRONG_PASSWORD",
    "message": "Password does not match"
  }
}

// 403: 수정 시간 초과
{
  "error": {
    "code": "EDIT_TIME_EXPIRED",
    "message": "Comments can only be edited within 30 minutes of creation"
  }
}
```

---

### 4. 댓글 삭제

#### `DELETE /api/comments/:id`

**요청 헤더**
```
X-Orbithall-API-Key: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Content-Type: application/json
```

**요청 본문**
```json
{
  "password": "mypassword123"
}
```

**삭제 동작**
- Soft delete: `is_deleted = true`, `deleted_at = NOW()`
- `posts.comment_count` 1 감소
- 대댓글이 있는 경우: 구조 유지 (조회 시 "[삭제됨]" 표시)
- 대댓글이 없는 경우: 숨김 처리

**성공 응답 (200 OK)**
```json
{
  "message": "Comment deleted successfully",
  "id": 123
}
```

**실패 응답 예시**
```json
// 403: 비밀번호 불일치
{
  "error": {
    "code": "WRONG_PASSWORD",
    "message": "Password does not match"
  }
}

// 404: 댓글 없음
{
  "error": {
    "code": "COMMENT_NOT_FOUND",
    "message": "Comment not found or already deleted"
  }
}
```

---

## 구현 단계

### 1. HTML Sanitization 패키지 구현
**파일**: `internal/sanitizer/html.go`

**구현 내용**:
- bluemonday 기반 HTML sanitizer 초기화
- `StrictPolicy`: 모든 HTML 태그 제거 (댓글용)
- `SanitizeComment(content string) string`: 댓글 내용 sanitize
- 허용 태그 없음 (순수 텍스트만)
- 악성 스크립트, HTML 태그 완전 제거

---

### 2. 입력 검증 패키지 구현
**파일**: `internal/validators/comment.go`

**구현 내용**:
- `CommentCreateInput`, `CommentUpdateInput`, `CommentDeleteInput` 구조체 정의
- `ValidationErrors` 타입 (map[string]string) 및 Error() 메서드
- `ValidateCommentCreate()`: author_name(1-100자), password(4-50자), content(1-10000자), parent_id 검증
- `ValidateCommentUpdate()`: password, content 검증
- `ValidateCommentDelete()`: password 검증
- 에러 메시지는 영어로만 작성

---

### 3. 미들웨어 구현
**파일**: `internal/handlers/middleware.go`

**구현 내용**:
- `AuthMiddleware()`: API 키 검증 → 사이트 조회(캐시) → 활성화 확인 → CORS 검증 → 컨텍스트 저장
- `RateLimitMiddleware()`: IP 기반 rate limiting (메모리 기반, sliding window)
- `GetSiteFromContext()`: 컨텍스트에서 사이트 정보 추출 헬퍼
- `isOriginAllowed()`: CORS origin 배열 검증 (대소문자 무시)
- `respondError()`, `respondJSON()`: 일관된 JSON 응답 헬퍼

**중요 사항**:
- `database.GetSiteByAPIKey()`는 내부적으로 1분 TTL 캐시 사용
- 모든 API 엔드포인트는 이 미들웨어를 거쳐야 함
- Rate limiting은 엔드포인트별로 다른 제한 적용 가능

---

### 4. 댓글 CRUD 핸들러 구현
**파일**: `internal/handlers/comments.go`

**구현 내용**:
- `CommentHandler` 구조체 및 `NewCommentHandler()` 생성자
- `CreateComment()`:
  - 사이트 추출 → **HTML sanitization** → 입력 검증 → post 조회/생성
  - **1depth 검증**: `parent_id`가 있으면 부모 댓글 조회 후 `parent_id` 확인
  - 부모가 대댓글이면 에러 반환 (400)
  - 비밀번호 해싱 → 댓글 생성
- `ListComments()`: 쿼리 파라미터 파싱 → post 조회 → 댓글 목록 조회(2단계 구조) → 페이지네이션 응답
- `UpdateComment()`:
  - 댓글 조회 → 사이트 격리 확인 → 비밀번호 확인 → 30분 시간 제한
  - **HTML sanitization** → content 수정
  - **IP/User-Agent 업데이트** (요청 시점의 값으로)
- `DeleteComment()`: 댓글 조회 → 사이트 격리 확인 → 비밀번호 확인 → soft delete
- `getIntQuery()`, `getIPAddress()` 헬퍼 함수

**주요 로직**:
- 모든 핸들러는 사이트 격리 확인 필수 (`post.SiteID != site.ID`)
- **생성/수정 시 모든 content를 sanitizer로 처리** (XSS 방어)
- bcrypt로 비밀번호 검증 (`bcrypt.CompareHashAndPassword`)
- IP 주소는 X-Forwarded-For → X-Real-IP → RemoteAddr 순 우선
- 수정 제한: 작성 후 30분 (`EditTimeLimit`)
- **수정 시 IP/User-Agent 업데이트**: 요청 시점의 값으로 덮어씀
- **대댓글 깊이 검증**: 부모가 대댓글이면 생성 차단

---

### 5. 데이터베이스 메서드 구현
**파일**: `internal/database/comments.go` (새로 생성)

**구현 내용**:
- `CreateComment()`:
  - `parent_id`가 있으면 부모 댓글 조회하여 `parent_id` 확인 (1depth 제한)
  - INSERT ... RETURNING으로 ID, 타임스탬프 반환
- `ListComments()`: 최상위 댓글 조회(페이지네이션) + 각 댓글의 대댓글 조회(1단계만)
- `getReplies()`: 대댓글 단순 조회 (비공개 메서드, 재귀 없음)
- `GetCommentByID()`: ID로 단일 댓글 조회 (author_password 포함)
- `UpdateComment()`: content, ip_address, user_agent, updated_at 수정
- `DeleteComment()`: is_deleted=TRUE, deleted_at=NOW() 설정
- `scanComment()`: DB row → Comment 구조체 변환 헬퍼

**주의사항**:
- 생성 시 깊이 검증: 부모가 대댓글이면 생성 거부
- 수정 시 IP/User-Agent 업데이트 포함
- 삭제된 댓글은 대댓글 유무에 따라 처리 로직 추가

---

### 6. 라우팅 설정
**파일**: `cmd/api/main.go`

**추가 내용**:
- `handlers.NewCommentHandler(db)` 초기화
- CORS 미들웨어 설정 (AllowedOrigins: "*", 미들웨어에서 동적 검증)
- `/api` 그룹에 `AuthMiddleware` 적용
- 4개 엔드포인트 등록:
  - `POST /posts/{slug}/comments`
  - `GET /posts/{slug}/comments`
  - `PUT /comments/{id}`
  - `DELETE /comments/{id}`

---

### 7. Comment 모델 확장
**파일**: `internal/models/comment.go`

**추가 내용**:
- `Replies []Comment` 필드 추가 (`json:"replies,omitempty"`)
- `IPAddressMasked string` 필드 추가 (`json:"ip_address_masked,omitempty"`)
- IP 마스킹 헬퍼 함수: `MaskIPAddress(ip string) string`
  - IPv4: 앞 2 옥텟만 표시 (예: `192.168.***.***`)
  - IPv6: 앞 4개 그룹만 표시 (예: `2001:0db8:****:****:****:****:****:****`)
- 조회 시 `IPAddressMasked` 필드를 설정하여 반환

## 검증 방법

### 1. 사전 준비
```bash
# 1. 테스트용 사이트 등록 (DB 직접 조작)
docker exec -it orbithall-db psql -U orbithall -d orbithall_db

INSERT INTO sites (name, domain, cors_origins, is_active)
VALUES ('테스트 블로그', 'localhost', ARRAY['http://localhost:3000'], TRUE)
RETURNING id, api_key;
-- 출력된 api_key를 아래 {API_KEY}로 사용

# 2. 환경변수 설정
export API_KEY="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export BASE_URL="http://localhost:8080"
```

### 2. 테스트 시나리오 (curl 명령어)

#### 시나리오 1: 댓글 생성 (성공)
```bash
curl -X POST "$BASE_URL/api/posts/my-first-post/comments" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "author_name": "홍길동",
    "password": "test1234",
    "content": "좋은 글 잘 읽었습니다!"
  }'

# 예상 응답: 201 Created
# {
#   "id": 1,
#   "post_id": 1,
#   "parent_id": null,
#   "author_name": "홍길동",
#   "content": "좋은 글 잘 읽었습니다!",
#   "is_deleted": false,
#   "created_at": "2025-10-15T14:30:00Z",
#   "updated_at": "2025-10-15T14:30:00Z",
#   "deleted_at": null
# }
```

#### 시나리오 2: 댓글 생성 (입력 검증 실패)
```bash
curl -X POST "$BASE_URL/api/posts/my-first-post/comments" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "author_name": "",
    "password": "123",
    "content": ""
  }'

# 예상 응답: 400 Bad Request
# {
#   "error": {
#     "code": "INVALID_INPUT",
#     "message": "Validation failed",
#     "details": {
#       "author_name": "이름은 필수입니다",
#       "password": "비밀번호는 4자 이상이어야 합니다",
#       "content": "내용은 필수입니다"
#     }
#   }
# }
```

#### 시나리오 3: 대댓글 생성
```bash
# 부모 댓글 ID를 1로 가정
curl -X POST "$BASE_URL/api/posts/my-first-post/comments" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "author_name": "작성자",
    "password": "reply123",
    "content": "감사합니다!",
    "parent_id": 1
  }'

# 예상 응답: 201 Created
```

#### 시나리오 4: 댓글 목록 조회
```bash
curl -X GET "$BASE_URL/api/posts/my-first-post/comments?page=1&limit=50" \
  -H "X-Orbithall-API-Key: $API_KEY"

# 예상 응답: 200 OK (트리 구조)
# {
#   "comments": [
#     {
#       "id": 1,
#       "author_name": "홍길동",
#       "content": "좋은 글 잘 읽었습니다!",
#       "replies": [
#         {
#           "id": 2,
#           "author_name": "작성자",
#           "content": "감사합니다!",
#           "replies": []
#         }
#       ]
#     }
#   ],
#   "pagination": { ... }
# }
```

#### 시나리오 5: 댓글 수정 (성공)
```bash
curl -X PUT "$BASE_URL/api/comments/1" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "test1234",
    "content": "수정된 내용입니다"
  }'

# 예상 응답: 200 OK
```

#### 시나리오 6: 댓글 수정 (비밀번호 불일치)
```bash
curl -X PUT "$BASE_URL/api/comments/1" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "wrongpassword",
    "content": "수정 시도"
  }'

# 예상 응답: 403 Forbidden
# {
#   "error": {
#     "code": "WRONG_PASSWORD",
#     "message": "Password does not match"
#   }
# }
```

#### 시나리오 7: 댓글 수정 (30분 초과)
```bash
# 30분 이상 지난 댓글 수정 시도
curl -X PUT "$BASE_URL/api/comments/1" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "test1234",
    "content": "30분 후 수정 시도"
  }'

# 예상 응답: 403 Forbidden
# {
#   "error": {
#     "code": "EDIT_TIME_EXPIRED",
#     "message": "Comments can only be edited within 30 minutes of creation"
#   }
# }
```

#### 시나리오 8: 댓글 삭제 (성공)
```bash
curl -X DELETE "$BASE_URL/api/comments/1" \
  -H "X-Orbithall-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "test1234"
  }'

# 예상 응답: 200 OK
# {
#   "message": "Comment deleted successfully",
#   "id": 1
# }
```

#### 시나리오 9: 삭제된 댓글 조회
```bash
curl -X GET "$BASE_URL/api/posts/my-first-post/comments" \
  -H "X-Orbithall-API-Key: $API_KEY"

# 예상 응답: 200 OK
# - 대댓글이 있는 경우: author_name="[삭제됨]", content="삭제된 댓글입니다"
# - 대댓글이 없는 경우: 목록에서 제외
```

#### 시나리오 10: 인증 실패 (API 키 없음)
```bash
curl -X GET "$BASE_URL/api/posts/my-first-post/comments"

# 예상 응답: 401 Unauthorized
# {
#   "error": {
#     "code": "MISSING_API_KEY",
#     "message": "API key is required"
#   }
# }
```

#### 시나리오 11: 인증 실패 (잘못된 API 키)
```bash
curl -X GET "$BASE_URL/api/posts/my-first-post/comments" \
  -H "X-Orbithall-API-Key: invalid-key-123"

# 예상 응답: 403 Forbidden
# {
#   "error": {
#     "code": "INVALID_API_KEY",
#     "message": "Invalid API key"
#   }
# }
```

### 3. DB 검증

#### 비밀번호 해싱 확인
```sql
-- 댓글 조회
SELECT id, author_name, author_password, content
FROM comments
WHERE id = 1;

-- author_password가 bcrypt 해시 형식인지 확인 (예: $2a$10$...)
```

#### Soft Delete 확인
```sql
-- 삭제된 댓글 확인
SELECT id, is_deleted, deleted_at
FROM comments
WHERE id = 1;

-- is_deleted = TRUE, deleted_at에 타임스탬프 있는지 확인
```

#### 사이트 격리 확인
```sql
-- 다른 사이트 등록
INSERT INTO sites (name, domain, cors_origins, is_active)
VALUES ('다른 블로그', 'other.com', ARRAY['http://other.com'], TRUE)
RETURNING api_key;

-- 다른 사이트 API 키로 첫 번째 사이트의 댓글 조회 시도
-- 결과: 빈 목록 또는 403 에러
```

### 4. 성능 테스트 (선택)

```bash
# Apache Bench로 동시 요청 테스트
ab -n 100 -c 10 \
  -H "X-Orbithall-API-Key: $API_KEY" \
  "$BASE_URL/api/posts/my-first-post/comments"

# 기대 결과:
# - 평균 응답 시간: < 100ms
# - 실패율: 0%
# - DB 연결 풀 정상 작동
```

## 의존성
- 선행 작업: 데이터베이스 연결 및 스키마 구현 (완료)
- 후속 작업: Rate Limiting 구현

## 예상 소요 시간
- 예상: 4-6시간

## 대댓글 처리 전략

### 1. 데이터 구조
- **트리 구조**: `parent_id`를 통한 자기 참조 관계
- **깊이 제한**: **1depth만 허용** (댓글 → 대댓글, 대댓글의 대댓글은 불가)
- **정렬 순서**: 같은 레벨 내에서는 `created_at ASC` (오래된 순)

### 2. 조회 방식
**단순 2단계 조회 (권장)**
```
1. 최상위 댓글 조회 (parent_id IS NULL) + 페이지네이션
2. 각 댓글의 대댓글 조회 (parent_id = 댓글 ID)
3. Go 코드에서 구조 조립
```
- 장점: 간단한 쿼리, 페이지네이션 용이, 재귀 불필요
- 주의: N+1 문제 방지를 위해 필요시 IN 절로 일괄 조회

### 3. 깊이 제한 구현
**생성 시 검증**
- 댓글 생성 시 `parent_id`가 있으면, 해당 댓글의 `parent_id`를 확인
- 부모 댓글이 이미 대댓글(parent_id != NULL)이면 생성 거부
- 에러: `INVALID_INPUT`, "대댓글에는 답글을 달 수 없습니다"

**조회 시 처리**
- 최상위 댓글만 페이지네이션
- 각 댓글의 대댓글은 전체 조회 (깊이 1이므로 성능 문제 없음)

### 4. 삭제된 댓글 처리
- **대댓글이 있는 경우**: 구조 유지, `author_name = "[삭제됨]"`, `content = "삭제된 댓글입니다"`
- **대댓글이 없는 경우**: 목록에서 완전히 제외

### 5. 주의사항
- **깊이 검증**: 생성 시 부모 댓글의 `parent_id` 확인 필수
- **순환 참조 방지**: `parent_id`가 자기 자신을 가리키지 않도록 검증
- **고아 댓글 방지**: 부모 댓글 삭제 시 `ON DELETE CASCADE`로 자동 처리 (DB 스키마에 이미 정의됨)

---

## 보안 고려사항

### 1. 비밀번호 보안
- **해싱**: bcrypt (cost 12)
- **저장**: 평문 비밀번호는 메모리에만 존재, DB에는 해시만 저장
- **응답**: JSON 응답에 절대 포함하지 않음 (`json:"-"` 태그)
- **검증**: bcrypt.CompareHashAndPassword() 사용
- **제한**: 최소 4자 (프론트엔드에서는 8자 이상 권장)

### 2. SQL Injection 방지
- **Prepared Statement**: 모든 쿼리에 파라미터 바인딩 (`$1`, `$2` 사용)
- **문자열 직접 삽입 금지**: `fmt.Sprintf()`로 쿼리 조립 절대 금지
- **사용자 입력 검증**: 길이 제한 및 타입 체크

### 2-1. 기술 스택 선택
- **database/sql 사용**: 프로젝트 규모가 작고 쿼리 제어가 중요하므로 ORM 없이 구현
- **장점**: 명확한 쿼리, 낮은 학습 곡선, 높은 성능, prepared statement로 안전

### 3. XSS 방어
- **HTML Sanitization**: bluemonday로 모든 HTML 태그 제거 (순수 텍스트만 허용)
- **적용 시점**: 댓글 생성 및 수정 시 content 처리
- **정책**: StrictPolicy (모든 태그 제거, 스크립트 차단)
- **입력 검증**: 길이 제한, 필수 필드 체크
- **출력 이스케이핑**: JSON 인코딩 자동 처리 (Go의 encoding/json)

### 4. 사이트 격리 (멀티 테넌시)
- **API 키 검증**: 모든 요청에 미들웨어 적용
- **site_id 필터링**: 모든 쿼리에 강제 포함
- **교차 접근 차단**: 댓글 수정/삭제 시 `post.SiteID != site.ID` 확인 필수

### 5. CORS 동적 검증
- **Origin 체크**: 요청 Origin이 사이트의 cors_origins 배열에 포함되는지 확인
- **캐싱**: 사이트 정보를 1분간 메모리 캐시 (TTL)
- **대소문자 무시**: strings.EqualFold() 사용

### 6. IP 주소 및 User-Agent 저장
- **저장 목적**: 스팸 방지, 어뷰징 탐지, 사칭 방지
- **저장 방식**: 전체 IP 주소 및 User-Agent 문자열 저장
- **응답 시 처리**:
  - 댓글 조회 시 IP 주소 부분 마스킹하여 반환 (예: `192.168.***.***`)
  - 익명 댓글 시스템에서 사칭 방지를 위한 최소한의 공개 정보
  - User-Agent는 응답에 포함하지 않음
- **GDPR 고려**: 보존 기간 설정 (90일 후 삭제, 추후 구현)

### 7. Rate Limiting
- **구현 범위**: 이번 작업에 포함
- **제한 규칙**:
  - IP 기반: 1분당 10개 댓글 생성
  - 사이트 기반: 초당 50 요청
  - 동일 IP의 수정/삭제: 초당 5 요청
- **구현 방식**: 메모리 기반 (sliding window 또는 token bucket)
- **에러 응답**: 429 Too Many Requests, `RATE_LIMIT_EXCEEDED`

### 8. 에러 정보 노출 제한
- **에러 메시지 언어**: 영어로만 제공
- **구현 위치**: 각 핸들러 및 서비스 레이어에서 영어 메시지 직접 반환
- **프로덕션**: 상세 에러는 로그에만 기록, 사용자에게는 일반 메시지
- **원칙**: DB 에러, 스택 트레이스 등 내부 정보는 절대 API 응답에 포함하지 않음
- **구현**: 에러 발생 시 log.Printf()로 기록 후 일반 메시지만 반환

### 9. 타이밍 공격 방지
- **비밀번호 비교**: bcrypt가 자동으로 처리 (constant-time comparison)
- **API 키 비교**: UUID 문자열 비교는 Go의 == 연산자 사용 (constant-time 아님, 추후 개선 고려)

### 10. HTTPS 강제 (프로덕션)
- Railway 배포 시 자동 HTTPS
- 로컬에서는 HTTP 허용

---

## 주의사항

### 필수 준수 사항
- ✅ 비밀번호는 절대 응답에 포함하지 않음
- ✅ IP 주소 부분 마스킹하여 반환 (사칭 방지)
- ✅ User-Agent는 비공개
- ✅ SQL Injection 방어 (prepared statement)
- ✅ XSS 방어 (입력 검증)
- ✅ **대댓글 1depth 제한** (댓글의 대댓글만 가능)
- ✅ 사이트 간 데이터 격리 (site_id 필터링)
- ✅ bcrypt 비밀번호 해싱 (cost 12)
- ✅ 30분 수정 제한 시간 준수
- ✅ Soft delete 구현 (is_deleted 플래그)

### 성능 최적화
- DB 연결 풀 설정 (MaxOpenConns: 25, MaxIdleConns: 5)
- 사이트 정보 메모리 캐싱 (TTL 1분)
- 인덱스 활용 (idx_comments_post_id, idx_comments_parent_id)
- 페이지네이션으로 대량 조회 방지 (기본 50개, 최대 100개)

### 에러 처리
- 모든 에러는 일관된 JSON 포맷 반환
- HTTP 상태 코드 정확히 사용
- 로그에는 상세 정보, 사용자에게는 일반 메시지

## 추가 구현 필요 사항

### 구현 단계에 포함되지 않았지만 필요한 메서드들

#### 1. Post 관련 메서드 (`internal/database/posts.go`)
- `GetOrCreatePost()`: 포스트 조회, 없으면 생성
- `GetPostBySlug()`: site_id + slug로 포스트 조회
- `GetPostByID()`: ID로 포스트 조회
- `IncrementCommentCount()`: 댓글 수 증가
- `DecrementCommentCount()`: 댓글 수 감소

#### 2. Site 관련 메서드 (이미 구현되어 있어야 함)
- `GetSiteByAPIKey()`: API 키로 사이트 조회 (1분 TTL 캐시 포함)

#### 3. Comment 모델 확장 (`internal/models/comment.go`)
- `Replies []Comment` 필드 추가 (`json:"replies,omitempty"`)
- 조회 시에만 채워짐

### 의존 라이브러리 추가

#### go.mod에 추가 필요
```bash
go get golang.org/x/crypto/bcrypt
go get github.com/microcosm-cc/bluemonday
```

---

## 작업 완료 체크리스트

### 코드 작성
- [x] `internal/sanitizer/html.go` 생성 (HTML sanitization)
- [x] `internal/validators/comment.go` 생성 (입력 검증)
- [ ] `internal/handlers/middleware.go` 생성 (인증 미들웨어, rate limiting)
- [ ] `internal/handlers/comments.go` 생성 (CRUD 핸들러)
- [x] `internal/database/comments.go` 생성 (DB 메서드)
- [x] `internal/database/posts.go` 보완 (Post 관련 메서드)
- [x] `internal/models/comment.go` 보완 (Replies, IPAddressMasked 필드 추가)
- [ ] `cmd/api/main.go` 라우팅 추가

### 테스트
- [x] HTML sanitization 테스트 (스크립트 태그 제거)
- [x] 입력 검증 테스트 (Create/Update/Delete)
- [x] 댓글 생성 테스트 (정상/실패 케이스) - Database 레이어
- [x] 댓글 조회 테스트 (페이지네이션, 트리 구조) - Database 레이어
- [x] 댓글 수정 테스트 (정상/실패 케이스) - Database 레이어
- [x] 댓글 삭제 테스트 (soft delete) - Database 레이어
- [x] 대댓글 생성/조회 테스트 (1depth 제한 검증) - Database 레이어
- [ ] 인증 실패 테스트 (API 키 검증) - Handler 레이어
- [ ] Rate limiting 테스트 - Handler 레이어
- [ ] 사이트 격리 테스트 (멀티 테넌시) - Handler 레이어
- [x] DB 검증 (해싱, soft delete) - Integration 테스트로 완료

### 문서
- [x] API 명세 작성
- [x] 검증 방법 (curl 명령어)
- [x] 대댓글 처리 전략
- [x] 보안 고려사항
- [ ] 완료 후 문서 업데이트 (실제 소요 시간, 변경된 파일)

---

## 참고 자료
- Chi 라우터: https://github.com/go-chi/chi
- bcrypt: https://pkg.go.dev/golang.org/x/crypto/bcrypt
- PostgreSQL CTE (재귀 쿼리): https://www.postgresql.org/docs/16/queries-with.html
- Go database/sql: https://pkg.go.dev/database/sql
- Bluemonday (XSS 필터): https://github.com/microcosm-cc/bluemonday

---

## 작업 이력

### [2025-10-14] 작업 문서 초안 작성
- 러프한 구조 작성
- 기본적인 작업 범위 정의

### [2025-10-15] 작업 문서 상세 보완
- API 명세 추가 (요청/응답 스키마, 에러 코드 정의)
- 검증 방법 구체화 (11개 시나리오 + curl 명령어)
- 대댓글 처리 전략 문서화 (재귀 방식, 깊이 제한)
- 보안 고려사항 10가지 추가
- 구현 단계에 코드 스니펫 추가
- 추가 구현 필요 사항 명시
- 작업 완료 체크리스트 작성

### [2025-10-17] Comment CRUD Database 레이어 구현 완료 (TDD)
- `internal/database/comments.go` 구현 완료
  - `CreateComment()`: bcrypt 해싱, 1-depth 검증 (대댓글의 대댓글 차단)
  - `GetCommentByID()`: ID로 댓글 조회 (삭제된 댓글 포함)
  - `UpdateComment()`: content, IP, User-Agent 수정 (삭제된 댓글 제외)
  - `DeleteComment()`: Soft delete (is_deleted=TRUE, deleted_at=NOW())
  - `ListComments()`: 2-level 계층 구조 조회 + 페이지네이션
  - `getReplies()`: 대댓글 조회 헬퍼 (비공개 함수)
  - `scanComment()`: DB row → Comment 모델 변환 헬퍼
- 16개 통합 테스트 작성 및 통과
  - CreateComment: 4개 시나리오 (최상위, 대댓글, 2-depth 거부, 존재하지 않는 부모)
  - GetCommentByID: 3개 시나리오 (존재, 없음, 삭제됨)
  - UpdateComment: 3개 시나리오 (성공, 없음, 삭제됨)
  - DeleteComment: 3개 시나리오 (성공, 없음, 이미 삭제됨)
  - ListComments: 3개 시나리오 (계층 구조, 페이지네이션, 빈 포스트)
- 테스트 헬퍼 구조 개선
  - `internal/database/testhelpers_test.go` 생성 (공통 setupTestDB)
  - Table-specific cleanup 함수는 각 테스트 파일에 유지
- 환경변수 관리 개선
  - godotenv 통합 (.env 파일 지원)
  - 로컬/Docker/Production 환경 모두 지원
  - `cmd/api/main.go`에 ENV=production 조건부 로딩 추가
- 전체 프로젝트 테스트 검증: 66개 테스트 통과
- 향후 최적화 작업 문서화
  - `docs/tasks/pending/007-comment-performance-optimization.md` 생성
  - 대댓글 페이지네이션 + "더보기" 기능
  - N+1 쿼리 최적화 (IN clause 배치 쿼리)
- 프로젝트 warmup 자동화
  - `.claude-project-rules.md`에 작업 관리 방식 섹션 추가
  - `.claude/commands/warmup.md` slash command 생성

# 작업: 포스트 Reaction API 구현

## 개요
블로그 포스트에 간단한 감정 표현(좋아요/아쉬워요)을 추가하는 기능입니다. 인증 없이 사용 가능하며, 세션 기반으로 중복 방지 및 토글 기능을 제공합니다.

## 핵심 요구사항
- 포스트당 2가지 reaction 타입: `like` (좋아요), `dislike` (아쉬워요)
- 인증 불필요 (익명 사용자도 가능)
- 세션 기반 중복 방지 (같은 세션에서는 토글만 가능)
- 실시간 카운트 제공

## 데이터베이스 설계

### 테이블: `post_reactions`
```
id               SERIAL PRIMARY KEY
site_id          INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE
post_id          VARCHAR(255) NOT NULL          -- 외부 블로그의 포스트 ID
reaction_type    VARCHAR(20) NOT NULL           -- 'like' or 'dislike'
session_id       VARCHAR(255) NOT NULL          -- 세션 식별자 (해시값)
ip_address       INET                            -- 스팸 방지용 (선택)
user_agent       TEXT                            -- 스팸 방지용 (선택)
created_at       TIMESTAMP DEFAULT NOW()

INDEX idx_post_reactions_lookup ON post_reactions(site_id, post_id)
INDEX idx_post_reactions_session ON post_reactions(site_id, post_id, session_id)
UNIQUE INDEX idx_unique_session_reaction ON post_reactions(site_id, post_id, session_id)
```

## API 명세

### 공통 사항
**인증 헤더**
```
X-Orbithall-API-Key: {site의 api_key UUID}
X-Orbithall-Session-ID: {클라이언트가 생성한 세션 UUID}
```

**에러 코드**
- `MISSING_API_KEY`: API 키 누락
- `INVALID_API_KEY`: 유효하지 않은 API 키
- `SITE_INACTIVE`: 비활성화된 사이트
- `MISSING_SESSION_ID`: 세션 ID 누락
- `INVALID_SESSION_ID`: 유효하지 않은 세션 ID 형식
- `INVALID_REACTION_TYPE`: 지원하지 않는 reaction 타입
- `INTERNAL_SERVER_ERROR`: 서버 오류

---

### 1. Reaction 추가/토글

**POST** `/api/reactions`

동일 세션에서 동일 타입 재요청 시 → 삭제 (토글)
동일 세션에서 다른 타입 요청 시 → 기존 삭제 후 새로 추가

**Request Body**
```json
{
  "post_id": "my-first-post",
  "reaction_type": "like"
}
```

**Response 200 OK**
```json
{
  "action": "added",           // "added" | "removed" | "changed"
  "reaction_type": "like",
  "counts": {
    "like": 42,
    "dislike": 3
  }
}
```

**Response 400 Bad Request**
```json
{
  "error": "INVALID_REACTION_TYPE",
  "message": "reaction_type must be 'like' or 'dislike'"
}
```

---

### 2. Reaction 카운트 조회

**GET** `/api/reactions?post_id={post_id}`

**Response 200 OK**
```json
{
  "post_id": "my-first-post",
  "counts": {
    "like": 42,
    "dislike": 3
  },
  "user_reaction": "like"      // 현재 세션의 reaction (없으면 null)
}
```

---

### 3. 여러 포스트 카운트 조회 (배치)

**POST** `/api/reactions/batch`

블로그 목록 페이지에서 여러 포스트의 reaction 수를 한 번에 조회

**Request Body**
```json
{
  "post_ids": ["post-1", "post-2", "post-3"]
}
```

**Response 200 OK**
```json
{
  "reactions": [
    {
      "post_id": "post-1",
      "counts": { "like": 10, "dislike": 1 },
      "user_reaction": "like"
    },
    {
      "post_id": "post-2",
      "counts": { "like": 5, "dislike": 2 },
      "user_reaction": null
    },
    {
      "post_id": "post-3",
      "counts": { "like": 0, "dislike": 0 },
      "user_reaction": null
    }
  ]
}
```

## 세션 관리

### 클라이언트 측 구현
```
1. localStorage에 session_id가 없으면 UUID v4 생성
2. 모든 API 요청에 X-Orbithall-Session-ID 헤더로 전송
3. 세션 ID는 영구 저장 (사용자가 직접 삭제하지 않는 한)
```

### 서버 측 검증
```
1. 세션 ID 형식 검증 (UUID 형식)
2. 세션 ID 해시화 후 DB 저장 (SHA-256)
3. 동일 세션 ID로 과도한 요청 시 rate limiting (선택)
```

## 구현 순서

### 1. 데이터베이스
- [ ] `post_reactions` 테이블 마이그레이션 작성
- [ ] 인덱스 최적화 (site_id + post_id 복합 인덱스)

### 2. 모델
- [ ] `internal/models/reaction.go` - PostReaction 구조체
- [ ] reaction_type enum 상수 정의 (`ReactionTypeLike`, `ReactionTypeDislike`)

### 3. 데이터베이스 레이어
- [ ] `internal/database/reactions.go`
  - [ ] AddReaction(siteID, postID, sessionID, reactionType)
  - [ ] RemoveReaction(siteID, postID, sessionID)
  - [ ] ChangeReaction(siteID, postID, sessionID, newReactionType)
  - [ ] GetUserReaction(siteID, postID, sessionID) - 현재 세션의 reaction 조회
  - [ ] GetReactionCounts(siteID, postID) - like/dislike 카운트
  - [ ] GetReactionCountsBatch(siteID, postIDs) - 여러 포스트 카운트 조회

### 4. 핸들러
- [ ] `internal/handlers/reactions.go`
  - [ ] POST /api/reactions - 토글 로직 (추가/삭제/변경)
  - [ ] GET /api/reactions - 단일 포스트 카운트
  - [ ] POST /api/reactions/batch - 여러 포스트 카운트

### 5. 미들웨어
- [ ] 세션 ID 검증 미들웨어
- [ ] 세션 ID 해시화 처리

### 6. 라우팅
- [ ] `cmd/api/main.go`에 reaction 라우트 추가

## 보안 고려사항

### 1. 세션 ID 관리
- 클라이언트가 생성한 UUID 사용 (서버는 검증만)
- DB 저장 시 SHA-256 해시화
- 세션 ID 노출되어도 특정 사용자 식별 불가

### 2. 스팸 방지
- IP + User-Agent 저장 (선택)
- Rate limiting: 동일 세션에서 초당 5회 제한 (선택)
- 동일 IP에서 하루 1000회 제한 (선택)

### 3. API 키 격리
- 다른 사이트의 reaction 조회/수정 불가
- site_id 기반 완전 격리

### 4. 입력 검증
- post_id: 1-255자, 영숫자 및 하이픈/언더스코어만 허용
- reaction_type: 'like' 또는 'dislike'만 허용
- session_id: UUID v4 형식 검증

## 추가 구현 고려사항

### 성능 최적화
- 카운트 조회 시 COUNT(*) 대신 집계 테이블 사용 고려 (트래픽 높을 경우)
- 배치 조회 시 IN 쿼리 최적화
- Redis 캐싱 (선택, 추후 트래픽 증가 시)

### UI 통합
- 블로그에서 localStorage 기반 세션 관리 라이브러리 제공
- Reaction 컴포넌트 예제 코드 제공 (React)

### 분석
- 포스트별 reaction 통계 대시보드 (추후)
- 시간대별 reaction 추이 분석 (추후)

## 테스트 시나리오

### 1. 기본 토글 동작
```bash
# 1. like 추가
curl -X POST http://localhost:8080/api/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"post_id":"post-1","reaction_type":"like"}'
# → {"action":"added","reaction_type":"like","counts":{"like":1,"dislike":0}}

# 2. 같은 세션에서 like 재요청 (토글 → 삭제)
curl -X POST http://localhost:8080/api/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"post_id":"post-1","reaction_type":"like"}'
# → {"action":"removed","reaction_type":"like","counts":{"like":0,"dislike":0}}
```

### 2. Reaction 타입 변경
```bash
# 1. like 추가
curl -X POST http://localhost:8080/api/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"post_id":"post-1","reaction_type":"like"}'
# → {"action":"added","counts":{"like":1,"dislike":0}}

# 2. dislike로 변경
curl -X POST http://localhost:8080/api/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"post_id":"post-1","reaction_type":"dislike"}'
# → {"action":"changed","reaction_type":"dislike","counts":{"like":0,"dislike":1}}
```

### 3. 카운트 조회
```bash
curl "http://localhost:8080/api/reactions?post_id=post-1" \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1"
# → {"post_id":"post-1","counts":{"like":5,"dislike":2},"user_reaction":"like"}
```

### 4. 배치 조회
```bash
curl -X POST http://localhost:8080/api/reactions/batch \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"post_ids":["post-1","post-2","post-3"]}'
```

### 5. 세션 격리 검증
```bash
# 세션 A에서 like
curl -X POST ... -H "X-Orbithall-Session-ID: session-a" -d '{"reaction_type":"like"}'

# 세션 B에서 dislike
curl -X POST ... -H "X-Orbithall-Session-ID: session-b" -d '{"reaction_type":"dislike"}'

# 결과: 두 reaction 모두 유지됨 (각각 독립적)
```

### 6. API 키 격리 검증
```bash
# 사이트 A의 reaction
curl -X POST ... -H "X-Orbithall-API-Key: site-a-key" -d '{"post_id":"post-1",...}'

# 사이트 B에서 사이트 A의 포스트 조회 시도
curl "http://localhost:8080/api/reactions?post_id=post-1" \
  -H "X-Orbithall-API-Key: site-b-key"
# → {"counts":{"like":0,"dislike":0}} (사이트 B 관점에서는 0개)
```

## 완료 체크리스트
- [ ] 데이터베이스 마이그레이션 작성 및 테스트
- [ ] 모델 및 상수 정의
- [ ] 데이터베이스 레이어 구현 (6개 메서드)
- [ ] 세션 ID 검증 미들웨어
- [ ] 핸들러 구현 (3개 엔드포인트)
- [ ] 라우팅 설정
- [ ] 토글 로직 단위 테스트
- [ ] API 통합 테스트 (6개 시나리오)
- [ ] API 키/세션 격리 검증
- [ ] 성능 테스트 (배치 조회)
- [ ] 클라이언트 SDK/예제 작성 (선택)

## 참고사항
- 이 기능은 댓글 시스템과 독립적으로 동작
- 추후 댓글에도 reaction 추가 고려 가능 (테이블 구조 동일하게 설계)
- 세션 만료는 구현하지 않음 (클라이언트가 영구 보관)

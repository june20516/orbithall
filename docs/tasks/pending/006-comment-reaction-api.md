# 작업: 댓글 Reaction API 구현

## 개요
댓글에 다양한 감정 표현을 추가하는 기능입니다. 인증 없이 사용 가능하며, 세션 기반으로 중복 방지 및 토글 기능을 제공합니다.

## 핵심 요구사항
- 댓글당 6가지 reaction 타입 지원
- 인증 불필요 (익명 사용자도 가능)
- 세션 기반 중복 방지 (같은 세션에서는 토글만 가능)
- 실시간 카운트 제공

## Reaction 타입

| 타입 | 한글명 | 설명 |
|------|--------|------|
| `like` | 좋아요 | 긍정적 반응 |
| `thanks` | 고마워요 | 감사 표현 |
| `funny` | 재밌어요 | 유머러스한 댓글 |
| `confused` | 모르겠어요 | 이해하기 어려움 |
| `angry` | 화나요 | 부정적 반응 |
| `sad` | 슬퍼요 | 안타까운 내용 |

## 데이터베이스 설계

### 테이블: `comment_reactions`
```
id               SERIAL PRIMARY KEY
site_id          INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE
comment_id       INTEGER NOT NULL REFERENCES comments(id) ON DELETE CASCADE
reaction_type    VARCHAR(20) NOT NULL           -- 'like', 'thanks', 'funny', 'confused', 'angry', 'sad'
session_id       VARCHAR(255) NOT NULL          -- 세션 식별자 (해시값)
ip_address       INET                            -- 스팸 방지용 (선택)
user_agent       TEXT                            -- 스팸 방지용 (선택)
created_at       TIMESTAMP DEFAULT NOW()

INDEX idx_comment_reactions_lookup ON comment_reactions(site_id, comment_id)
INDEX idx_comment_reactions_session ON comment_reactions(site_id, comment_id, session_id)
UNIQUE INDEX idx_unique_session_reaction ON comment_reactions(site_id, comment_id, session_id)
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
- `COMMENT_NOT_FOUND`: 댓글을 찾을 수 없음
- `INTERNAL_SERVER_ERROR`: 서버 오류

---

### 1. Reaction 추가/토글

**POST** `/api/comments/{comment_id}/reactions`

동일 세션에서 동일 타입 재요청 시 → 삭제 (토글)
동일 세션에서 다른 타입 요청 시 → 기존 삭제 후 새로 추가

**Request Body**
```json
{
  "reaction_type": "like"
}
```

**Response 200 OK**
```json
{
  "action": "added",           // "added" | "removed" | "changed"
  "reaction_type": "like",
  "counts": {
    "like": 15,
    "thanks": 3,
    "funny": 8,
    "confused": 1,
    "angry": 0,
    "sad": 2
  }
}
```

**Response 400 Bad Request**
```json
{
  "error": "INVALID_REACTION_TYPE",
  "message": "reaction_type must be one of: like, thanks, funny, confused, angry, sad"
}
```

**Response 404 Not Found**
```json
{
  "error": "COMMENT_NOT_FOUND",
  "message": "Comment not found or deleted"
}
```

---

### 2. Reaction 카운트 조회

**GET** `/api/comments/{comment_id}/reactions`

**Response 200 OK**
```json
{
  "comment_id": 123,
  "counts": {
    "like": 15,
    "thanks": 3,
    "funny": 8,
    "confused": 1,
    "angry": 0,
    "sad": 2
  },
  "user_reaction": "like"      // 현재 세션의 reaction (없으면 null)
}
```

---

### 3. 여러 댓글 카운트 조회 (배치)

**POST** `/api/comments/reactions/batch`

댓글 목록에서 여러 댓글의 reaction 수를 한 번에 조회

**Request Body**
```json
{
  "comment_ids": [123, 456, 789]
}
```

**Response 200 OK**
```json
{
  "reactions": [
    {
      "comment_id": 123,
      "counts": {
        "like": 15,
        "thanks": 3,
        "funny": 8,
        "confused": 1,
        "angry": 0,
        "sad": 2
      },
      "user_reaction": "like"
    },
    {
      "comment_id": 456,
      "counts": {
        "like": 5,
        "thanks": 1,
        "funny": 0,
        "confused": 0,
        "angry": 0,
        "sad": 0
      },
      "user_reaction": null
    },
    {
      "comment_id": 789,
      "counts": {
        "like": 0,
        "thanks": 0,
        "funny": 0,
        "confused": 0,
        "angry": 0,
        "sad": 0
      },
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
4. 포스트 reaction과 동일한 세션 ID 사용
```

### 서버 측 검증
```
1. 세션 ID 형식 검증 (UUID 형식)
2. 세션 ID 해시화 후 DB 저장 (SHA-256)
3. 동일 세션 ID로 과도한 요청 시 rate limiting (선택)
```

## 구현 순서

### 1. 데이터베이스
- [ ] `comment_reactions` 테이블 마이그레이션 작성
- [ ] 인덱스 최적화 (site_id + comment_id 복합 인덱스)
- [ ] comment_id 외래키 제약조건 (ON DELETE CASCADE)

### 2. 모델
- [ ] `internal/models/comment_reaction.go` - CommentReaction 구조체
- [ ] reaction_type enum 상수 정의
  - [ ] `ReactionTypeLike`
  - [ ] `ReactionTypeThanks`
  - [ ] `ReactionTypeFunny`
  - [ ] `ReactionTypeConfused`
  - [ ] `ReactionTypeAngry`
  - [ ] `ReactionTypeSad`

### 3. 데이터베이스 레이어
- [ ] `internal/database/comment_reactions.go`
  - [ ] AddCommentReaction(siteID, commentID, sessionID, reactionType)
  - [ ] RemoveCommentReaction(siteID, commentID, sessionID)
  - [ ] ChangeCommentReaction(siteID, commentID, sessionID, newReactionType)
  - [ ] GetUserCommentReaction(siteID, commentID, sessionID) - 현재 세션의 reaction 조회
  - [ ] GetCommentReactionCounts(siteID, commentID) - 6가지 타입별 카운트
  - [ ] GetCommentReactionCountsBatch(siteID, commentIDs) - 여러 댓글 카운트 조회

### 4. 핸들러
- [ ] `internal/handlers/comment_reactions.go`
  - [ ] POST /api/comments/{comment_id}/reactions - 토글 로직 (추가/삭제/변경)
  - [ ] GET /api/comments/{comment_id}/reactions - 단일 댓글 카운트
  - [ ] POST /api/comments/reactions/batch - 여러 댓글 카운트

### 5. 미들웨어
- [ ] 세션 ID 검증 미들웨어 (포스트 reaction과 공유)
- [ ] 세션 ID 해시화 처리

### 6. 라우팅
- [ ] `cmd/api/main.go`에 comment reaction 라우트 추가

## 보안 고려사항

### 1. 세션 ID 관리
- 클라이언트가 생성한 UUID 사용 (서버는 검증만)
- DB 저장 시 SHA-256 해시화
- 포스트 reaction과 동일한 세션 ID 체계 사용

### 2. 스팸 방지
- IP + User-Agent 저장 (선택)
- Rate limiting: 동일 세션에서 초당 10회 제한 (선택)
- 동일 IP에서 하루 2000회 제한 (선택)

### 3. API 키 격리
- 다른 사이트의 reaction 조회/수정 불가
- site_id 기반 완전 격리

### 4. 댓글 존재 확인
- reaction 추가 시 comment_id 유효성 검증
- 삭제된 댓글(is_deleted=true)에는 reaction 불가
- 외래키 제약조건으로 데이터 일관성 보장

### 5. 입력 검증
- comment_id: 양의 정수
- reaction_type: 6가지 타입 중 하나만 허용
- session_id: UUID v4 형식 검증

## 추가 구현 고려사항

### 성능 최적화
- 카운트 조회 시 GROUP BY 최적화
- 배치 조회 시 IN 쿼리 최적화
- Redis 캐싱 (선택, 추후 트래픽 증가 시)

### UI 통합
- 댓글 목록에 reaction 버튼 표시
- 토글 애니메이션 처리
- 실시간 카운트 업데이트

### 분석
- 댓글별 reaction 통계 (어떤 댓글이 가장 많은 공감을 받았는지)
- Reaction 타입별 분포 분석
- 포스트별 댓글 reaction 통계 (추후)

## 테스트 시나리오

### 1. 기본 토글 동작
```bash
# 1. like 추가
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"reaction_type":"like"}'
# → {"action":"added","reaction_type":"like","counts":{...}}

# 2. 같은 세션에서 like 재요청 (토글 → 삭제)
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"reaction_type":"like"}'
# → {"action":"removed","reaction_type":"like","counts":{...}}
```

### 2. Reaction 타입 변경
```bash
# 1. like 추가
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"reaction_type":"like"}'

# 2. funny로 변경
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"reaction_type":"funny"}'
# → {"action":"changed","reaction_type":"funny","counts":{...}}
```

### 3. 다양한 reaction 타입 테스트
```bash
# thanks
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-2" \
  -d '{"reaction_type":"thanks"}'

# confused
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-3" \
  -d '{"reaction_type":"confused"}'

# angry
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-4" \
  -d '{"reaction_type":"angry"}'

# sad
curl -X POST http://localhost:8080/api/comments/123/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-5" \
  -d '{"reaction_type":"sad"}'
```

### 4. 카운트 조회
```bash
curl "http://localhost:8080/api/comments/123/reactions" \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1"
# → {"comment_id":123,"counts":{"like":5,"thanks":2,...},"user_reaction":"like"}
```

### 5. 배치 조회
```bash
curl -X POST http://localhost:8080/api/comments/reactions/batch \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"comment_ids":[123,456,789]}'
```

### 6. 삭제된 댓글 처리
```bash
# 삭제된 댓글에 reaction 추가 시도
curl -X POST http://localhost:8080/api/comments/999/reactions \
  -H "X-Orbithall-API-Key: test-key" \
  -H "X-Orbithall-Session-ID: session-1" \
  -d '{"reaction_type":"like"}'
# → 404 COMMENT_NOT_FOUND
```

### 7. 세션 격리 검증
```bash
# 세션 A에서 like
curl -X POST ... -H "X-Orbithall-Session-ID: session-a" -d '{"reaction_type":"like"}'

# 세션 B에서 funny
curl -X POST ... -H "X-Orbithall-Session-ID: session-b" -d '{"reaction_type":"funny"}'

# 결과: 두 reaction 모두 유지됨 (각각 독립적)
```

### 8. API 키 격리 검증
```bash
# 사이트 A의 reaction
curl -X POST ... -H "X-Orbithall-API-Key: site-a-key" ...

# 사이트 B에서 사이트 A의 댓글 조회 시도
curl "http://localhost:8080/api/comments/123/reactions" \
  -H "X-Orbithall-API-Key: site-b-key"
# → {"counts":{"like":0,"thanks":0,...}} (사이트 B 관점에서는 0개)
```

## 완료 체크리스트
- [ ] 데이터베이스 마이그레이션 작성 및 테스트
- [ ] 모델 및 상수 정의 (6가지 타입)
- [ ] 데이터베이스 레이어 구현 (6개 메서드)
- [ ] 세션 ID 검증 미들웨어 (공유)
- [ ] 핸들러 구현 (3개 엔드포인트)
- [ ] 라우팅 설정
- [ ] 댓글 존재 및 삭제 여부 검증
- [ ] 토글 로직 단위 테스트
- [ ] API 통합 테스트 (8개 시나리오)
- [ ] API 키/세션 격리 검증
- [ ] 성능 테스트 (배치 조회)
- [ ] 클라이언트 SDK/예제 작성 (선택)

## 참고사항
- 포스트 reaction (005)과 구조 유사하지만 타입이 더 다양함
- 세션 ID 체계는 포스트 reaction과 동일하게 사용
- 댓글이 삭제되면 모든 reaction도 CASCADE로 삭제됨
- 추후 대댓글에도 동일한 reaction 시스템 적용 가능

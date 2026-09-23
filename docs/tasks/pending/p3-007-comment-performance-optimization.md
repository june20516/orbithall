# [WIP] 댓글 시스템 성능 최적화

## 작성일
2025-10-17

## 우선순위
- [ ] 긴급
- [ ] 높음
- [x] 보통
- [ ] 낮음

## 작업 개요
댓글 시스템의 대댓글 페이지네이션과 N+1 쿼리 문제 해결

## 작업 목적
대댓글이 많은 댓글과 많은 수의 최상위 댓글을 효율적으로 처리하여 성능 개선

## 작업 범위

### 1. 대댓글 페이지네이션 + "더보기" 기능

#### 현재 문제
- 대댓글이 수십~수백 개인 경우 모든 대댓글을 한번에 로드
- 초기 로딩 시간 증가 및 불필요한 데이터 전송

#### 해결 방안
- 대댓글 페이지네이션 API 추가
- 프론트엔드에서 "더보기" 버튼으로 추가 로드
- 제안 엔드포인트: `GET /api/comments/:id/replies?page=2&limit=20`

#### 구현 사항
- `ListReplies(db, parentID, limit, offset)` 함수 추가
- 대댓글 조회 핸들러 구현
- 초기 응답에는 대댓글 3-5개만 포함, 나머지는 필요시 로드

### 2. N+1 쿼리 문제 해결 (IN Clause 최적화)

#### 현재 문제
```go
// 현재: 최상위 댓글 50개 = 51번의 쿼리 (1 + 50)
for _, comment := range comments {
    replies := getReplies(db, comment.ID)  // 각 댓글마다 개별 쿼리
}
```

#### 성능 영향
- 50개 최상위 댓글 → 51번의 데이터베이스 쿼리
- 네트워크 왕복 시간(RTT) 누적
- 데이터베이스 부하 증가

#### 해결 방안
```go
// 최적화: 2번의 쿼리로 감소 (1 + 1)
// 1. 모든 댓글 ID 수집
commentIDs := []int64{1, 2, 3, ..., 50}

// 2. 배치 쿼리로 모든 대댓글 조회
query := `
    SELECT id, post_id, parent_id, ...
    FROM comments
    WHERE parent_id IN (1, 2, 3, ..., 50)
    ORDER BY parent_id, created_at ASC
`

// 3. 메모리에서 parent_id로 그룹핑
repliesMap := map[int64][]*Comment{}
for _, reply := range allReplies {
    repliesMap[reply.ParentID] = append(repliesMap[reply.ParentID], reply)
}
```

#### 구현 사항
- `batchGetReplies(db *sql.DB, parentIDs []int64)` 함수 추가
- ListComments 함수에서 IN clause 사용하도록 리팩토링
- 성능 개선: 51 queries → 2 queries

## 의존성
- 선행 작업: 댓글 CRUD API 구현 완료
- 관련 파일: `internal/database/comments.go`, `internal/handlers/comments.go`

## 예상 소요 시간
- 대댓글 페이지네이션: 2-3시간
- N+1 쿼리 최적화: 1-2시간
- 총 예상: 3-5시간

## 주의사항
- IN clause의 파라미터 개수 제한 확인 (PostgreSQL은 ~32,767개)
- 대댓글 페이지네이션 시 total count 제공 필요
- 기존 API와의 하위 호환성 고려

---

## 작업 이력

### [2025-10-17] 작업 문서 작성
- Comment CRUD database 구현 완료 후 식별된 성능 개선 사항
- Reply pagination과 N+1 문제를 pending task로 기록

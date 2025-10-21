# ADR-004: Comment Sorting Strategy

## Status
Accepted

## Context
댓글 목록 조회 시 정렬 순서를 보장하는 방법에는 여러 선택지가 있습니다:

1. **`ORDER BY created_at ASC`만 사용**
   - 생성 시간 순으로 정렬
   - 문제: 같은 트랜잭션 내에서 생성된 댓글은 순서가 보장되지 않음

2. **`ORDER BY created_at ASC, id ASC`** (복합 정렬)
   - 생성 시간 우선, 동일 시간이면 ID 순
   - ID는 SERIAL 타입으로 자동 증가하여 생성 순서 보장

3. **별도 `sort_order` 컬럼 추가**
   - 명시적 정렬 순서 저장
   - 오버헤드 증가, 복잡도 상승

문제 상황:
- 통합 테스트에서 트랜잭션 내 여러 댓글 생성 시 `ORDER BY created_at ASC`만으로는 순서가 보장되지 않음
- PostgreSQL의 `NOW()` 함수는 트랜잭션 시작 시각을 반환하므로, 같은 트랜잭션 내 모든 INSERT가 동일한 timestamp를 가짐
- 실제 프로덕션에서도 짧은 시간 내 여러 댓글 생성 시 밀리초 단위까지 동일할 수 있음

## Decision
**`ORDER BY created_at ASC, id ASC`** 복합 정렬을 채택합니다.

적용 대상:
1. **최상위 댓글 조회** (`ListComments`):
   ```sql
   ORDER BY created_at ASC, id ASC
   ```

2. **대댓글 조회** (`getReplies`):
   ```sql
   ORDER BY created_at ASC, id ASC
   ```

## Consequences

### Positive
- **순서 보장**: `created_at`이 동일해도 `id`로 일관된 순서 보장
- **트랜잭션 안전**: 테스트 환경(트랜잭션)과 프로덕션 환경 모두 동일한 동작
- **인덱스 활용**: 기존 인덱스(`idx_comments_post_id`, `idx_comments_parent_id`)에 `created_at`, `id` 포함되어 있어 성능 영향 없음
- **단순성**: 별도 컬럼 추가 없이 기존 필드만 활용
- **예측 가능성**: 생성 순서가 곧 조회 순서

### Negative
- **미세한 정렬 차이**: `created_at`이 동일한 경우 사용자가 보기엔 무작위처럼 보일 수 있음 (실제로는 생성 순서)
  - **대응책**: 댓글 시스템 특성상 밀리초 단위 동시 생성은 드물며, ID 순 정렬이 자연스러움

### Neutral
- 사용자 정의 정렬 순서 변경 불가 (항상 생성 순)

## Alternatives Considered

### Alternative 1: `ORDER BY created_at ASC`만 사용
- **거부 이유**: 트랜잭션 내 순서 보장 안 됨, 테스트 실패

### Alternative 2: `ORDER BY id ASC`만 사용
- **거부 이유**: `created_at`을 무시하면 시간 정보 손실, 비즈니스 로직과 맞지 않음

### Alternative 3: `sort_order` 컬럼 추가
- **거부 이유**:
  - 스키마 복잡도 증가
  - INSERT/UPDATE 시 추가 관리 필요
  - 현재 요구사항에 과한 해결책

## Implementation Details
적용된 코드 위치:
- `internal/database/comments.go:181` - ListComments 최상위 댓글 정렬
- `internal/database/comments.go:237` - getReplies 대댓글 정렬

## References
- PostgreSQL NOW() vs CLOCK_TIMESTAMP() 문서: https://www.postgresql.org/docs/current/functions-datetime.html
- `internal/database/comments.go:177-183` - 최상위 댓글 쿼리
- `internal/database/comments.go:233-238` - 대댓글 쿼리
- `internal/handlers/comments_test.go:TestListComments_Success_TreeStructure` - 순서 검증 테스트

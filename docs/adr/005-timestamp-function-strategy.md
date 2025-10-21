# ADR-005: Timestamp Function Strategy (NOW() vs CLOCK_TIMESTAMP())

## Status
Accepted

## Context
PostgreSQL에서 타임스탬프를 기록할 때 사용할 수 있는 함수는 여러 가지가 있습니다:

1. **NOW() / CURRENT_TIMESTAMP**
   - 현재 트랜잭션의 시작 시각 반환
   - 같은 트랜잭션 내 여러 호출 시 동일한 값 반환
   - 트랜잭션 일관성 보장

2. **CLOCK_TIMESTAMP()**
   - 실제 현재 시각 반환 (wall-clock time)
   - 호출 시점마다 다른 값 반환
   - 트랜잭션과 무관하게 실제 시각 반영

3. **STATEMENT_TIMESTAMP()**
   - 현재 SQL 문 시작 시각 반환
   - 같은 문 내 여러 호출 시 동일한 값

문제 상황:
- `created_at`에 `NOW()` 사용 시: 같은 트랜잭션 내 모든 레코드가 동일한 timestamp 가짐 (정상 동작)
- `updated_at`에 `NOW()` 사용 시: UPDATE 시점의 실제 시각이 아닌 트랜잭션 시작 시각 기록됨
- 통합 테스트(트랜잭션 기반)에서 `created_at`과 `updated_at`이 동일하게 나타나는 문제 발생

## Decision
필드별로 다른 타임스탬프 함수를 사용합니다:

| 필드 | 함수 | 이유 |
|------|------|------|
| `created_at` | `NOW()` | 트랜잭션 시작 시각으로 일관성 있는 생성 시각 기록 |
| `updated_at` (UPDATE 시) | `CLOCK_TIMESTAMP()` | 실제 수정 시점의 시각 정확히 반영 |
| `deleted_at` (soft delete 시) | `CLOCK_TIMESTAMP()` | 실제 삭제 시점의 시각 정확히 반영 |

### 구현 원칙
- **생성 시각 (`created_at`)**: 트랜잭션 일관성이 중요하므로 `NOW()` 사용
  - 스키마 default: `DEFAULT NOW()`
  - 같은 트랜잭션 내 여러 레코드는 동일 생성 시각 (예상된 동작)

- **수정/삭제 시각 (`updated_at`, `deleted_at`)**: 실제 시점이 중요하므로 `CLOCK_TIMESTAMP()` 사용
  - `UpdateComment`: `SET updated_at = CLOCK_TIMESTAMP()`
  - `DeleteComment`: `SET deleted_at = CLOCK_TIMESTAMP()`

## Consequences

### Positive
- **정확성**: UPDATE/DELETE 시점의 실제 시각 정확히 기록
- **테스트 안정성**: 트랜잭션 기반 테스트에서도 `created_at != updated_at` 보장
- **디버깅 용이**: 수정/삭제 시각이 실제 발생 시각을 반영하여 로그 분석 쉬움
- **비즈니스 로직 정확성**: "30분 수정 제한" 같은 시간 기반 로직이 정확히 동작

### Negative
- **일관성 차이**: `created_at`은 트랜잭션 시각, `updated_at`은 실제 시각으로 다른 기준
  - **대응책**: 문서화로 명확히 하고, 각 필드의 목적에 맞는 선택이므로 문제없음
- **트랜잭션과 무관**: `CLOCK_TIMESTAMP()`는 트랜잭션 롤백 시에도 시간은 되돌릴 수 없음
  - **대응책**: timestamp 자체는 롤백 안 되지만, 레코드 자체가 롤백되므로 문제없음

### Neutral
- 프로덕션(트랜잭션 없음)에서는 `NOW()`와 `CLOCK_TIMESTAMP()`가 사실상 동일하게 동작
- 차이는 주로 테스트 환경에서 발생

## Technical Details

### NOW() vs CLOCK_TIMESTAMP() 동작 차이

**트랜잭션 환경에서:**
```sql
BEGIN;
SELECT NOW();           -- 2025-10-21 10:00:00
SELECT CLOCK_TIMESTAMP(); -- 2025-10-21 10:00:00.001
COMMIT;
```

**일반 환경에서:**
```sql
SELECT NOW();           -- 2025-10-21 10:00:00
SELECT CLOCK_TIMESTAMP(); -- 2025-10-21 10:00:00 (거의 동일)
```

### 구현 위치
- `migrations/002_create_comments_table.sql`: `created_at TIMESTAMPTZ DEFAULT NOW()`
- `internal/database/comments.go:112`: `SET updated_at = CLOCK_TIMESTAMP()`
- `internal/database/comments.go:140`: `SET deleted_at = CLOCK_TIMESTAMP()`

## Alternatives Considered

### Alternative 1: 모든 필드에 `NOW()` 사용
- **거부 이유**:
  - UPDATE 시 `updated_at`이 실제 수정 시각을 반영하지 못함
  - 트랜잭션 기반 테스트에서 `created_at == updated_at`이 되어 검증 불가
  - 비즈니스 로직(30분 제한)이 부정확하게 동작할 수 있음

### Alternative 2: 모든 필드에 `CLOCK_TIMESTAMP()` 사용
- **거부 이유**:
  - 같은 트랜잭션 내 생성된 레코드들의 생성 시각이 미세하게 달라짐
  - 트랜잭션 일관성 개념과 맞지 않음
  - 불필요한 정밀도로 인한 혼란 가능성

### Alternative 3: 애플리케이션 레벨에서 시각 생성
- **거부 이유**:
  - DB와 애플리케이션 서버 시간 불일치 가능성
  - DB의 timestamp 기능을 활용하지 못함
  - 코드 복잡도 증가

## Future Considerations
- 현재 방식으로 문제가 없으므로 변경 계획 없음
- 만약 감사 로그(audit log) 기능 추가 시, 모든 변경 사항을 `CLOCK_TIMESTAMP()`로 기록하는 별도 테이블 고려 가능

## References
- PostgreSQL 공식 문서: https://www.postgresql.org/docs/current/functions-datetime.html
- `internal/database/comments.go:106-131` - UpdateComment 구현
- `internal/database/comments.go:133-159` - DeleteComment 구현
- `internal/handlers/comments_test.go:TestUpdateComment_Success` - updated_at 검증 테스트

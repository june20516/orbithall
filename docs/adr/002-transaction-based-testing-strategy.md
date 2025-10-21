# ADR-002: Transaction-based Testing Strategy

## Status
Accepted

## Context
통합 테스트에서 데이터베이스 상태를 관리하는 방법에는 여러 선택지가 있습니다:

1. **Cleanup 함수 방식**: 각 테스트 후 명시적으로 INSERT한 데이터를 DELETE
2. **Transaction Rollback 방식**: 각 테스트를 트랜잭션 내에서 실행하고 종료 시 롤백
3. **Database Truncate 방식**: 각 테스트 후 모든 테이블을 TRUNCATE
4. **Test Database Recreation**: 각 테스트마다 데이터베이스를 새로 생성

기존 코드에서는 cleanup 함수 방식을 사용하고 있었으나, 다음과 같은 문제가 있었습니다:
- Cleanup 함수 작성이 번거롭고 오류가 발생하기 쉬움
- 외래 키 제약조건으로 인한 삭제 순서 관리 필요
- 테스트 실패 시 cleanup이 실행되지 않아 데이터 누수 발생 가능
- 테스트 코드가 길어지고 가독성 저하

## Decision
**Transaction Rollback 방식**을 채택합니다.

모든 통합 테스트는 다음 패턴을 따릅니다:
```go
func TestSomething(t *testing.T) {
    db := testhelpers.SetupTestDB(t)
    defer db.Close()

    // 트랜잭션 시작 및 cleanup 함수 설정
    ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
    defer cleanup()

    // 테스트 실행 (tx 사용)
    // ...
}
```

`SetupTxTest` 헬퍼 함수는 다음을 제공합니다:
- 새로운 트랜잭션 시작
- Context 생성
- Cleanup 함수 (자동 롤백)

## Consequences

### Positive
- **코드 간결성**: Cleanup 로직 불필요, defer cleanup()만으로 자동 정리
- **안전성**: 테스트 실패해도 항상 롤백되어 데이터 누수 없음
- **속도**: DELETE 쿼리 실행 불필요, 롤백이 더 빠름
- **격리성**: 각 테스트가 완전히 독립적인 상태에서 실행
- **유지보수성**: 외래 키 삭제 순서 고려 불필요

### Negative
- **PostgreSQL NOW() 동작**: 트랜잭션 내에서 NOW()는 트랜잭션 시작 시각을 반환하므로, 같은 트랜잭션 내 여러 INSERT에서 동일한 timestamp 발생
  - **대응책**: `ORDER BY created_at ASC, id ASC`처럼 id를 secondary sort key로 사용하여 순서 보장
- **CLOCK_TIMESTAMP() 필요성**: UPDATE/DELETE 시 실제 현재 시각이 필요한 경우 CLOCK_TIMESTAMP() 사용 필요
  - **대응책**: `UpdateComment`, `DeleteComment` 함수에서 CLOCK_TIMESTAMP() 사용

### Neutral
- 트랜잭션 롤백이 실제 프로덕션 동작과 다를 수 있으나, 단위 테스트의 목적(비즈니스 로직 검증)에는 무관

## References
- `internal/testhelpers/testhelpers.go:69-82` - SetupTxTest 구현
- `internal/handlers/comments_test.go` - 모든 테스트에서 transaction 패턴 적용
- `internal/database/comments.go:112,140` - CLOCK_TIMESTAMP() 사용 예시

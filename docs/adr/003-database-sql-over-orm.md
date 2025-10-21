# ADR-003: database/sql Over ORM

## Status
Accepted

## Context
Go에서 PostgreSQL을 사용할 때 데이터베이스 접근 방식에는 여러 선택지가 있습니다:

1. **ORM (GORM, ent 등)**
   - 장점: 타입 안전성, 자동 마이그레이션, 관계 매핑, 보일러플레이트 감소
   - 단점: 학습 곡선, 성능 오버헤드, 복잡한 쿼리 작성 어려움, 마법같은 동작

2. **Query Builder (sqlc, sqlx 등)**
   - 장점: SQL 작성, 타입 안전성, 생성 코드
   - 단점: 추가 빌드 단계, 도구 의존성

3. **Raw SQL (database/sql)**
   - 장점: 완전한 제어, 학습 곡선 없음, 표준 라이브러리, 성능
   - 단점: 보일러플레이트, 타입 안전성 부족, 수동 매핑

OrbitHall은 다음과 같은 특성을 가집니다:
- **단순한 데이터 모델**: Sites, Posts, Comments 3개 테이블
- **복잡한 관계 없음**: 1:N 관계만 존재, N:M 없음
- **성능 중요**: 댓글 시스템은 트래픽이 많을 수 있음
- **팀 규모**: 1인 또는 소규모 팀
- **명시성 선호**: 코드 동작이 명확하게 보여야 함

## Decision
**Raw SQL (database/sql)**을 사용합니다.

데이터베이스 레이어는 다음 패턴을 따릅니다:
```go
// internal/database/comments.go
func CreateComment(ctx context.Context, db DBTX, ...) (*models.Comment, error) {
    query := `
        INSERT INTO comments (post_id, parent_id, author_name, ...)
        VALUES ($1, $2, $3, ...)
        RETURNING id, post_id, parent_id, ...
    `

    row := db.QueryRowContext(ctx, query, postID, parentID, authorName, ...)
    comment, err := scanComment(row)
    if err != nil {
        return nil, fmt.Errorf("failed to create comment: %w", err)
    }

    return comment, nil
}
```

핵심 원칙:
- 모든 쿼리는 명시적으로 작성
- DBTX 인터페이스로 `*sql.DB`와 `*sql.Tx` 모두 지원
- Context 사용으로 취소/타임아웃 지원
- 에러는 명확한 메시지와 함께 wrapping

## Consequences

### Positive
- **명시성**: 모든 SQL이 코드에 그대로 보임, 디버깅 쉬움
- **성능**: 불필요한 추상화 레이어 없음, 최적화된 쿼리 작성 가능
- **단순성**: 표준 라이브러리만 사용, 추가 학습 불필요
- **제어**: SQL의 모든 기능 활용 가능 (CTE, RETURNING, CLOCK_TIMESTAMP 등)
- **트랜잭션 지원**: DBTX 인터페이스로 트랜잭션/일반 DB 통일된 방식 처리
- **도구 독립성**: 빌드 도구나 코드 생성 단계 불필요

### Negative
- **보일러플레이트**: Scan 코드 수동 작성 필요
  - **대응책**: `scanComment` 같은 헬퍼 함수로 재사용
- **타입 안전성 부족**: 컴파일 타임 쿼리 검증 없음
  - **대응책**: 통합 테스트로 모든 쿼리 검증
- **리팩토링 어려움**: 스키마 변경 시 수동으로 모든 쿼리 수정 필요
  - **대응책**: 프로젝트 규모가 작아 현재로선 문제없음, 향후 필요시 sqlc 도입 고려

### Neutral
- ORM을 사용했다면 더 빠른 개발이 가능했을 수 있으나, 장기적 유지보수성과 성능을 우선시

## Future Considerations
프로젝트가 다음 조건을 만족하게 되면 **sqlc** 도입을 재고할 수 있습니다:
- 테이블 수가 10개 이상으로 증가
- 복잡한 쿼리가 많아져 타입 안전성이 중요해짐
- 팀 규모가 커져 컴파일 타임 검증 필요

단, GORM 같은 Full ORM은 여전히 고려하지 않습니다. OrbitHall의 철학은 "명시성과 제어"이므로, Query Builder까지만 도입합니다.

## References
- `internal/database/db.go:8-13` - DBTX 인터페이스 정의
- `internal/database/comments.go` - Raw SQL 패턴 예시
- `internal/handlers/comments_test.go` - 통합 테스트로 쿼리 검증

# Architecture Decision Records (ADR)

## ADR이란?

ADR(Architecture Decision Records)은 소프트웨어 프로젝트에서 내린 **중요한 아키텍처 결정을 기록하는 문서**입니다.

### 왜 ADR을 작성하나요?

- **맥락 보존**: 왜 이런 결정을 내렸는지 배경과 이유를 기록
- **지식 공유**: 팀원이나 미래의 나 자신에게 결정 과정 전달
- **의사결정 추적**: 시간순으로 결정 기록, 나중에 변경 시 참고
- **대안 검토**: 고려했던 다른 옵션들과 선택하지 않은 이유 명시

## ADR 작성 시기

다음과 같은 경우 ADR을 작성합니다:

- 기술 스택 선택 (데이터베이스, 프레임워크, 라이브러리 등)
- 아키텍처 패턴 결정 (마이크로서비스, 모노리스, 이벤트 소싱 등)
- 배포 전략 수립 (CI/CD, 인프라, 컨테이너 오케스트레이션 등)
- 보안 정책 결정
- 데이터 모델링 주요 결정
- 성능/확장성 관련 중요한 트레이드오프

**작성하지 않는 경우:**
- 단순한 버그 수정
- 코드 스타일 변경
- 사소한 리팩토링

## ADR 문서 구조

```markdown
# ADR-XXX: 결정 제목

## Status
Proposed | Accepted | Deprecated | Superseded

## Context
- 어떤 문제/상황인가?
- 왜 결정이 필요한가?
- 어떤 제약사항이 있는가?

## Decision
- 무엇을 결정했는가?
- 어떻게 구현하는가?

## Consequences
### 장점
- 이 결정의 긍정적 영향

### 단점
- 이 결정의 부정적 영향 또는 트레이드오프

## Alternatives Considered
- 고려했던 다른 옵션들
- 선택하지 않은 이유

## Related Decisions
(선택사항) 관련된 다른 ADR

## References
(선택사항) 참고 문서, 링크
```

## Status 의미

- **Proposed**: 제안됨, 아직 검토 중
- **Accepted**: 승인됨, 현재 적용 중
- **Deprecated**: 더 이상 권장하지 않음 (하지만 아직 사용 중)
- **Superseded**: 다른 ADR로 대체됨 (ADR-XXX by ADR-YYY)

## 새 ADR 작성 방법

1. **번호 결정**: 마지막 ADR 번호 + 1
   ```bash
   ls docs/adr/ | grep -E '^[0-9]+' | sort -n | tail -1
   # 001-database-migration-strategy.md
   # 다음 번호: 002
   ```

2. **파일 생성**: `docs/adr/XXX-descriptive-title.md`
   ```bash
   touch docs/adr/002-deployment-platform.md
   ```

3. **템플릿 작성**: 위 구조 참고하여 작성

4. **이 README 업데이트**: 아래 목록에 새 ADR 추가

## 현재 ADR 목록

| 번호 | 제목 | 상태 | 날짜 |
|------|------|------|------|
| [001](001-database-migration-strategy.md) | 데이터베이스 마이그레이션 전략 | Accepted | 2025-10-02 |
| [002](002-transaction-based-testing-strategy.md) | 트랜잭션 기반 테스트 전략 | Accepted | 2025-10-22 |
| [003](003-database-sql-over-orm.md) | 데이터베이스 접근 방식 (SQL vs ORM) | Accepted | 2025-10-22 |
| [004](004-comment-sorting-strategy.md) | 댓글 정렬 전략 | Accepted | 2025-10-22 |
| [005](005-timestamp-function-strategy.md) | Timestamp 함수 전략 | Accepted | 2025-10-22 |
| [006](006-widget-versioning-deployment-strategy.md) | Widget 버전 관리 및 배포 전략 | Accepted | 2025-10-24 |

## 참고 자료

- [ADR GitHub Organization](https://adr.github.io/)
- [Documenting Architecture Decisions - Michael Nygard](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)

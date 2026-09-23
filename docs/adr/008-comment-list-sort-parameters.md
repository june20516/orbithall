# ADR-008: 공개 댓글 목록 정렬 파라미터

## Status
Accepted (2026-09-23). [ADR-004](004-comment-sorting-strategy.md)를 확장한다.

## Context
공개 목록 API는 최상위 댓글을 항상 `created_at ASC, id ASC`로 정렬했다(ADR-004). 위젯은 1페이지 50개만 요청하므로, 댓글이 50개를 넘으면 오래된 댓글만 보이고 새로 단 댓글은 화면에서 사라진 것처럼 보였다.

또한 "삭제됐고 살아 있는 대댓글도 없는" 최상위 댓글은 SQL로 페이지를 자른 뒤 핸들러에서 제외됐다. 그래서 `total_comments`가 실제 보이는 댓글보다 크고, 페이지가 `per_page`보다 짧거나 비어 있을 수 있었다.

## Decision

### 정렬 파라미터
`GET /api/posts/{slug}/comments`에 두 파라미터를 추가한다.

| 파라미터 | 허용 값 | 기본값 |
|---|---|---|
| `sort` | `created_at` | `created_at` |
| `direction` | `desc`, `asc` | `desc` |

- `desc`는 최상위 댓글을 `created_at DESC, id DESC`로, `asc`는 `created_at ASC, id ASC`로 정렬한다. ADR-004의 `id` 타이브레이커는 방향과 함께 뒤집어 유지한다.
- **대댓글은 방향과 무관하게 항상 오래된 순**이다. 대화 흐름은 시간 순서대로 읽는 것이 자연스럽다.
- 허용하지 않는 값은 400 `INVALID_INPUT`으로 거절하고, 어떤 파라미터가 문제인지 `details`에 담는다. `page`/`limit`은 가장 가까운 유효 값으로 고칠 수 있지만, 오타 난 정렬 값에는 "가까운 값"이 없다. 조용히 기본값으로 처리하면 순서가 통째로 뒤집혀도 알아채기 어렵다.
- 응답에 적용된 정렬을 담는다.

```json
{
  "comments": [],
  "sort": "created_at",
  "direction": "desc",
  "pagination": { "current_page": 1, "total_pages": 2, "total_comments": 58, "per_page": 50 }
}
```

### 기본값을 최신순으로
기본값을 `desc`로 두어, 파라미터를 모르는 클라이언트도 최신 댓글부터 받는다. 이 저장소의 공개 API를 쓰는 곳은 블로그(codeverse)에 붙은 위젯뿐이고, 배포된 1.1.1 위젯도 백엔드 배포 시점부터 최신순으로 보인다. 이는 의도한 방향이며, 한 페이지(50개) 안이라 잘리는 항목도 없다.

### 보이는 댓글 기준으로 세고 자르기
개수 조회와 목록 조회 모두 "보이는 최상위 댓글" 조건을 건다.

```sql
WHERE c.post_id = $1 AND c.parent_id IS NULL
  AND (c.is_deleted = FALSE
       OR EXISTS (SELECT 1 FROM comments r
                  WHERE r.post_id = $1 AND r.parent_id = c.id AND r.is_deleted = FALSE))
```

서브쿼리의 `post_id`를 바깥 컬럼(`c.post_id`)이 아니라 파라미터 `$1`로 고정한다. 상관 컬럼을 `c.id` 하나로 줄여야 플래너가 인덱스를 타기 때문이다. `c.post_id`로 쓰면 PostgreSQL이 hashed SubPlan으로 바꾸면서 `comments` 전체를 훑는다(측정: 21만 행에서 21.7ms / 4807 buffers → `$1`로 0.457ms / 137 buffers).

핸들러의 기존 필터는 삭제된 댓글의 작성자·내용을 비우고 IP를 마스킹하는 일을 계속 맡는다.

## Consequences
### Positive
- 위젯이 최신 댓글부터 보여 주고, "댓글 더 보기"로 이어서 불러올 수 있다.
- `total_comments`가 화면에 보이는 댓글 수와 맞아 "남은 개수"를 믿을 수 있다.
- 페이지가 `per_page`만큼 채워진다.

### Negative
- 기본 정렬이 바뀌므로, 오래된 순을 원하는 클라이언트는 `direction=asc`를 명시해야 한다.
- 목록 쿼리에 `EXISTS` 서브쿼리가 붙는다.

### Neutral
- 어드민 API(`GetAdminComments`)는 바꾸지 않는다. 삭제된 댓글도 모두 보여 줘야 하기 때문이다.

## Alternatives Considered
### `order=newest|oldest` 같은 자체 값
읽기는 쉽지만 널리 쓰이는 형식이 아니다. `sort` + `direction`은 GitHub API 등에서 쓰는 방식이라 처음 보는 사람도 짐작할 수 있다.

### `sort=-created_at` (JSON:API 형식)
파라미터 하나로 끝나지만, 기준과 방향이 한 값에 섞여 검증과 문서화가 덜 직관적이다.

### 잘못된 값을 기본값으로 처리
`page`/`limit`과 일관되지만, 정렬이 조용히 뒤집히는 편이 오류를 더 늦게 드러낸다.

## Related Decisions
- [ADR-004](004-comment-sorting-strategy.md): `created_at, id` 복합 정렬 (이 ADR에서 방향 선택을 더함)

## References
- `internal/database/comments.go`: `ListComments`, `topLevelOrderBy`, `visibleTopLevelCondition`
- `internal/handlers/comments.go`: `parseCommentSort`

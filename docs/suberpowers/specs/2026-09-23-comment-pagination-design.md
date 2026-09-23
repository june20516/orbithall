# 댓글 목록 최신순 정렬과 "댓글 더 보기" 설계

## 작성일
2026-09-23

## 배경 (2026-09-22~23 코드 확인)

- 위젯(`widget/src/components/CommentWidget.tsx`)은 폼이 맨 위, 목록이 아래다. 댓글은 `getComments(slug)`로 page=1, limit=50을 한 번만 요청하고 더 보기가 없다.
- 공개 목록 API는 최상위 댓글을 `created_at ASC, id ASC`로 정렬해 페이지로 나눈다(limit 최대 100, 초과 시 50). 응답에 `pagination`이 있고, 답글은 댓글마다 전부 딸려 온다.
- 그래서 오래된 최상위 댓글 50개만 보이고, 51번째 작성자는 자기 댓글이 사라진 것처럼 느낀다.
- 블로그(codeverse)는 위젯 `@1.1.1`을 쓴다. 이 저장소 외에 공개 API를 쓰는 곳은 없다.

### 코드에서 추가로 확인한 사실

1. **페이지 경계와 개수에 보이지 않는 댓글이 섞인다.** `internal/database/comments.go`의 `ListComments`는 모든 최상위 댓글을 세고 자르는데, "삭제됐고 살아 있는 답글도 없는" 댓글은 그 뒤 핸들러(`filterDeletedCommentsAndMaskIP`)에서 빠진다. `total_comments`가 실제보다 크고, 페이지가 limit보다 짧거나 비어 있을 수 있다.
2. **답글 작성도 최상위 작성과 같은 핸들러를 탄다.** `Comment.tsx`의 `onReply`가 `CommentWidget`의 `handleCommentSubmit`으로 이어진다.
3. **ADR-004는 "사용자 정의 정렬 순서 변경 불가(항상 생성 순)"라고 적고 있다.** 정렬 파라미터를 추가하면 ADR에 남겨야 한다.
4. **위젯에는 테스트가 없다.** DOM 없이 `bun test`로 돌릴 수 있는 구조가 필요하다.

### ADR-004의 `created_at ASC, id ASC`

`created_at`은 스키마 기본값 `NOW()`로 들어가고(ADR-005), `NOW()`는 트랜잭션 시작 시각이다. 통합 테스트는 트랜잭션 안에서 돌기 때문에(ADR-002) 한 테스트에서 만든 댓글의 `created_at`이 모두 같아 정렬이 흔들렸고, 그 대응으로 `id`를 두 번째 키로 넣었다. 운영에서는 요청마다 트랜잭션이 달라 값이 겹치지 않는다. 정렬 키를 유일하게 만드는 것은 페이지네이션에서 정석이므로 이번에도 유지한다. 방향을 바꿀 때는 두 키를 함께 뒤집는다.

## 결정

### 백엔드 (공개 목록 API만 변경, 어드민 API는 그대로)

- 파라미터를 두 개 추가한다.

| 파라미터 | 허용 값 | 기본값 |
|---|---|---|
| `sort` | `created_at` | `created_at` |
| `direction` | `desc`, `asc` | `desc` |

- 그 외의 값은 400 `INVALID_INPUT`으로 거절한다. `page`/`limit`처럼 가장 가까운 유효 값으로 고칠 수 없고, 조용히 기본값으로 처리하면 순서가 통째로 뒤집혀도 알아채기 어렵기 때문이다.
- `desc`면 최상위 댓글을 `created_at DESC, id DESC`로, `asc`면 지금처럼 `created_at ASC, id ASC`로 정렬한다. **답글은 방향과 무관하게 항상 `created_at ASC, id ASC`다.**
- **기본값을 `desc`(최신순)로 둔다.** 배포된 1.1.1 위젯도 백엔드 배포 시점부터 최신순으로 보인다. 목표하는 동작이고, 블로그 댓글은 한 페이지 안이라 잘리는 항목도 없다.
- 응답에 적용된 정렬을 함께 내린다.

```json
{
  "comments": [],
  "sort": "created_at",
  "direction": "desc",
  "pagination": { "current_page": 1, "total_pages": 2, "total_comments": 58, "per_page": 50 }
}
```

- **보이는 댓글 기준으로 세고 자른다.** 개수 조회와 목록 조회 모두에 조건을 넣는다.

```sql
WHERE c.post_id = $1 AND c.parent_id IS NULL
  AND (c.is_deleted = FALSE
       OR EXISTS (SELECT 1 FROM comments r WHERE r.parent_id = c.id AND r.is_deleted = FALSE))
```

핸들러의 기존 필터는 삭제된 댓글의 작성자·내용을 비우고 IP를 마스킹하는 일을 계속 맡는다.

### 위젯 (1.2.0)

- 폼은 맨 위 그대로, 목록은 `sort=created_at&direction=desc`를 명시해 최신순으로 받는다. 페이지 크기는 지금처럼 50이다.
- 목록 아래에 **"댓글 N개 더 보기"** 버튼을 둔다.
  - `current_page < total_pages`일 때만 보인다.
  - N은 남은 최상위 댓글 수(`total_comments - 불러온 최상위 댓글 수`)다. 위 SQL 변경으로 정확해진다.
  - 누르면 다음 페이지를 이어 붙인다. **id로 중복을 거른다.**
  - 불러오는 동안 비활성화하고 문구를 바꾼다. 실패하면 목록은 그대로 두고 버튼 아래에 오류를 보여 준다.
- 작성·수정·삭제 후 동작

| 상황 | 동작 | 이유 |
|---|---|---|
| 최상위 댓글 작성 | 1페이지만 다시 불러와 목록을 교체 | 최신순이라 내 댓글이 맨 위에 온다 |
| 답글 작성, 수정, 삭제 | 지금까지 펼친 1..N 페이지를 다시 불러와 교체 | 펼친 범위가 접히지 않는다 |

삭제 결과를 위젯에서 직접 반영하는 방식은, 서버의 "보이는 댓글" 규칙을 위젯에 복제해야 해서 택하지 않았다.

- 한국어·영어 문구를 추가한다: `comments.loadMore`(`댓글 {count}개 더 보기` / `Show {count} more comments`), `comments.loadingMore`(`불러오는 중...` / `Loading...`).
- 합치기·남은 개수·펼친 범위 다시 불러오기는 `widget/src/utils/commentPages.ts`의 순수 함수로 분리하고 `bun test`로 TDD한다. 화면은 로컬 브라우저로 확인한다.
- **삭제된 댓글 CSS 수정**: `widget/src/styles.css`의 `.orb-comment-deleted`는 흐림(opacity)과 기울임꼴을 하위 전체에 적용해, 삭제되지 않은 답글까지 삭제된 것처럼 보인다. 자식 선택자로 좁혀 삭제 자리만 흐리게 한다.

### 문서

| 파일 | 변경 |
|---|---|
| `docs/adr/008-comment-list-sort-parameters.md` (신규) | 정렬 파라미터, 기본값을 최신순으로 둔 이유와 영향, 보이는 댓글 기준 페이지 |
| `docs/adr/004-comment-sorting-strategy.md` | ADR-008에서 확장되었다는 한 줄 추가 (상태는 Accepted 유지) |
| `docs/adr/README.md` | 목록에 008 추가 |
| `internal/handlers/comments.go` | swagger 주석에 `sort`, `direction` 추가 |
| `README.md` | 목록 API 파라미터·응답 예시, 위젯 설치 주소 `@1.2.0` |
| `widget/README.md` | 설치 주소, 목록 동작(최신순·더 보기), 1.1.1 → 1.2.0 업그레이드 안내 |
| `docs/specs/widget-integration-guide.md` | 목록 동작과 예시 주소 |
| `docs/tasks/pending/008-widget-pagination.md` | 내용을 채워 completed로 이동 |

### 배포

1. `feat/comment-pagination`(origin/develop 기준, upstream 없음)에 백엔드·위젯·문서를 함께 담는다. publish 스크립트가 `origin/main`을 빌드하므로 위젯 코드와 버전도 같은 PR에 있어야 한다.
2. feature → develop → main. main 머지로 Render가 재배포되면 `/health`와 운영 API의 `direction=desc`/`asc`/잘못된 값 응답을 확인한다.
3. 사용자 확인을 받은 뒤 `bun run publish`로 `v1.2.0` 태그를 발행하고, `@1.2.0`의 응답 코드와 `x-jsd-version-type`을 curl로 확인한다.
4. 블로그 주소 변경은 codeverse 세션이 한다. 이 세션은 codeverse를 건드리지 않는다.

### 캡처용 로컬 환경 (구현 후)

- docker compose의 DB와 API(8080)는 계속 켜 둔다. 이미 떠 있는 4173 페이지도 그대로 둔다.
- 로컬 API를 가리키는 위젯 빌드와 시험 페이지를 scratchpad에 만들어 `http://localhost:5500`으로 띄우고, 로컬 테스트 사이트(id 21)의 `cors_origins`에 추가한다. 저장소 파일은 바꾸지 않는다.
- `orbithall-widget-demo` 포스트의 기존 댓글을 지우고 다시 넣는다. 보이는 최상위 댓글 58개(더 보기 버튼이 보이도록), 그중 몇 개에 답글 1~3개, 답글이 달린 채 삭제된 댓글 하나, 수정된 댓글 하나. 자연스러운 한국어 이름과 내용.
- 시드 스크립트는 저장소에 남기지 않고 scratchpad에만 둔다.
- 운영 DB에는 쓰지 않는다.

## 한계

- offset 방식이라, 불러오는 사이에 다른 사람이 댓글을 지우면 경계가 당겨져 한 개를 놓칠 수 있다. 블로그 규모에서는 드물어 수용한다. 커서 방식이 더 견고하지만 이번 범위에는 과하다.
- 남은 개수는 최상위 댓글 기준이다. 답글 수는 세지 않는다.

## 검증

- 백엔드(TDD, 트랜잭션 기반 통합 테스트)
  - 기본값이 최신순이고 응답에 `sort`, `direction`이 있다
  - `direction=asc`는 기존과 같은 오래된 순
  - 답글은 방향과 무관하게 오래된 순
  - `sort`/`direction`에 잘못된 값 → 400 `INVALID_INPUT`
  - 보이지 않는 삭제 댓글은 `total_comments`와 페이지 경계에서 빠진다
  - ASC를 전제하던 기존 테스트는 `direction=asc`를 명시하도록 고친다
- 위젯: `bun test`로 순수 함수(중복 제거 합치기, 남은 개수, 펼친 범위 다시 불러오기)를 검증하고, 로컬 페이지에서 더 보기·작성·수정·삭제 흐름과 삭제 댓글 답글 표시를 눈으로 확인한다
- 배포 후: `/health` 200, 운영 API 정렬 응답, `@1.2.0` CDN 응답

## 범위 밖

- 대댓글 페이지네이션(작업 007)
- 어드민 API와 화면
- 커서 기반 페이지네이션
- codeverse 저장소 변경

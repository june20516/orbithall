# 어드민 댓글 삭제 API 설계

## 작성일
2026-09-23

## 배경 (2026-09-23 코드 확인)

- 댓글 삭제는 공개 API `DELETE /api/comments/{id}`뿐이고, 작성자 비밀번호와 작성 후 30분 이내 조건이 필요하다. 사이트 소유자가 스팸·부적절한 댓글을 정리할 방법이 없다.
- 작업 문서: `docs/tasks/pending/p1-018-admin-comment-delete-api.md`. 어드민 대응 과제는 orbithall-admin `docs/pending/admin-comment-delete.md`.

### 코드에서 확인한 사실

1. **어드민 핸들러의 공통 흐름.** `internal/handlers/admin.go`의 핸들러는 context에서 사용자 추출 → `database.HasUserSiteAccess`로 권한 확인 → 작업 순서다. 에러는 `http.Error`로 평문을 보내고, 삭제 성공은 `204 No Content`다(`DeleteSite`).
2. **`HasUserSiteAccess`는 지금 `role = 'owner'`만 허용한다**(`internal/database/user_sites.go:145`). 이 작업에서 그대로 쓰면 소유자만 삭제할 수 있다. 매니저 역할(p5-014)이 들어올 때 조회와 수정 권한을 나눈다.
3. **`database.DeleteComment`는 soft delete다.** `is_deleted = TRUE`, `deleted_at`을 기록하고 `is_deleted = FALSE`인 행만 갱신한다. 갱신이 0건이면 `fmt.Errorf("comment not found or already deleted")`를 돌려주어 호출자가 원인을 구분할 수 없다.
4. **댓글에는 `site_id`가 없다.** 댓글 → 포스트(`GetPostByID`) → 사이트 순서로 따라가야 권한을 확인할 수 있다.
5. **통계와 게시글 목록은 `is_deleted` 기준으로 센다**(`internal/database/sites.go:145`, `internal/database/posts.go:167`). 삭제 결과가 코드 변경 없이 반영된다.
6. **대댓글은 부모 삭제와 무관하게 남는다.** soft delete라 행이 그대로 있고, 공개 목록은 "삭제됐지만 살아 있는 답글이 있는" 부모를 내용을 비운 채 보여 준다.

## 결정

### API

`DELETE /admin/comments/{id}` (JWT 인증, `/admin` 라우트 그룹), 핸들러 `AdminHandler.DeleteComment`.

| 상황 | 응답 |
|---|---|
| `id`가 양의 정수가 아님 | 400 `Invalid comment ID` |
| context에 사용자 없음 | 401 `Unauthorized` |
| 댓글이 없음 | 404 `Comment not found` |
| 댓글의 사이트에 권한 없음 | 403 `Forbidden` |
| 삭제 성공 | 204 |
| 이미 삭제된 댓글 | 204 (멱등) |
| DB 오류 | 500 |

에러 본문은 기존 어드민 핸들러처럼 `http.Error` 평문이다.

### 처리 순서

1. 사용자 추출, `id` 파싱
2. `GetCommentByID` → 없으면 404
3. `GetPostByID(comment.PostID)` → 사이트 ID 확보 (댓글은 FK로 포스트에 묶여 있어 없으면 500)
4. `HasUserSiteAccess(user.ID, post.SiteID)` → 없으면 403
5. `comment.IsDeleted`면 204
6. `DeleteComment` → 성공이면 204, `errors.Is(err, database.ErrCommentNotFound)`면 204, 그 외 500

**이미 삭제된 댓글 여부는 권한 확인 뒤에 판단한다.** 권한 없는 사용자에게 삭제 상태를 알려 주지 않기 위해서다.

**재삭제를 204로 처리하는 이유.** 결과 상태(삭제됨)가 같고, 어드민에서 두 번 클릭하거나 재시도해도 에러가 나지 않는다.

기존 함수 네 개를 조합한다. 어드민 삭제는 한 건씩 오는 요청이라 쿼리 수보다 기존 핸들러와 같은 흐름이 더 중요하다. "댓글 → 사이트 ID" 전용 조회나 소유권을 조건으로 건 단일 UPDATE는 쓰지 않는다. 후자는 0건일 때 404·403·204를 구분할 수 없다.

### 동시 삭제 경합

두 요청이 동시에 오면 둘 다 2단계에서 삭제 전 상태를 보고 6단계로 가고, 늦은 쪽이 0건 갱신을 받는다. 이 경우도 204여야 하므로 `DeleteComment`의 0건 에러를 `database.ErrCommentNotFound`로 감싼다.

```go
if rowsAffected == 0 {
	return fmt.Errorf("%w: not found or already deleted", ErrCommentNotFound)
}
```

기존 호출부(공개 삭제 핸들러, 테스트)는 `err != nil`만 확인하므로 동작이 바뀌지 않는다.

### 테스트

`internal/handlers/admin_test.go` (트랜잭션 기반, ADR-002):

- 성공: 204, DB의 `is_deleted = TRUE`, `deleted_at` 기록
- 이미 삭제된 댓글: 204
- 없는 댓글: 404
- 다른 사용자 사이트의 댓글: 403, 댓글은 삭제되지 않음
- 잘못된 `id`(`abc`, `0`): 400
- 사용자 context 없음: 401
- 대댓글이 달린 부모 삭제: 대댓글은 `is_deleted = FALSE` 유지
- 삭제 후 `GetSiteStats`의 삭제 댓글 수 증가

`internal/database/comments_test.go`:

- 이미 삭제된 댓글과 없는 댓글에 `DeleteComment` → `errors.Is(err, ErrCommentNotFound)`

### 그 밖의 변경

- `cmd/api/main.go`: `/admin` 그룹에 `r.Delete("/comments/{id}", adminHandler.DeleteComment)`
- swagger 주석 추가 후 `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go` 재생성
- 작업 문서: 시작 시 `docs/tasks/active/018-admin-comment-delete-api.md`로 옮기고, 완료 후 결과를 기록해 `completed/`로 이동

## 범위 밖

- 삭제 취소(복구), 영구 삭제(hard delete), 여러 댓글 일괄 삭제
- 매니저 역할과 조회·수정 권한 분리 (p5-014)
- 어드민 UI (orbithall-admin)

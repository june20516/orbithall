# 어드민 댓글 삭제 API Implementation Plan

> **agentic worker에게:** REQUIRED SUB-SKILL: 이 plan을 task 단위로 구현하려면 suberpower:subagent-driven-development(권장) 또는 suberpower:executing-plans를 사용하세요. Step은 추적을 위해 checkbox(`- [ ]`) 문법을 사용합니다.

**Goal:** 사이트 소유자가 어드민에서 `DELETE /admin/comments/{id}`로 댓글을 soft delete할 수 있게 한다.

**Architecture:** `AdminHandler.DeleteComment`가 기존 함수 `GetCommentByID` → `GetPostByID` → `HasUserSiteAccess` → `DeleteComment`를 차례로 호출한다. 이미 삭제된 댓글과 동시 삭제 경합은 204로 처리하고, 경합을 구분하려고 `database.DeleteComment`의 0건 에러를 `database.ErrCommentNotFound`로 감싼다.

**Tech Stack:** Go 1.25, chi v5, PostgreSQL 18, swaggo(`~/go/bin/swag`)

**Spec:** `docs/suberpowers/specs/2026-09-23-admin-comment-delete-design.md`

**작업 브랜치:** `feat/admin-comment-delete` (이미 생성됨, upstream 없음)

**테스트 전제:** `docker compose ps`에서 `orbithall-db`가 healthy여야 한다. 테스트는 로컬 Go로 돌린다(`go test ./internal/...`).

---

## 파일 구조

| 파일 | 변경 | 책임 |
|---|---|---|
| `docs/tasks/pending/p1-018-admin-comment-delete-api.md` → `docs/tasks/active/018-admin-comment-delete-api.md` | 이동 | 작업 시작 표시 |
| `internal/database/comments.go` | 수정 (`DeleteComment`) | 0건 에러를 sentinel로 감쌈 |
| `internal/database/comments_test.go` | 수정 (`TestDeleteComment`) | sentinel 검증 |
| `internal/handlers/admin.go` | 수정 (끝에 핸들러 추가, import `errors`) | 어드민 댓글 삭제 |
| `internal/handlers/admin_test.go` | 수정 (끝에 테스트 추가) | 핸들러 테스트 |
| `cmd/api/main.go` | 수정 (`/admin` 그룹) | 라우트 등록 |
| `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` | 재생성 | API 문서 |
| `docs/tasks/active/018-...` → `docs/tasks/completed/018-...` | 이동·수정 | 결과 기록 |

---

### Task 1: 작업 문서를 active로 이동

**Files:**
- Move: `docs/tasks/pending/p1-018-admin-comment-delete-api.md` → `docs/tasks/active/018-admin-comment-delete-api.md`

- [ ] **Step 1: 이동**

실행: `git mv docs/tasks/pending/p1-018-admin-comment-delete-api.md docs/tasks/active/018-admin-comment-delete-api.md`

- [ ] **Step 2: Commit**

```bash
git commit -m "docs: 018 어드민 댓글 삭제 작업 시작"
```

---

### Task 2: `DeleteComment` 0건 에러를 `ErrCommentNotFound`로 감싸기

**Files:**
- Modify: `internal/database/comments.go` (`DeleteComment`, 약 175~200행)
- Test: `internal/database/comments_test.go` (`TestDeleteComment`, 약 402~484행)

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/database/comments_test.go`의 `TestDeleteComment` 안, "존재하지 않는 댓글 삭제 시 에러 반환" 서브테스트의 `// Then` 부분을 다음으로 바꾼다.

```go
		// Then: ErrCommentNotFound 반환
		if !errors.Is(err, ErrCommentNotFound) {
			t.Fatalf("expected ErrCommentNotFound, got: %v", err)
		}
```

"이미 삭제된 댓글 삭제 시 에러 반환" 서브테스트의 `// Then` 부분도 다음으로 바꾼다.

```go
		// Then: ErrCommentNotFound 반환
		if !errors.Is(err, ErrCommentNotFound) {
			t.Fatalf("expected ErrCommentNotFound, got: %v", err)
		}
```

파일 상단 import에 `"errors"`가 없으면 추가한다.

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/database/ -run TestDeleteComment -count=1 -v`
기대: 두 서브테스트가 `expected ErrCommentNotFound, got: comment not found or already deleted`로 FAIL

- [ ] **Step 3: 구현**

`internal/database/comments.go`의 `DeleteComment`에서 0건 분기를 바꾼다.

```go
	if rowsAffected == 0 {
		return fmt.Errorf("%w: not found or already deleted", ErrCommentNotFound)
	}
```

함수 주석 마지막 줄을 다음으로 바꾼다.

```go
// 이미 삭제됐거나 없는 댓글이면 ErrCommentNotFound를 감싼 에러를 반환합니다
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/database/ -count=1`
기대: `ok  	github.com/june20516/orbithall/internal/database`

실행: `go test ./internal/handlers/ -count=1`
기대: `ok` (공개 삭제 핸들러는 `err != nil`만 보므로 영향 없음)

- [ ] **Step 5: Commit**

```bash
git add internal/database/comments.go internal/database/comments_test.go
git commit -m "refactor: 댓글 삭제 0건 에러를 ErrCommentNotFound로 감싸기"
```

---

### Task 3: `AdminHandler.DeleteComment` 핸들러 (TDD)

**Files:**
- Modify: `internal/handlers/admin.go` (파일 끝에 추가, import에 `"errors"` 추가)
- Test: `internal/handlers/admin_test.go` (파일 끝에 추가)

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/handlers/admin_test.go` 끝에 추가한다. 이 파일은 이미 `bytes`, `context`, `encoding/json`, `net/http`, `net/http/httptest`, `strconv`, `testing`, `chi`, `database`, `models`, `testhelpers`를 import한다.

```go
// adminDeleteFixture는 어드민 댓글 삭제 테스트용 사용자·사이트·포스트입니다
type adminDeleteFixture struct {
	owner *models.User
	site  *models.Site
	post  models.Post
}

// setupAdminDeleteFixture는 소유자와 소유자의 사이트, 포스트를 만듭니다
func setupAdminDeleteFixture(ctx context.Context, t *testing.T, tx database.DBTX) adminDeleteFixture {
	t.Helper()

	owner := &models.User{Email: "owner@example.com", Name: "Owner", GoogleID: "google-owner"}
	if err := database.CreateUser(ctx, tx, owner); err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}

	site := &models.Site{
		Name:        "Owner Blog",
		Domain:      "owner.com",
		CORSOrigins: []string{"https://owner.com"},
		IsActive:    true,
	}
	if err := database.CreateSiteForUser(ctx, tx, site, owner.ID); err != nil {
		t.Fatalf("Failed to create site: %v", err)
	}

	post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "post-1", "Post 1")
	return adminDeleteFixture{owner: owner, site: site, post: post}
}

// requestAdminDeleteComment는 user를 context에 넣고 DELETE /admin/comments/{id}를 호출합니다
// user가 nil이면 context에 사용자를 넣지 않습니다
func requestAdminDeleteComment(ctx context.Context, tx database.DBTX, user *models.User, commentID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/admin/comments/"+commentID, nil)
	if user != nil {
		ctx = context.WithValue(ctx, userContextKey, user)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", commentID)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	NewAdminHandler(tx).DeleteComment(rec, req)
	return rec
}

// TestAdminDeleteComment는 어드민 댓글 삭제 기능을 테스트합니다
func TestAdminDeleteComment(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("삭제 성공 - 204, soft delete 기록", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "spammer", "pass", "spam", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// When: 소유자가 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(comment.ID, 10))

		// Then: 204, is_deleted와 deleted_at 기록
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
		deleted, err := database.GetCommentByID(ctx, tx, comment.ID)
		if err != nil {
			t.Fatalf("Failed to get comment: %v", err)
		}
		if !deleted.IsDeleted {
			t.Error("Expected is_deleted=true, got false")
		}
		if deleted.DeletedAt == nil {
			t.Error("Expected deleted_at to be set, got nil")
		}
	})

	t.Run("이미 삭제된 댓글 - 204 멱등", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 이미 삭제된 댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if err := database.DeleteComment(ctx, tx, comment.ID); err != nil {
			t.Fatalf("Failed to delete comment: %v", err)
		}

		// When: 다시 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(comment.ID, 10))

		// Then: 204
		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
	})

	t.Run("없는 댓글 - 404", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자만 존재
		f := setupAdminDeleteFixture(ctx, t, tx)

		// When: 없는 ID 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, "999999999")

		// Then: 404
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("다른 사용자 사이트의 댓글 - 403, 삭제되지 않음", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 댓글과 사이트가 없는 다른 사용자
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		stranger := &models.User{Email: "stranger@example.com", Name: "Stranger", GoogleID: "google-stranger"}
		if err := database.CreateUser(ctx, tx, stranger); err != nil {
			t.Fatalf("Failed to create stranger: %v", err)
		}

		// When: 다른 사용자가 삭제
		rec := requestAdminDeleteComment(ctx, tx, stranger, strconv.FormatInt(comment.ID, 10))

		// Then: 403, 댓글은 그대로
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, rec.Code, rec.Body.String())
		}
		unchanged, err := database.GetCommentByID(ctx, tx, comment.ID)
		if err != nil {
			t.Fatalf("Failed to get comment: %v", err)
		}
		if unchanged.IsDeleted {
			t.Error("Expected comment to remain, but it was deleted")
		}
	})

	t.Run("다른 사용자 사이트의 삭제된 댓글 - 403", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 삭제된 댓글과 다른 사용자
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if err := database.DeleteComment(ctx, tx, comment.ID); err != nil {
			t.Fatalf("Failed to delete comment: %v", err)
		}
		stranger := &models.User{Email: "stranger@example.com", Name: "Stranger", GoogleID: "google-stranger"}
		if err := database.CreateUser(ctx, tx, stranger); err != nil {
			t.Fatalf("Failed to create stranger: %v", err)
		}

		// When: 다른 사용자가 삭제
		rec := requestAdminDeleteComment(ctx, tx, stranger, strconv.FormatInt(comment.ID, 10))

		// Then: 삭제 여부를 드러내지 않고 403
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, rec.Code, rec.Body.String())
		}
	})

	t.Run("잘못된 ID - 400", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자
		f := setupAdminDeleteFixture(ctx, t, tx)

		for _, invalidID := range []string{"abc", "0", "-1"} {
			// When: 잘못된 ID로 삭제
			rec := requestAdminDeleteComment(ctx, tx, f.owner, invalidID)

			// Then: 400
			if rec.Code != http.StatusBadRequest {
				t.Errorf("id=%q: Expected status %d, got %d", invalidID, http.StatusBadRequest, rec.Code)
			}
		}
	})

	t.Run("사용자 context 없음 - 401", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// When: 사용자 없이 삭제
		rec := requestAdminDeleteComment(ctx, tx, nil, strconv.FormatInt(comment.ID, 10))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("대댓글이 달린 부모 삭제 - 대댓글 유지", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 부모 댓글과 대댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		parent, err := database.CreateComment(ctx, tx, f.post.ID, nil, "parent", "pass", "parent", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create parent: %v", err)
		}
		reply, err := database.CreateComment(ctx, tx, f.post.ID, &parent.ID, "reply", "pass", "reply", "2.2.2.2", "ua")
		if err != nil {
			t.Fatalf("Failed to create reply: %v", err)
		}

		// When: 부모 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(parent.ID, 10))

		// Then: 204, 대댓글은 삭제되지 않음
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
		remaining, err := database.GetCommentByID(ctx, tx, reply.ID)
		if err != nil {
			t.Fatalf("Failed to get reply: %v", err)
		}
		if remaining.IsDeleted {
			t.Error("Expected reply to remain, but it was deleted")
		}
	})

	t.Run("삭제 후 사이트 통계에 반영", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 활성 댓글 2개
		f := setupAdminDeleteFixture(ctx, t, tx)
		target, err := database.CreateComment(ctx, tx, f.post.ID, nil, "a", "pass", "a", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if _, err := database.CreateComment(ctx, tx, f.post.ID, nil, "b", "pass", "b", "2.2.2.2", "ua"); err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// When: 한 개 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(target.ID, 10))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}

		// Then: 활성 1, 삭제 1
		stats, err := database.GetSiteStats(ctx, tx, f.site.ID)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		if stats.CommentCount != 1 || stats.DeletedCommentCount != 1 {
			t.Errorf("Expected comment_count=1, deleted_comment_count=1, got %d, %d", stats.CommentCount, stats.DeletedCommentCount)
		}
	})
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/handlers/ -run TestAdminDeleteComment -count=1`
기대: `NewAdminHandler(tx).DeleteComment undefined` 컴파일 에러로 FAIL

- [ ] **Step 3: 구현**

`internal/handlers/admin.go` import 블록에 `"errors"`를 추가한다.

```go
import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/validators"
)
```

파일 끝에 추가한다.

```go
// DeleteComment는 사이트 소유자가 댓글을 soft delete합니다
// 이미 삭제된 댓글도 204를 반환합니다 (멱등)
// @Summary      댓글 삭제
// @Description  사이트 소유자가 댓글을 삭제합니다 (soft delete). 대댓글은 유지되며, 이미 삭제된 댓글도 204를 반환합니다
// @Tags         admin
// @Produce      plain
// @Param        id path int true "Comment ID"
// @Success      204 "No Content"
// @Failure      400 {string} string "Invalid comment ID"
// @Failure      401 {string} string "Unauthorized"
// @Failure      403 {string} string "Forbidden"
// @Failure      404 {string} string "Comment not found"
// @Failure      500 {string} string "Failed to delete comment"
// @Security     BearerAuth
// @Router       /admin/comments/{id} [delete]
func (h *AdminHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URL 파라미터에서 comment_id 추출
	commentID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || commentID <= 0 {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	// 댓글 조회 (삭제된 댓글 포함)
	comment, err := database.GetCommentByID(r.Context(), h.db, commentID)
	if err != nil {
		http.Error(w, "Failed to get comment", http.StatusInternalServerError)
		return
	}
	if comment == nil {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	// 댓글이 속한 사이트 확인 (댓글은 FK로 포스트에 묶여 있으므로 포스트가 없으면 서버 오류)
	post, err := database.GetPostByID(r.Context(), h.db, comment.PostID)
	if err != nil || post == nil {
		http.Error(w, "Failed to get post", http.StatusInternalServerError)
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, post.SiteID)
	if err != nil {
		http.Error(w, "Failed to check site access", http.StatusInternalServerError)
		return
	}
	if !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 이미 삭제된 댓글은 권한 확인 후에 멱등 처리
	if comment.IsDeleted {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 삭제 (동시 요청으로 먼저 삭제된 경우도 ErrCommentNotFound로 오므로 성공 처리)
	err = database.DeleteComment(r.Context(), h.db, commentID)
	if err != nil && !errors.Is(err, database.ErrCommentNotFound) {
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}

	// 204 No Content 응답
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/handlers/ -run TestAdminDeleteComment -count=1 -v`
기대: 서브테스트 9개 모두 PASS

실행: `go test ./... -count=1`
기대: 모든 패키지 `ok` (DB 없는 패키지는 `no test files`)

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/admin.go internal/handlers/admin_test.go
git commit -m "feat: 어드민 댓글 삭제 핸들러 추가"
```

---

### Task 4: 라우트 등록과 swagger 재생성

**Files:**
- Modify: `cmd/api/main.go` (`/admin` 라우트 그룹, 약 200~202행)
- Regenerate: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`

- [ ] **Step 1: 라우트 추가**

`cmd/api/main.go`의 `/admin` 그룹에서 `r.Get("/posts/{slug}/comments", adminHandler.GetPostComments)` 바로 아래에 추가한다.

```go
		r.Delete("/comments/{id}", adminHandler.DeleteComment)
```

- [ ] **Step 2: 빌드 확인**

실행: `go build ./...`
기대: 출력 없이 성공

- [ ] **Step 3: swagger 재생성**

실행: `~/go/bin/swag init -g cmd/api/main.go --output ./docs`
기대: `create docs.go at docs/docs.go` 등 3개 파일 생성 로그

실행: `git diff --stat docs/docs.go docs/swagger.json docs/swagger.yaml`
기대: 3개 파일에 `/admin/comments/{id}` delete 경로만 추가됨. 다른 경로가 크게 바뀌었다면 swag 버전 차이이므로 되돌리고(`git checkout docs/docs.go docs/swagger.json docs/swagger.yaml`) 사용자에게 알린다.

실행: `grep -n '"/admin/comments/{id}"' docs/swagger.json`
기대: 1줄 이상

- [ ] **Step 4: 라우팅 수동 확인**

API 컨테이너는 air로 재시작된다(`.air.toml`). JWT 없이 호출해 라우트가 JWT 미들웨어 뒤에 있는지 확인한다.

실행: `curl -s -o /dev/null -w "%{http_code}\n" -X DELETE http://localhost:8080/admin/comments/1`
기대: `401` (404라면 라우트가 등록되지 않은 것)

- [ ] **Step 5: Commit**

```bash
git add cmd/api/main.go docs/docs.go docs/swagger.json docs/swagger.yaml
git commit -m "feat: 어드민 댓글 삭제 라우트와 API 문서 추가"
```

---

### Task 5: 작업 문서 완료 처리

**Files:**
- Move/Modify: `docs/tasks/active/018-admin-comment-delete-api.md` → `docs/tasks/completed/018-admin-comment-delete-api.md`

- [ ] **Step 1: 결과 기록**

`docs/tasks/active/018-admin-comment-delete-api.md`의 "이미 삭제된 댓글은 409 또는 멱등 처리 중 선택" 줄을 "이미 삭제된 댓글은 204 멱등 처리"로 바꾸고, 문서 끝에 추가한다.

```markdown
## 결과 (2026-09-23)

- `DELETE /admin/comments/{id}` 추가 (`AdminHandler.DeleteComment`)
  - 400 잘못된 ID, 401 사용자 없음, 404 없는 댓글, 403 권한 없음, 204 삭제 성공·이미 삭제됨
  - 이미 삭제됐는지는 권한 확인 뒤에 판단 (권한 없는 사용자에게 삭제 여부를 드러내지 않음)
- `database.DeleteComment`의 0건 에러를 `ErrCommentNotFound`로 감싸 동시 삭제 경합도 204로 처리
- 대댓글 유지, 사이트 통계·게시글 목록 삭제 수 반영은 테스트로 확인 (코드 변경 없음)
- 설계: `docs/suberpowers/specs/2026-09-23-admin-comment-delete-design.md`
```

- [ ] **Step 2: 이동**

실행: `git mv docs/tasks/active/018-admin-comment-delete-api.md docs/tasks/completed/018-admin-comment-delete-api.md`

- [ ] **Step 3: Commit**

```bash
git add docs/tasks
git commit -m "docs: 018 어드민 댓글 삭제 작업 완료 기록"
```

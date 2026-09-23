# 댓글 최신순 정렬과 "댓글 더 보기" Implementation Plan

> **agentic worker에게:** REQUIRED SUB-SKILL: 이 plan을 task 단위로 구현하려면 suberpower:subagent-driven-development(권장) 또는 suberpower:executing-plans를 사용하세요. Step은 추적을 위해 checkbox(`- [ ]`) 문법을 사용합니다.

**Goal:** 공개 댓글 목록 API에 `sort`/`direction`을 추가해 기본을 최신순으로 바꾸고, 위젯 1.2.0에 "댓글 더 보기"를 넣는다. 삭제된 댓글의 답글이 흐리게 보이는 CSS도 함께 고친다.

**Architecture:** 백엔드는 보이는 댓글만 세고 자르도록 SQL을 고치고 정렬 방향을 파라미터로 받는다. 위젯은 페이지를 이어 붙이는 순수 함수(`commentPages.ts`)를 만들어 `bun test`로 검증하고, `CommentWidget`이 그 함수로 상태를 관리한다.

**Tech Stack:** Go 1.x (chi, database/sql), PostgreSQL 18, Preact + TypeScript, Bun

**Spec:** `docs/suberpowers/specs/2026-09-23-comment-pagination-design.md`

**작업 브랜치:** `feat/comment-pagination` (origin/develop 기준, upstream 없음)

---

## 파일 구조

| 파일 | 변경 | 책임 |
|---|---|---|
| `internal/database/comments.go` | 수정 | 정렬 방향 타입, 보이는 댓글 조건, `ListComments` 시그니처 |
| `internal/database/comments_test.go` | 수정 | 기존 호출에 방향 인자 추가, 방향별 정렬 테스트 |
| `internal/handlers/comments.go` | 수정 | `sort`/`direction` 파싱·검증, 응답 필드, swagger 주석 |
| `internal/handlers/comments_test.go` | 수정 | 기본 최신순, `asc`, 잘못된 값 400, 보이는 댓글 기준 페이지 |
| `widget/src/types.ts` | 수정 | `CommentsPagination`, `CommentsResponse`, `SortDirection` |
| `widget/src/api/client.ts` | 수정 | `getComments`에 방향 전달 |
| `widget/src/utils/commentPages.ts` | 생성 | 페이지 합치기·남은 개수·다시 불러오기 (순수 함수) |
| `widget/src/utils/commentPages.test.ts` | 생성 | 위 함수의 테스트 |
| `widget/src/components/CommentWidget.tsx` | 수정 | 페이지 상태, 더 보기 버튼, 작성/수정/삭제 후 동작 |
| `widget/src/i18n/locales.ts` | 수정 | 더 보기 문구(ko/en) |
| `widget/src/styles.css` | 수정 | 더 보기 버튼 영역, 삭제된 댓글 선택자 수정 |
| `widget/package.json` | 수정 | version 1.2.0, test 스크립트 |
| 문서 | 수정 | ADR-008 외 spec의 문서 표 참조 |

커밋 메시지는 모두 다음 두 줄로 끝낸다.

```
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
```

Go 테스트는 로컬 docker DB(`orbithall-db`)의 `test_orbithall_db`를 쓴다. 이미 떠 있으며, `.env`의 `TEST_DATABASE_URL`을 `internal/testhelpers`가 읽는다.

---

### Task 1: DB 계층 — 정렬 방향과 보이는 댓글 기준

**Files:**
- Modify: `internal/database/comments.go:165-232` (`ListComments`)
- Test: `internal/database/comments_test.go:486-660` (`TestListComments`)

- [ ] **Step 1: 기존 테스트의 호출부를 새 시그니처로 바꾸고 실패를 확인**

`internal/database/comments_test.go`에서 `ListComments(ctx, tx, postID, ` 로 시작하는 호출 6곳의 마지막에 방향 인자를 추가한다. 기존 기대값이 오래된 순이므로 전부 `SortAsc`를 넘긴다.

```go
comments, total, err := ListComments(ctx, tx, postID, 10, 0, SortAsc)
```
```go
page1, total1, err := ListComments(ctx, tx, postID, 2, 0, SortAsc)
```
```go
page2, total2, err := ListComments(ctx, tx, postID, 2, 2, SortAsc)
```
```go
page3, total3, err := ListComments(ctx, tx, postID, 2, 4, SortAsc)
```

(6곳 모두 동일한 방식으로 `, SortAsc`를 덧붙인다.)

실행: `go build ./... 2>&1 | head -5`
기대: `SortAsc` 미정의와 인자 개수 불일치로 컴파일 실패

- [ ] **Step 2: 새 동작을 검증하는 테스트를 `TestListComments`의 마지막 `t.Run` 뒤에 추가**

```go
	t.Run("최신순 정렬은 최상위 댓글을 뒤집고 대댓글은 오래된 순을 유지", func(t *testing.T) {
		// Given: 최상위 댓글 2개와 첫 댓글의 대댓글 2개
		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-desc", "Test Post").ID

		first, err := CreateComment(ctx, tx, postID, nil, "Author1", "pass", "First", "10.0.0.1", "Agent")
		if err != nil {
			t.Fatalf("failed to create first: %v", err)
		}
		second, err := CreateComment(ctx, tx, postID, nil, "Author2", "pass", "Second", "10.0.0.2", "Agent")
		if err != nil {
			t.Fatalf("failed to create second: %v", err)
		}
		reply1, err := CreateComment(ctx, tx, postID, &first.ID, "Reply1", "pass", "Reply 1", "10.0.0.3", "Agent")
		if err != nil {
			t.Fatalf("failed to create reply1: %v", err)
		}
		reply2, err := CreateComment(ctx, tx, postID, &first.ID, "Reply2", "pass", "Reply 2", "10.0.0.4", "Agent")
		if err != nil {
			t.Fatalf("failed to create reply2: %v", err)
		}

		// When: 최신순으로 조회
		comments, total, err := ListComments(ctx, tx, postID, 10, 0, SortDesc)

		// Then: 최상위는 나중에 만든 댓글이 먼저, 대댓글은 만든 순서 그대로
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total=2, got %d", total)
		}
		if len(comments) != 2 {
			t.Fatalf("expected 2 top-level comments, got %d", len(comments))
		}
		if comments[0].ID != second.ID {
			t.Errorf("expected newest comment first, got id %d", comments[0].ID)
		}
		if comments[1].ID != first.ID {
			t.Errorf("expected oldest comment last, got id %d", comments[1].ID)
		}
		if len(comments[1].Replies) != 2 {
			t.Fatalf("expected 2 replies, got %d", len(comments[1].Replies))
		}
		if comments[1].Replies[0].ID != reply1.ID || comments[1].Replies[1].ID != reply2.ID {
			t.Errorf("expected replies in oldest-first order, got %d, %d",
				comments[1].Replies[0].ID, comments[1].Replies[1].ID)
		}
	})

	t.Run("보이지 않는 삭제 댓글은 개수와 페이지에서 제외", func(t *testing.T) {
		// Given: 살아 있는 댓글 2개, 답글 없이 삭제된 댓글 1개, 살아 있는 답글이 있는 삭제된 댓글 1개
		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-visible", "Test Post").ID

		alive1, err := CreateComment(ctx, tx, postID, nil, "Alive1", "pass", "Alive 1", "10.0.0.1", "Agent")
		if err != nil {
			t.Fatalf("failed to create alive1: %v", err)
		}
		lonelyDeleted, err := CreateComment(ctx, tx, postID, nil, "Lonely", "pass", "Lonely", "10.0.0.2", "Agent")
		if err != nil {
			t.Fatalf("failed to create lonelyDeleted: %v", err)
		}
		if err := DeleteComment(ctx, tx, lonelyDeleted.ID); err != nil {
			t.Fatalf("failed to delete lonelyDeleted: %v", err)
		}
		deletedWithReply, err := CreateComment(ctx, tx, postID, nil, "Parent", "pass", "Parent", "10.0.0.3", "Agent")
		if err != nil {
			t.Fatalf("failed to create deletedWithReply: %v", err)
		}
		if _, err := CreateComment(ctx, tx, postID, &deletedWithReply.ID, "Reply", "pass", "Reply", "10.0.0.4", "Agent"); err != nil {
			t.Fatalf("failed to create reply: %v", err)
		}
		if err := DeleteComment(ctx, tx, deletedWithReply.ID); err != nil {
			t.Fatalf("failed to delete deletedWithReply: %v", err)
		}
		alive2, err := CreateComment(ctx, tx, postID, nil, "Alive2", "pass", "Alive 2", "10.0.0.5", "Agent")
		if err != nil {
			t.Fatalf("failed to create alive2: %v", err)
		}

		// When: 오래된 순으로 조회
		comments, total, err := ListComments(ctx, tx, postID, 10, 0, SortAsc)

		// Then: 답글 없이 삭제된 댓글만 빠진다
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total=3 (lonely deleted excluded), got %d", total)
		}
		if len(comments) != 3 {
			t.Fatalf("expected 3 comments, got %d", len(comments))
		}
		ids := []int64{comments[0].ID, comments[1].ID, comments[2].ID}
		expected := []int64{alive1.ID, deletedWithReply.ID, alive2.ID}
		for i := range expected {
			if ids[i] != expected[i] {
				t.Errorf("index %d: expected id %d, got %d", i, expected[i], ids[i])
			}
		}
	})
```

- [ ] **Step 3: 테스트 실행으로 실패 확인**

실행: `go test ./internal/database/ -run TestListComments -count=1 2>&1 | tail -5`
기대: 컴파일 에러(`undefined: SortAsc`, `SortDesc`, 인자 개수 불일치)

- [ ] **Step 4: `internal/database/comments.go`의 `ListComments`를 교체**

파일 상단 `func scanComment` 바로 앞(`import` 블록 뒤)에 타입을 추가한다.

```go
// SortDirection은 최상위 댓글의 정렬 방향입니다
type SortDirection string

const (
	// SortAsc는 오래된 댓글부터 정렬합니다
	SortAsc SortDirection = "asc"
	// SortDesc는 최신 댓글부터 정렬합니다
	SortDesc SortDirection = "desc"
)

// topLevelOrderBy는 정렬 방향에 맞는 ORDER BY 절을 돌려줍니다.
// created_at만으로는 같은 트랜잭션에서 만든 댓글의 순서가 흔들리므로 id를 함께 씁니다 (ADR-004).
func topLevelOrderBy(direction SortDirection) string {
	if direction == SortDesc {
		return "ORDER BY created_at DESC, id DESC"
	}
	return "ORDER BY created_at ASC, id ASC"
}

// visibleTopLevelCondition은 공개 목록에 보이는 최상위 댓글 조건입니다.
// 삭제됐고 살아 있는 대댓글도 없는 댓글은 응답에서 제외되므로, 개수와 페이지 계산에서도 빼야 합니다.
const visibleTopLevelCondition = `
	AND (c.is_deleted = FALSE
		OR EXISTS (
			SELECT 1 FROM comments r
			WHERE r.post_id = c.post_id AND r.parent_id = c.id AND r.is_deleted = FALSE
		))
`
```

`ListComments` 전체를 아래로 바꾼다.

```go
// ListComments는 포스트의 최상위 댓글을 페이지 단위로 조회하고 각 댓글의 대댓글을 붙입니다.
// direction은 최상위 댓글에만 적용되고, 대댓글은 항상 오래된 순입니다.
func ListComments(ctx context.Context, db DBTX, postID int64, limit, offset int, direction SortDirection) ([]*models.Comment, int, error) {
	// 1단계: 보이는 최상위 댓글 총 개수 조회
	var total int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM comments c
		WHERE c.post_id = $1 AND c.parent_id IS NULL
	`+visibleTopLevelCondition, postID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments: %w", err)
	}

	// 2단계: 보이는 최상위 댓글 조회 (페이지네이션)
	query := `
		SELECT c.id, c.post_id, c.parent_id, c.author_name, c.author_password, c.content, c.ip_address, c.user_agent, c.is_deleted, c.created_at, c.updated_at, c.deleted_at
		FROM comments c
		WHERE c.post_id = $1 AND c.parent_id IS NULL
	` + visibleTopLevelCondition + topLevelOrderBy(direction) + `
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.ParentID,
			&comment.AuthorName,
			&comment.AuthorPassword,
			&comment.Content,
			&comment.IPAddress,
			&comment.UserAgent,
			&comment.IsDeleted,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&comment.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	// 3단계: 각 최상위 댓글의 대댓글 조회
	for _, comment := range comments {
		replies, err := getReplies(ctx, db, postID, comment.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get replies for comment %d: %w", comment.ID, err)
		}
		comment.Replies = replies
	}

	return comments, total, nil
}
```

- [ ] **Step 5: 호출부 수정 (핸들러 컴파일 오류 해소)**

`internal/handlers/comments.go:300`을 임시로 오래된 순 호출로 바꾼다. Task 2에서 파라미터를 붙인다.

```go
	comments, totalCount, err := database.ListComments(ctx, h.db, post.ID, limit, offset, database.SortAsc)
```

- [ ] **Step 6: 테스트 실행으로 통과 확인**

실행: `go test ./internal/database/ -run TestListComments -count=1 2>&1 | tail -3`
기대: `ok  	github.com/june20516/orbithall/internal/database`

- [ ] **Step 7: Commit**

```bash
git add internal/database/comments.go internal/database/comments_test.go internal/handlers/comments.go
git commit -F - <<'EOF'
feat: 댓글 목록 조회에 정렬 방향과 보이는 댓글 기준 추가

삭제됐고 살아 있는 대댓글도 없는 댓글을 개수와 페이지 계산에서 제외한다.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 2: 핸들러 — `sort`/`direction` 파라미터

**Files:**
- Modify: `internal/handlers/comments.go` (`ListComments`와 swagger 주석)
- Test: `internal/handlers/comments_test.go`

- [ ] **Step 1: 기존 테스트를 새 기본값(최신순)에 맞게 고치고 실패를 확인**

`internal/handlers/comments_test.go`에서 정렬을 전제하는 기존 테스트의 요청 URL에 `direction=asc`를 붙인다. 대상은 `TestListComments_Success_TreeStructure`, `TestListComments_Success_Pagination`, `TestListComments_DeletedComments`, `TestListComments_DeletedReplyUnderActiveParent`, `TestListComments_MixedRepliesUnderDeletedParent`다.

```go
	req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments?direction=asc", nil)
```
```go
	req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments?page=1&limit=2&direction=asc", nil)
```

(URL에 이미 쿼리 문자열이 있으면 `&direction=asc`를, 없으면 `?direction=asc`를 붙인다.)

- [ ] **Step 2: 새 동작 테스트를 `comments_test.go`의 ListComments 테스트들 뒤에 추가**

```go
func TestListComments_DefaultDirectionIsNewestFirst(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 최상위 댓글 2개
	apiKey := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "sort.test.com", []string{"http://localhost:3000"}, true).APIKey
	site, _ := database.GetSiteByAPIKey(ctx, tx, apiKey)
	post, _ := database.GetOrCreatePost(ctx, tx, site.ID, "test-post", "Test Post")

	database.CreateComment(ctx, tx, post.ID, nil, "Author1", "pass", "First", "10.0.0.1", "Agent")
	database.CreateComment(ctx, tx, post.ID, nil, "Author2", "pass", "Second", "10.0.0.2", "Agent")

	handler := NewCommentHandler(tx)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))
	rec := httptest.NewRecorder()

	// When: 정렬 파라미터 없이 조회
	handler.ListComments(rec, req)

	// Then: 최신 댓글이 먼저 오고 응답에 적용된 정렬이 담긴다
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Comments []struct {
			Content string `json:"content"`
		} `json:"comments"`
		Sort      string `json:"sort"`
		Direction string `json:"direction"`
	}
	json.NewDecoder(rec.Body).Decode(&response)

	if len(response.Comments) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(response.Comments))
	}
	if response.Comments[0].Content != "Second" {
		t.Errorf("Expected newest comment first, got %q", response.Comments[0].Content)
	}
	if response.Sort != "created_at" {
		t.Errorf("Expected sort created_at, got %q", response.Sort)
	}
	if response.Direction != "desc" {
		t.Errorf("Expected direction desc, got %q", response.Direction)
	}
}

func TestListComments_InvalidSortParameters(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	apiKey := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "invalid-sort.test.com", []string{"http://localhost:3000"}, true).APIKey
	site, _ := database.GetSiteByAPIKey(ctx, tx, apiKey)
	database.GetOrCreatePost(ctx, tx, site.ID, "test-post", "Test Post")

	handler := NewCommentHandler(tx)

	cases := []struct {
		name  string
		query string
	}{
		{name: "알 수 없는 정렬 기준", query: "?sort=author_name"},
		{name: "알 수 없는 정렬 방향", query: "?direction=descending"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments"+tc.query, nil)
			req.Header.Set("X-Orbithall-API-Key", apiKey)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("slug", "test-post")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(withSiteContext(req.Context(), site))
			rec := httptest.NewRecorder()

			handler.ListComments(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			json.NewDecoder(rec.Body).Decode(&response)

			if response.Error.Code != ErrInvalidInput {
				t.Errorf("Expected error code %s, got %s", ErrInvalidInput, response.Error.Code)
			}
		})
	}
}
```

- [ ] **Step 3: 테스트 실행으로 실패 확인**

실행: `go test ./internal/handlers/ -run 'TestListComments' -count=1 2>&1 | tail -10`
기대: `TestListComments_DefaultDirectionIsNewestFirst`가 `Expected newest comment first, got "First"`로 실패, `TestListComments_InvalidSortParameters`가 `Expected status 400, got 200`으로 실패

- [ ] **Step 4: 파싱 함수 추가**

`internal/handlers/comments.go`의 `hideDeletedRepliesAndMaskIP` 함수 바로 뒤에 추가한다.

```go
// 공개 댓글 목록에서 허용하는 정렬 기준
const commentSortCreatedAt = "created_at"

// parseCommentSort는 sort/direction 쿼리 파라미터를 검증해 돌려줍니다.
// 값이 없으면 기본값(created_at, desc)을 쓰고, 허용하지 않는 값이면 어떤 파라미터가 문제인지 알려줍니다.
func parseCommentSort(r *http.Request) (string, database.SortDirection, map[string]string) {
	sortField := r.URL.Query().Get("sort")
	if sortField == "" {
		sortField = commentSortCreatedAt
	}
	if sortField != commentSortCreatedAt {
		return "", "", map[string]string{"sort": "supported values: created_at"}
	}

	direction := database.SortDirection(r.URL.Query().Get("direction"))
	if direction == "" {
		direction = database.SortDesc
	}
	if direction != database.SortAsc && direction != database.SortDesc {
		return "", "", map[string]string{"direction": "supported values: asc, desc"}
	}

	return sortField, direction, nil
}
```

- [ ] **Step 5: 핸들러 본문에 반영**

`ListComments`의 "3. 쿼리 파라미터 파싱" 블록 끝(limit 검증 뒤)에 추가한다.

```go
	// 정렬 파라미터 파싱 (기본값: created_at, desc)
	sortField, direction, sortErr := parseCommentSort(r)
	if sortErr != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid sort parameter", sortErr)
		return
	}
```

포스트가 없을 때의 응답을 바꾼다.

```go
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"comments":  []models.Comment{},
			"sort":      sortField,
			"direction": string(direction),
			"pagination": map[string]interface{}{
				"current_page":   page,
				"total_pages":    0,
				"total_comments": 0,
				"per_page":       limit,
			},
		})
```

Task 1 Step 5에서 임시로 넣은 호출을 바꾼다.

```go
	comments, totalCount, err := database.ListComments(ctx, h.db, post.ID, limit, offset, direction)
```

마지막 응답을 바꾼다.

```go
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"comments":  comments,
		"sort":      sortField,
		"direction": string(direction),
		"pagination": map[string]interface{}{
			"current_page":   page,
			"total_pages":    totalPages,
			"total_comments": totalCount,
			"per_page":       limit,
		},
	})
```

- [ ] **Step 6: swagger 주석 수정**

`ListComments`의 주석에서 `@Param limit ...` 줄 뒤에 두 줄을 넣고, 400 응답 설명을 정렬까지 포함하도록 바꾼다.

```go
// @Param sort query string false "정렬 기준 (created_at)" default(created_at)
// @Param direction query string false "정렬 방향 (desc: 최신순, asc: 오래된 순)" Enums(asc, desc) default(desc)
```

- [ ] **Step 7: 테스트 실행으로 통과 확인**

실행: `go test ./internal/... -count=1 2>&1 | tail -6`
기대: 모든 패키지 `ok` 또는 `no test files`

- [ ] **Step 8: Commit**

```bash
git add internal/handlers/comments.go internal/handlers/comments_test.go
git commit -F - <<'EOF'
feat: 공개 댓글 목록에 sort/direction 파라미터 추가

기본값은 created_at, desc(최신순)이며 응답에 적용된 정렬을 담는다.
허용하지 않는 값은 400 INVALID_INPUT으로 거절한다.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 3: 위젯 — 페이지 합치기 순수 함수 (TDD)

**Files:**
- Modify: `widget/src/types.ts`
- Create: `widget/src/utils/commentPages.ts`
- Test: `widget/src/utils/commentPages.test.ts`
- Modify: `widget/package.json` (test 스크립트)

- [ ] **Step 1: 타입 추가**

`widget/src/types.ts`의 `CommentsResponse`를 아래로 바꾼다.

```ts
export type SortDirection = 'asc' | 'desc';

export interface CommentsPagination {
  currentPage: number;
  totalPages: number;
  totalComments: number;
  perPage: number;
}

export interface CommentsResponse {
  comments: Comment[];
  sort: string;
  direction: SortDirection;
  pagination: CommentsPagination;
}
```

- [ ] **Step 2: 실패하는 테스트 작성 — `widget/src/utils/commentPages.test.ts`**

```ts
import { test, expect } from "bun:test";
import {
  hasNextPage,
  mergeCommentPages,
  reloadPages,
  remainingCommentCount,
} from "./commentPages";
import type { Comment, CommentsPagination, CommentsResponse } from "../types";

function comment(id: number): Comment {
  return {
    id,
    postId: 1,
    parentId: null,
    authorName: `작성자 ${id}`,
    content: `댓글 ${id}`,
    isDeleted: false,
    createdAt: "2026-09-23T00:00:00Z",
    updatedAt: "2026-09-23T00:00:00Z",
    replies: [],
  } as unknown as Comment;
}

function pagination(currentPage: number, totalPages: number, totalComments: number): CommentsPagination {
  return { currentPage, totalPages, totalComments, perPage: 50 };
}

function response(comments: Comment[], p: CommentsPagination): CommentsResponse {
  return { comments, sort: "created_at", direction: "desc", pagination: p };
}

test("mergeCommentPages는 새 댓글만 뒤에 이어 붙인다", () => {
  const merged = mergeCommentPages([comment(3), comment(2)], [comment(1)]);

  expect(merged.map((c) => c.id)).toEqual([3, 2, 1]);
});

test("mergeCommentPages는 이미 있는 id를 거른다", () => {
  const merged = mergeCommentPages([comment(3), comment(2)], [comment(2), comment(1)]);

  expect(merged.map((c) => c.id)).toEqual([3, 2, 1]);
});

test("remainingCommentCount는 남은 최상위 댓글 수를 돌려준다", () => {
  expect(remainingCommentCount(pagination(1, 2, 58), 50)).toBe(8);
});

test("remainingCommentCount는 음수를 돌려주지 않는다", () => {
  expect(remainingCommentCount(pagination(2, 2, 58), 60)).toBe(0);
});

test("hasNextPage는 마지막 페이지에서 false다", () => {
  expect(hasNextPage(pagination(1, 2, 58))).toBe(true);
  expect(hasNextPage(pagination(2, 2, 58))).toBe(false);
});

test("reloadPages는 펼친 페이지를 순서대로 다시 불러와 합친다", async () => {
  const requested: number[] = [];
  const pages: Record<number, CommentsResponse> = {
    1: response([comment(5), comment(4)], pagination(1, 3, 6)),
    2: response([comment(3), comment(2)], pagination(2, 3, 6)),
  };

  const result = await reloadPages(async (page) => {
    requested.push(page);
    return pages[page];
  }, 2);

  expect(requested).toEqual([1, 2]);
  expect(result.comments.map((c) => c.id)).toEqual([5, 4, 3, 2]);
  expect(result.pagination.currentPage).toBe(2);
});

test("reloadPages는 총 페이지 수를 넘겨 요청하지 않는다", async () => {
  const requested: number[] = [];

  const result = await reloadPages(async (page) => {
    requested.push(page);
    return response([comment(1)], pagination(1, 1, 1));
  }, 3);

  expect(requested).toEqual([1]);
  expect(result.comments.map((c) => c.id)).toEqual([1]);
});
```

- [ ] **Step 3: 테스트를 실행해 실패 확인**

실행: `cd widget && bun test src/utils/commentPages.test.ts 2>&1 | tail -5`
기대: `Cannot find module './commentPages'` 로 실패

- [ ] **Step 4: `widget/src/utils/commentPages.ts` 작성**

```ts
import type { Comment, CommentsPagination, CommentsResponse } from "../types";

/**
 * 이미 불러온 목록 뒤에 새 페이지를 이어 붙입니다.
 * 불러오는 사이에 댓글이 늘면 페이지 경계가 밀려 같은 댓글이 다시 올 수 있으므로 id로 거릅니다.
 */
export function mergeCommentPages(
  existing: Comment[],
  incoming: Comment[]
): Comment[] {
  const seen = new Set(existing.map((comment) => comment.id));
  const added = incoming.filter((comment) => {
    if (seen.has(comment.id)) {
      return false;
    }
    seen.add(comment.id);
    return true;
  });

  return added.length === 0 ? existing : [...existing, ...added];
}

/** 더 보기 버튼에 보여 줄 남은 최상위 댓글 수입니다 */
export function remainingCommentCount(
  pagination: CommentsPagination,
  loadedCount: number
): number {
  return Math.max(pagination.totalComments - loadedCount, 0);
}

/** 다음 페이지가 남아 있는지 확인합니다 */
export function hasNextPage(pagination: CommentsPagination): boolean {
  return pagination.currentPage < pagination.totalPages;
}

/**
 * 지금까지 펼친 1..lastPage 페이지를 다시 불러와 합칩니다.
 * 수정이나 삭제 뒤에도 사용자가 펼쳐 둔 범위를 유지하기 위해 씁니다.
 */
export async function reloadPages(
  fetchPage: (page: number) => Promise<CommentsResponse>,
  lastPage: number
): Promise<{ comments: Comment[]; pagination: CommentsPagination }> {
  const first = await fetchPage(1);
  let comments = first.comments || [];
  let pagination = first.pagination;

  for (let page = 2; page <= lastPage && page <= pagination.totalPages; page++) {
    const next = await fetchPage(page);
    comments = mergeCommentPages(comments, next.comments || []);
    pagination = next.pagination;
  }

  return { comments, pagination };
}
```

- [ ] **Step 5: 테스트를 실행해 통과 확인**

실행: `cd widget && bun test src/utils/commentPages.test.ts 2>&1 | tail -5`
기대: `7 pass`, `0 fail`

- [ ] **Step 6: package.json에 test 스크립트 추가**

`widget/package.json`의 `scripts`에서 `"build"` 줄 뒤에 추가한다.

```json
    "test": "bun test",
```

- [ ] **Step 7: Commit**

```bash
git add widget/src/types.ts widget/src/utils/commentPages.ts widget/src/utils/commentPages.test.ts widget/package.json
git commit -F - <<'EOF'
feat: 위젯 댓글 페이지 합치기 유틸 추가

id로 중복을 거르는 합치기, 남은 개수, 펼친 범위 다시 불러오기를
순수 함수로 분리하고 테스트를 붙였다.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 4: 위젯 — API 클라이언트와 문구

**Files:**
- Modify: `widget/src/api/client.ts`
- Modify: `widget/src/i18n/locales.ts`

- [ ] **Step 1: `getComments`에 정렬 방향 추가**

`widget/src/api/client.ts`의 `getComments`를 바꾼다.

```ts
  async getComments(
    postSlug: string,
    page = 1,
    limit = 50,
    direction: SortDirection = "desc"
  ): Promise<CommentsResponse> {
    return this.request<CommentsResponse>(
      `/posts/${postSlug}/comments?page=${page}&limit=${limit}&sort=created_at&direction=${direction}`
    );
  }
```

같은 파일 첫 줄의 import에 타입을 추가한다.

```ts
import type {
  Comment,
  CommentSubmitData,
  CommentsResponse,
  SortDirection,
} from "../types";
```

- [ ] **Step 2: 문구 추가**

`widget/src/i18n/locales.ts`의 ko 객체에서 `'comments.title': '댓글',` 줄 뒤에 추가한다.

```ts
    'comments.loadMore': '댓글 {count}개 더 보기',
    'comments.loadingMore': '불러오는 중...',
    'comments.loadMoreError': '댓글을 더 불러오지 못했습니다. 다시 시도해주세요.',
```

en 객체에서 `'comments.title': 'Comments',` 줄 뒤에 추가한다.

```ts
    'comments.loadMore': 'Show {count} more comments',
    'comments.loadingMore': 'Loading...',
    'comments.loadMoreError': 'Failed to load more comments. Please try again.',
```

- [ ] **Step 3: 타입 확인**

실행: `cd widget && bunx tsc --noEmit 2>&1 | head -5`
기대: 출력 없음 (CommentWidget은 Task 5에서 고치므로, 이 단계에서 `getComments` 관련 오류가 없으면 된다)

- [ ] **Step 4: Commit**

```bash
git add widget/src/api/client.ts widget/src/i18n/locales.ts
git commit -F - <<'EOF'
feat: 위젯 댓글 조회에 정렬 방향 전달과 더 보기 문구 추가

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 5: 위젯 — 더 보기 동작과 CSS

**Files:**
- Modify: `widget/src/components/CommentWidget.tsx`
- Modify: `widget/src/styles.css`

- [ ] **Step 1: `CommentWidget.tsx` 전체 교체**

```tsx
import { useState, useEffect, useMemo } from "preact/hooks";
import { CommentList } from "./CommentList";
import { CommentForm } from "./CommentForm";
import { Button } from "./Button";
import type { Comment, CommentSubmitData, CommentsPagination } from "../types";
import { OrbitHallAPIClient } from "../api/client";
import {
  hasNextPage,
  mergeCommentPages,
  reloadPages,
  remainingCommentCount,
} from "../utils/commentPages";
import { getErrorMessage, type ErrorResponse } from "../utils/errorMessages";
import { useI18n } from "../i18n/context";

interface CommentWidgetProps {
  apiUrl: string;
  apiKey: string;
  postSlug: string;
}

// 한 번에 불러오는 최상위 댓글 수
const PAGE_SIZE = 50;

export function CommentWidget({
  apiUrl,
  apiKey,
  postSlug,
}: CommentWidgetProps) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [pagination, setPagination] = useState<CommentsPagination | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loadMoreError, setLoadMoreError] = useState<string | null>(null);
  const { t } = useI18n();

  // API 클라이언트 생성
  const apiClient = useMemo(
    () => new OrbitHallAPIClient(apiUrl, apiKey),
    [apiUrl, apiKey]
  );

  // 최신순으로 한 페이지 조회
  const fetchPage = (page: number) =>
    apiClient.getComments(postSlug, page, PAGE_SIZE, "desc");

  const toErrorMessage = (err: unknown): string => {
    if (err instanceof Error && (err as any).response) {
      return getErrorMessage((err as any).response as ErrorResponse, t);
    }
    return t("error.NETWORK_ERROR");
  };

  // 첫 페이지부터 다시 불러오기 (최초 로드, 최상위 댓글 작성 후)
  const loadFirstPage = async () => {
    try {
      setLoading(true);
      const data = await fetchPage(1);
      setComments(data.comments || []);
      setPagination(data.pagination);
      setError(null);
      setLoadMoreError(null);
    } catch (err) {
      console.error("OrbitHall: Failed to load comments", err);
      setError(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  // 펼쳐 둔 범위를 유지한 채 다시 불러오기 (답글 작성, 수정, 삭제 후)
  const reloadOpenedPages = async () => {
    const lastPage = pagination?.currentPage ?? 1;
    try {
      const data = await reloadPages(fetchPage, lastPage);
      setComments(data.comments);
      setPagination(data.pagination);
      setError(null);
      setLoadMoreError(null);
    } catch (err) {
      console.error("OrbitHall: Failed to reload comments", err);
      setError(toErrorMessage(err));
    }
  };

  // 다음 페이지를 이어 붙이기
  const handleLoadMore = async () => {
    if (!pagination || loadingMore) {
      return;
    }

    try {
      setLoadingMore(true);
      setLoadMoreError(null);
      const data = await fetchPage(pagination.currentPage + 1);
      setComments((current) => mergeCommentPages(current, data.comments || []));
      setPagination(data.pagination);
    } catch (err) {
      console.error("OrbitHall: Failed to load more comments", err);
      setLoadMoreError(t("comments.loadMoreError"));
    } finally {
      setLoadingMore(false);
    }
  };

  // 컴포넌트 마운트 시, postSlug 변경 시 첫 페이지 조회
  useEffect(() => {
    setComments([]);
    setPagination(null);
    loadFirstPage();
  }, [postSlug]);

  // 댓글/답글 작성 핸들러
  const handleCommentSubmit = async (commentData: CommentSubmitData) => {
    try {
      await apiClient.createComment(postSlug, commentData);
      // 최상위 댓글은 맨 위에 오도록 첫 페이지부터, 답글은 펼친 범위를 유지한 채 다시 불러온다
      if (commentData.parentId) {
        await reloadOpenedPages();
      } else {
        await loadFirstPage();
      }
    } catch (err) {
      console.error("OrbitHall: Failed to submit comment", err);
      throw err;
    }
  };

  // 댓글 수정 핸들러
  const handleCommentUpdate = async (
    commentId: number,
    content: string,
    password: string
  ) => {
    try {
      await apiClient.updateComment(commentId, content, password);
      await reloadOpenedPages();
    } catch (err) {
      console.error("OrbitHall: Failed to update comment", err);
      throw err;
    }
  };

  // 댓글 삭제 핸들러
  const handleCommentDelete = async (commentId: number, password: string) => {
    try {
      await apiClient.deleteComment(commentId, password);
      await reloadOpenedPages();
    } catch (err) {
      console.error("OrbitHall: Failed to delete comment", err);
      throw err;
    }
  };

  const showLoadMore = !loading && !error && pagination !== null && hasNextPage(pagination);
  const remaining = pagination ? remainingCommentCount(pagination, comments.length) : 0;

  return (
    <div className="orb-widget">
      <div className="orb-header">
        <h3>{t("comments.title")}</h3>
      </div>

      <CommentForm onSubmit={handleCommentSubmit} />

      {loading && <div className="orb-loading">{t("loading")}</div>}

      {error && <div className="orb-error">{error}</div>}

      {!loading && !error && comments.length === 0 && (
        <div className="orb-empty">{t("empty")}</div>
      )}

      {!loading && !error && comments.length > 0 && (
        <CommentList
          comments={comments}
          onReply={handleCommentSubmit}
          onUpdate={handleCommentUpdate}
          onDelete={handleCommentDelete}
        />
      )}

      {showLoadMore && (
        <div className="orb-load-more">
          <Button
            label={
              loadingMore
                ? t("comments.loadingMore")
                : t("comments.loadMore").replace("{count}", String(remaining))
            }
            onClick={handleLoadMore}
            variant="clear"
            type="secondary"
            disabled={loadingMore}
          />
          {loadMoreError && (
            <div className="orb-load-more-error">{loadMoreError}</div>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: CSS 수정 — 삭제된 댓글 선택자**

`widget/src/styles.css`의 삭제된 댓글 블록을 바꾼다.

old:
```css
/* 삭제된 댓글 */
.orb-comment-deleted {
  opacity: 0.6;
}

.orb-comment-deleted .orb-comment-content {
  font-style: italic;
  color: var(--orb-text-secondary);
}
```
new:
```css
/* 삭제된 댓글 */
/* 흐림과 기울임꼴은 삭제된 댓글 자리에만 적용한다.
   자손 선택자를 쓰면 살아 있는 답글까지 삭제된 것처럼 보인다. */
.orb-comment-deleted > .orb-comment-content {
  opacity: 0.6;
  font-style: italic;
  color: var(--orb-text-secondary);
}
```

- [ ] **Step 3: CSS 추가 — 더 보기 영역**

`widget/src/styles.css`의 `.orb-comment-list` 블록 뒤에 추가한다.

```css
/* 더 보기 */
.orb-load-more {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--orb-spacing-xs);
  margin-top: var(--orb-spacing-md);
}

.orb-load-more-error {
  font-size: var(--orb-font-size-sm);
  color: var(--orb-error-color);
  text-align: center;
}
```

- [ ] **Step 4: 타입과 빌드 확인**

실행: `cd widget && bunx tsc --noEmit && bun test && bun run build 2>&1 | tail -4`
기대: 타입 오류 없음, `7 pass`, 빌드 성공

- [ ] **Step 5: Commit**

```bash
git add widget/src/components/CommentWidget.tsx widget/src/styles.css
git commit -F - <<'EOF'
feat: 위젯에 댓글 더 보기 추가와 삭제 댓글 스타일 수정

최신순 목록에 더 보기 버튼을 붙이고, 답글 작성·수정·삭제 후에는
펼친 범위를 유지한 채 다시 불러온다. 삭제된 댓글의 흐림 효과가
살아 있는 답글까지 번지던 선택자를 자식 선택자로 좁혔다.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 6: 로컬 브라우저 확인

로컬 API(8080)와 DB는 이미 떠 있다. 아직 시드 데이터(Task 9)는 없으므로, 여기서는 현재 `orbithall-widget-demo`의 댓글 5개로 화면을 확인한다.

- [ ] **Step 1: 로컬 API가 새 코드로 떠 있는지 확인**

실행:
```bash
curl -s "http://localhost:8080/api/posts/orbithall-widget-demo/comments" -H "X-Orbithall-API-Key: orb_test_local_dev_key_12345" | head -c 200
curl -s -o /dev/null -w "invalid direction -> %{http_code}\n" "http://localhost:8080/api/posts/orbithall-widget-demo/comments?direction=descending" -H "X-Orbithall-API-Key: orb_test_local_dev_key_12345"
```
기대: 첫 응답에 `"sort":"created_at"`과 `"direction":"desc"`, 두 번째는 `400`
(컨테이너가 air로 재시작 중이면 몇 초 뒤 다시 실행한다)

- [ ] **Step 2: 로컬 API용 위젯 빌드와 페이지 준비**

```bash
cd widget && bun run build && cd ..
D=/private/tmp/claude-501/-Users-bran-personal-orbithall/ce5fe0f8-7c5c-4818-8f17-4378f094d655/scratchpad/widget-capture
cp static/embed.js static/embed.css "$D"/
```
(4173 서버는 이미 떠 있다. 새로 띄울 필요가 없다면 그대로 쓴다.)

- [ ] **Step 3: 브라우저로 확인**

Playwright로 `http://localhost:4173/`을 열어 확인한다.
- 목록이 최신순인지 (서연 → 삭제된 댓글 → 민지)
- 삭제된 댓글 밑의 답글이 흐리지 않고 정상으로 보이는지
- 댓글이 5개뿐이라 더 보기 버튼은 보이지 않아야 한다

콘솔 오류가 없어야 한다(favicon 404는 무시). 스크린샷은 `.playwright-mcp/`에 저장되므로 확인 후 `rm -rf .playwright-mcp`로 지운다.

---

### Task 7: 문서 갱신

**Files:**
- Create: `docs/adr/008-comment-list-sort-parameters.md`
- Modify: `docs/adr/004-comment-sorting-strategy.md`, `docs/adr/README.md`, `README.md`, `widget/README.md`, `docs/specs/widget-integration-guide.md`
- Move: `docs/tasks/pending/008-widget-pagination.md` → `docs/tasks/completed/`

- [ ] **Step 1: ADR-008 작성**

````markdown
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
                  WHERE r.post_id = c.post_id AND r.parent_id = c.id AND r.is_deleted = FALSE))
```

핸들러의 기존 필터는 삭제된 댓글의 작성자·내용을 비우고 IP를 마스킹하는 일을 계속 맡는다.

## Consequences
### Positive
- 위젯이 최신 댓글부터 보여 주고, "댓글 더 보기"로 이어서 불러올 수 있다.
- `total_comments`가 화면에 보이는 댓글 수와 맞아 "남은 개수"를 믿을 수 있다.
- 페이지가 `per_page`만큼 채워진다.

### Negative
- 기본 정렬이 바뀌므로, 오래된 순을 원하는 클라이언트는 `direction=asc`를 명시해야 한다.
- 목록 쿼리에 `EXISTS` 서브쿼리가 붙는다. `comments(parent_id)` 인덱스를 타고, 한 포스트의 댓글 수가 수천 단위인 블로그 규모에서는 영향이 작다.

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
````

- [ ] **Step 2: ADR-004에 한 줄 추가**

`## Status`의 `Accepted` 아래에 넣는다.

```markdown
Accepted

> 정렬 방향 선택은 [ADR-008](008-comment-list-sort-parameters.md)에서 확장되었다. `created_at`과 `id`를 함께 쓰는 원칙은 그대로다.
```

- [ ] **Step 3: ADR README 목록에 추가**

`| [007](007-widget-semver-tag-release.md) | Widget semver 태그 릴리스 | Accepted | 2026-09-22 |` 줄 뒤에 추가한다.

```markdown
| [008](008-comment-list-sort-parameters.md) | 공개 댓글 목록 정렬 파라미터 | Accepted | 2026-09-23 |
```

- [ ] **Step 4: README.md의 목록 API 설명 갱신**

`GET /api/posts/:slug/comments` 아래 설명과 응답 예시를 아래로 바꾼다(116행 부근).

````markdown
GET /api/posts/:slug/comments
```

**쿼리 파라미터**

| 이름 | 값 | 기본값 | 설명 |
|------|-----|--------|------|
| `page` | 1 이상 | 1 | 페이지 번호 |
| `limit` | 1~100 | 50 | 페이지당 최상위 댓글 수 |
| `sort` | `created_at` | `created_at` | 정렬 기준 |
| `direction` | `desc`, `asc` | `desc` | `desc`는 최신순, `asc`는 오래된 순 |

`sort`나 `direction`에 허용하지 않는 값을 주면 400 `INVALID_INPUT`을 반환합니다. 대댓글은 방향과 무관하게 항상 오래된 순입니다.

**응답**

```json
{
  "comments": [...],
  "sort": "created_at",
  "direction": "desc",
  "pagination": {
    "current_page": 1,
    "total_pages": 2,
    "total_comments": 58,
    "per_page": 50
  }
}
````

(기존 응답 예시 블록을 위 내용으로 대체한다. `total_comments`는 화면에 보이는 최상위 댓글 수다.)

- [ ] **Step 5: 설치 주소를 1.2.0으로**

실행:
```bash
sed -i '' 's#orbithall@1\.1\.1/#orbithall@1.2.0/#g' README.md widget/README.md docs/specs/widget-integration-guide.md
grep -c "orbithall@1.2.0/" README.md widget/README.md docs/specs/widget-integration-guide.md
```
기대: `README.md:2`, `widget/README.md:5`, `docs/specs/widget-integration-guide.md:4`

- [ ] **Step 6: widget/README.md에 목록 동작과 업그레이드 안내 추가**

`#### CDN URL` 블록 뒤, `#### 배포 안전장치` 앞에 추가한다.

````markdown
#### 1.1.1에서 1.2.0으로 올릴 때

설치 주소의 버전만 바꾸면 됩니다. 설정은 그대로입니다.

- 댓글이 **최신순**으로 보입니다(1.1.1은 오래된 순).
- 최상위 댓글을 50개씩 나눠 보여 주고, 남은 댓글이 있으면 목록 아래에 **"댓글 N개 더 보기"** 버튼이 보입니다. 1.1.1은 50개까지만 보여 주고 나머지는 볼 방법이 없었습니다.
- 대댓글은 지금처럼 각 댓글 아래에 오래된 순으로 모두 보입니다.
- 댓글을 쓰면 목록 맨 위에 자기 댓글이 보입니다. 답글·수정·삭제 뒤에는 펼쳐 둔 범위가 그대로 유지됩니다.
````

- [ ] **Step 7: 통합 가이드에 목록 동작 추가**

`docs/specs/widget-integration-guide.md`의 `### 배포 전략` 앞(번들 출력 설명 뒤)에 절을 추가한다.

````markdown
### 목록 동작
- 최상위 댓글을 최신순으로 50개씩 조회합니다(`sort=created_at&direction=desc`).
- 남은 페이지가 있으면 목록 아래 "댓글 더 보기" 버튼으로 이어서 불러옵니다. 이미 불러온 댓글은 id로 걸러 중복되지 않습니다.
- 대댓글은 각 댓글 아래에 오래된 순으로 모두 표시됩니다.
- 댓글 작성 후에는 첫 페이지를 다시 불러와 작성한 댓글이 맨 위에 보입니다. 답글·수정·삭제 후에는 펼쳐 둔 범위를 유지한 채 다시 불러옵니다.
````

- [ ] **Step 8: 작업 문서 008 채우고 completed로 이동**

실행: `git mv docs/tasks/pending/008-widget-pagination.md docs/tasks/completed/008-widget-pagination.md`

파일 전체를 아래로 바꾼다(제목의 `[WIP]` 제거).

```markdown
# 위젯 댓글 페이지네이션

## 작성일

2025-10-24

## 우선순위

- [ ] 긴급
- [ ] 높음
- [x] 보통
- [ ] 낮음

## 작업 개요

위젯이 최상위 댓글 50개만 한 번 조회해서, 댓글이 50개를 넘으면 오래된 댓글만 보이고 새 댓글은 화면에 나오지 않았다. 목록을 최신순으로 바꾸고 "댓글 더 보기"를 붙였다.

## 작업 범위

### 포함
- 공개 목록 API에 `sort`/`direction` 파라미터 추가, 기본값을 최신순으로 변경 (ADR-008)
- 보이는 댓글 기준으로 개수·페이지 계산
- 위젯 1.2.0: 최신순 목록, "댓글 더 보기", 작성·수정·삭제 후 동작 정리
- 삭제된 댓글의 흐림 효과가 답글까지 번지던 CSS 수정

### 제외
- 대댓글 페이지네이션 (작업 007)
- 어드민 API와 화면
- 커서 기반 페이지네이션

## 주요 결정사항

- 정렬 파라미터는 `sort=created_at&direction=desc|asc`로 했다. 자체 값(`order=newest`)보다 널리 쓰이는 형식이다.
- 기본값을 최신순으로 두어, 파라미터를 모르는 1.1.1 위젯도 배포 즉시 최신순으로 보인다.
- 허용하지 않는 정렬 값은 400으로 거절한다. 조용히 기본값으로 처리하면 순서가 뒤집혀도 알기 어렵다.
- 페이지는 offset 방식에 id 중복 제거를 더했다. 불러오는 사이 다른 사람이 댓글을 지우면 한 개를 놓칠 수 있지만, 블로그 규모에서는 드물어 수용했다.

## 작업 이력

### [2026-09-23] 구현과 배포
- 백엔드: `sort`/`direction` 파라미터, 보이는 댓글 기준 개수·페이지, swagger 주석
- 위젯 1.2.0: 최신순 목록, "댓글 N개 더 보기", 답글·수정·삭제 후 펼친 범위 유지, 삭제 댓글 CSS 수정
- 문서: ADR-008, README API 설명, widget/README 업그레이드 안내, 통합 가이드

### [2026-09-23] 작업 완료
```

- [ ] **Step 9: Commit**

```bash
git add docs/adr/008-comment-list-sort-parameters.md docs/adr/004-comment-sorting-strategy.md docs/adr/README.md README.md widget/README.md docs/specs/widget-integration-guide.md docs/tasks/completed/008-widget-pagination.md
git commit -F - <<'EOF'
docs: 댓글 정렬 파라미터 ADR-008과 위젯 1.2.0 문서 갱신

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 8: 배포 (⚠️ 태그 push 전 사용자 확인)

- [ ] **Step 1: 위젯 버전 올리기**

`widget/package.json`의 `"version": "1.1.1"`을 `"version": "1.2.0"`으로 바꾼다.

```bash
git add widget/package.json
git commit -F - <<'EOF'
chore: 위젯 버전 1.2.0

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

- [ ] **Step 2: 전체 검증**

실행:
```bash
go build ./... && go vet ./... && go test ./internal/... -count=1 2>&1 | tail -6
cd widget && bunx tsc --noEmit && bun test 2>&1 | tail -3 && cd ..
git status --porcelain
```
기대: Go 테스트 전부 `ok`, 위젯 테스트 `7 pass`, 작업 트리 깨끗

- [ ] **Step 3: PR 생성 (feature → develop)**

```bash
git status -sb | head -1     # upstream 없음 확인
git push -u origin feat/comment-pagination
gh pr create --repo june20516/orbithall --base develop --head feat/comment-pagination \
  --title "feat: 댓글 목록 최신순 정렬과 더 보기" --body-file <본문 파일>
```

본문에 넣을 것: 배경(50개 이후가 안 보임), 백엔드 변경(`sort`/`direction`, 기본 최신순, 보이는 댓글 기준), 위젯 1.2.0 동작, 삭제 댓글 CSS 수정, 검증 결과(Go 테스트, 위젯 테스트, 로컬 브라우저), 배포 순서(백엔드 먼저, 태그는 확인 후), 끝에 다음 두 줄.

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
```

- [ ] **Step 4: develop 머지 → main PR → CI 확인 → 머지**

사용자 확인을 받은 뒤 진행한다. main PR에서는 `Validate Production Build`(Docker 빌드)가 돈다.

- [ ] **Step 5: 운영 확인**

```bash
curl -s -o /dev/null -w "health %{http_code}\n" https://orbithall.onrender.com/health
# 블로그 페이지 소스에 공개된 위젯 API 키를 그대로 쓴다
KEY=$(curl -sL https://june20516.github.io/ | grep -o 'apiKey:"[^"]*"' | head -1 | cut -d'"' -f2)
curl -s "https://orbithall.onrender.com/api/posts/orbithall-introduce/comments" -H "X-Orbithall-API-Key: $KEY" | head -c 200
curl -s -o /dev/null -w "invalid direction -> %{http_code}\n" "https://orbithall.onrender.com/api/posts/orbithall-introduce/comments?direction=descending" -H "X-Orbithall-API-Key: $KEY"
```
기대: health 200, 응답에 `"direction":"desc"`, 잘못된 값은 400
(API 키는 블로그 페이지 소스에 공개된 값이다. Origin 헤더 없이 호출하면 CORS 검사를 타지 않는다.)

- [ ] **Step 6: ⚠️ 사용자 확인 후 태그 발행**

확인받을 내용: `bun run publish`로 `v1.2.0` 태그를 GitHub에 push한다(되돌리기 어려움).

```bash
cd widget && bun run publish
```
기대: `배포 버전: 1.2.0`, 빌드, `* [new tag] v1.2.0`, CDN 주소 출력, 원래 브랜치 복귀

- [ ] **Step 7: CDN 확인**

```bash
for v in 1.2.0 1.2 1 latest; do for f in embed.js embed.css; do
  printf "%-7s %-10s " "$v" "$f"
  curl -s -o /dev/null -D - "https://cdn.jsdelivr.net/gh/june20516/orbithall@$v/static/$f" \
    | grep -iE '^(HTTP|x-jsd-version:|x-jsd-version-type|cache-control)' | tr -d '\r' | tr '\n' ' '
  echo
done; done
```
기대: `1.2.0`은 200 + `version` + `immutable`, 나머지는 200 + `version` + `x-jsd-version: 1.2.0`

---

### Task 9: 캡처용 로컬 환경

**저장소 파일은 바꾸지 않는다. 시드 스크립트는 scratchpad에만 둔다. 운영 DB에는 쓰지 않는다.**

- [ ] **Step 1: 시드 스크립트 작성**

`/private/tmp/claude-501/-Users-bran-personal-orbithall/ce5fe0f8-7c5c-4818-8f17-4378f094d655/scratchpad/seed-demo.sql`에 작성한다. 로컬 테스트 사이트(id 21)의 `orbithall-widget-demo` 포스트만 건드린다.

```sql
-- 캡처용 시드 (로컬 DB 전용)
BEGIN;

-- 대상 포스트의 기존 댓글 제거
DELETE FROM comments
WHERE post_id = (SELECT id FROM posts WHERE site_id = 21 AND slug = 'orbithall-widget-demo');

-- 보이는 최상위 댓글 58개 (최신이 위로 오도록 시간 간격을 둔다)
INSERT INTO comments (post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at)
SELECT
  (SELECT id FROM posts WHERE site_id = 21 AND slug = 'orbithall-widget-demo'),
  NULL,
  (ARRAY['민지','지훈','서연','도윤','하늘','준호','예린','태현','수아','현우'])[1 + (i % 10)] || ' ' || i,
  '$2a$10$abcdefghijklmnopqrstuv',
  (ARRAY[
    '댓글창이 깔끔해서 읽기 편하네요.',
    '로그인 없이 남길 수 있는 게 좋습니다.',
    '시리즈 다음 편도 기대할게요.',
    '직접 만드셨다니 대단합니다.',
    '위젯 붙이는 방법이 궁금했는데 잘 봤어요.',
    '저도 블로그에 붙여 보고 싶네요.',
    '설명이 자세해서 따라 하기 좋았습니다.',
    '답글도 되는군요. 잘 쓰겠습니다.'
  ])[1 + (i % 8)],
  '192.168.0.' || (1 + (i % 200)),
  'Mozilla/5.0',
  FALSE,
  NOW() - (i || ' hours')::interval,
  NOW() - (i || ' hours')::interval
FROM generate_series(1, 58) AS i;

COMMIT;
```

이어서 답글, 삭제된 댓글, 수정된 댓글을 넣는다(같은 파일 아래에 이어서 작성).

```sql
BEGIN;

-- 최근 댓글 세 개에 답글 1~3개
INSERT INTO comments (post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at)
SELECT p.post_id, p.id, '작성자', '$2a$10$abcdefghijklmnopqrstuv', r.content, '192.168.0.9', 'Mozilla/5.0', FALSE,
       p.created_at + interval '30 minutes' * r.n, p.created_at + interval '30 minutes' * r.n
FROM (
  SELECT id, post_id, created_at, row_number() OVER (ORDER BY created_at DESC) AS rn
  FROM comments
  WHERE post_id = (SELECT id FROM posts WHERE site_id = 21 AND slug = 'orbithall-widget-demo')
    AND parent_id IS NULL
) p
JOIN (VALUES
  (1, 1, '읽어 주셔서 감사합니다.'),
  (1, 2, '궁금한 점 있으면 또 남겨 주세요.'),
  (2, 1, '저도 같은 생각이에요.'),
  (3, 1, '반가워요!'),
  (3, 2, '다음 편에서 더 자세히 다룰게요.'),
  (3, 3, '의견 고맙습니다.')
) AS r(rn, n, content) ON r.rn = p.rn;

-- 답글이 달린 채 삭제된 댓글 (네 번째 최신 댓글)
WITH target AS (
  SELECT id, post_id, created_at
  FROM comments
  WHERE post_id = (SELECT id FROM posts WHERE site_id = 21 AND slug = 'orbithall-widget-demo')
    AND parent_id IS NULL
  ORDER BY created_at DESC
  OFFSET 3 LIMIT 1
)
INSERT INTO comments (post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted, created_at, updated_at)
SELECT post_id, id, '지나가던 독자', '$2a$10$abcdefghijklmnopqrstuv', '아래 답글만 남아 있는 경우입니다.', '192.168.0.50', 'Mozilla/5.0', FALSE,
       created_at + interval '1 hour', created_at + interval '1 hour'
FROM target;

UPDATE comments SET is_deleted = TRUE, deleted_at = NOW() - interval '2 hours'
WHERE id = (
  SELECT id FROM comments
  WHERE post_id = (SELECT id FROM posts WHERE site_id = 21 AND slug = 'orbithall-widget-demo')
    AND parent_id IS NULL
  ORDER BY created_at DESC
  OFFSET 3 LIMIT 1
);

-- 수정된 댓글 (두 번째 최신 댓글)
UPDATE comments SET content = '오타가 있어서 고쳤습니다. 잘 읽었어요!', updated_at = created_at + interval '10 minutes'
WHERE id = (
  SELECT id FROM comments
  WHERE post_id = (SELECT id FROM posts WHERE site_id = 21 AND slug = 'orbithall-widget-demo')
    AND parent_id IS NULL
  ORDER BY created_at DESC
  OFFSET 1 LIMIT 1
);

COMMIT;
```

- [ ] **Step 2: 시드 실행과 확인**

```bash
S=/private/tmp/claude-501/-Users-bran-personal-orbithall/ce5fe0f8-7c5c-4818-8f17-4378f094d655/scratchpad
docker exec -i orbithall-db psql -U orbithall -d orbithall_db -v ON_ERROR_STOP=1 < "$S/seed-demo.sql"
curl -s "http://localhost:8080/api/posts/orbithall-widget-demo/comments?page=1&limit=50" \
  -H "X-Orbithall-API-Key: orb_test_local_dev_key_12345" \
  | bun -e 'const j=JSON.parse(await Bun.stdin.text()); console.log("보이는 최상위:", j.pagination.total_comments, "페이지:", j.pagination.total_pages, "정렬:", j.sort, j.direction); console.log("첫 댓글:", j.comments[0].author_name, "| 삭제 표시:", j.comments.filter(c=>c.is_deleted).length, "| 수정된 댓글:", j.comments.filter(c=>c.updated_at!==c.created_at).length);'
```
기대: 보이는 최상위 58개(삭제한 댓글은 답글이 있어 자리로 남는다), 페이지 2, 정렬 `created_at desc`, 첫 페이지에 삭제 표시 1개와 수정된 댓글 1개, 남은 개수 8개

- [ ] **Step 3: 5500 포트로 시험 페이지 띄우기**

```bash
S=/private/tmp/claude-501/-Users-bran-personal-orbithall/ce5fe0f8-7c5c-4818-8f17-4378f094d655/scratchpad
mkdir -p "$S/widget-demo-5500"
cp /Users/bran/personal/orbithall/static/embed.js /Users/bran/personal/orbithall/static/embed.css "$S/widget-demo-5500"/
cp "$S/widget-capture/index.html" "$S/widget-demo-5500/index.html"
docker exec orbithall-db psql -U orbithall -d orbithall_db -At -c \
  "UPDATE sites SET cors_origins = array_append(cors_origins, 'http://localhost:5500') WHERE id = 21 AND NOT ('http://localhost:5500' = ANY(cors_origins)) RETURNING cors_origins"
python3 -m http.server 5500 --bind 127.0.0.1 --directory "$S/widget-demo-5500"   # 백그라운드로 실행
```

- [ ] **Step 4: 브라우저 확인**

Playwright로 `http://localhost:5500/`을 열어 확인한다.
- 최신순 목록, 답글, "삭제된 댓글입니다." 자리와 그 아래 정상으로 보이는 답글, "(수정됨)" 표시
- 목록 아래 "댓글 8개 더 보기" 버튼 → 누르면 다음 페이지가 이어 붙고 버튼이 사라짐
- 콘솔 오류 없음

확인 후 `.playwright-mcp/`를 지운다.

- [ ] **Step 5: 최종 요약 보고**

- 새 CDN 주소(고정 `@1.2.0`, 범위 `@1`)
- 백엔드 정렬 옵션 이름과 기본값
- 위젯 동작(최신순, 더 보기, 작성·수정·삭제 후)
- curl 확인 결과(운영 API, CDN)
- 캡처용 시험 페이지 주소와 로컬 API 주소

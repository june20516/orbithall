# Admin 통계 및 콘텐츠 조회 API

## 작성일
2025-11-05

## 시작일
2025-11-05

## 우선순위
- [ ] 긴급
- [x] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
Admin 사용자가 자신의 사이트 통계(Post 수, 댓글 수)를 확인하고, Post 목록과 댓글 전체를 조회할 수 있는 API 구현. TDD 방식으로 개발.

## 작업 완료 체크리스트

### 1. Database 메서드 추가 (TDD)
- [x] `internal/database/sites_test.go`에 테스트 추가
  - [x] `TestGetSiteStats` - 사이트 통계 조회
- [x] `internal/database/sites.go`에 메서드 구현
  - [x] `GetSiteStats(siteID)` - post_count, comment_count, deleted_comment_count 반환

### 2. Database 메서드 추가 - Post 목록 (TDD)
- [x] `internal/database/posts_test.go`에 테스트 추가
  - [x] `TestListPostsBySite` - Post 목록 조회 (활성/삭제 댓글 수 포함)
- [x] `internal/database/posts.go`에 메서드 구현
  - [x] `ListPostsBySite(siteID)` - Post 목록 with active_comment_count, deleted_comment_count

### 3. Database 메서드 추가 - Admin 댓글 조회 (TDD)
- [x] `internal/database/comments_test.go`에 테스트 추가
  - [x] `TestGetAdminComments` - 삭제된 댓글 포함, IP 전체 노출
- [x] `internal/database/comments.go`에 메서드 추가
  - [x] `GetAdminComments(postID)` - 모든 댓글 조회 (삭제된 것 포함)
  - [x] `getAdminReplies(parentID)` - 대댓글도 삭제된 것 포함

### 4. Admin 핸들러 구현 (TDD)
- [x] `internal/handlers/admin_test.go`에 테스트 추가
  - [x] `TestGetSiteStats` - 통계 조회 성공/권한 없음
  - [x] `TestListSitePosts` - Post 목록 조회 성공/권한 없음
  - [x] `TestGetPostComments` - 댓글 조회 성공/권한 없음
- [x] `internal/handlers/admin.go`에 핸들러 추가
  - [x] `GetSiteStats(w, r)` - GET /admin/sites/:id/stats
  - [x] `ListSitePosts(w, r)` - GET /admin/sites/:id/posts
  - [x] `GetPostComments(w, r)` - GET /admin/posts/:slug/comments?site_id={id}
  - [x] IPAddressUnmasked 필드 추가 (Admin 전용 전체 IP 노출)

### 5. 라우팅 설정
- [x] `cmd/api/main.go` 수정
  - [x] 3개 엔드포인트 등록
- [x] 빌드 확인

### 6. Swagger 문서 추가
- [x] 각 핸들러에 Swagger 주석 추가
- [x] Air에서 자동 `swag init` 실행 설정 (`.air.toml`)

### 7. 테스트
- [x] 모든 유닛 테스트 실행 (`go test ./internal/handlers`) - 45개 테스트 모두 통과

## 작업 목적
- Admin 사용자가 사이트 통계를 한눈에 파악
- Post별 댓글 활동 모니터링
- 문제 댓글 관리 (삭제된 댓글, IP 추적)

## 작업 범위

### 포함 사항
- 사이트 통계 조회 (Post 수, 댓글 수)
- Post 목록 조회 (각 Post별 댓글 수)
- Admin용 댓글 조회 (삭제된 댓글 포함, IP 마스킹 없음)

### 제외 사항
- 댓글 일괄 삭제/관리 기능 (추후)
- 통계 차트/그래프 (프론트엔드)
- 스팸 필터링 (추후)

## 기술적 접근

### 새로운 엔드포인트
```
GET /admin/sites/{id}/stats
  - 응답: { post_count: 10, comment_count: 150 }

GET /admin/sites/{id}/posts
  - 응답: [{ slug, comment_count, created_at }, ...]

GET /admin/posts/{slug}/comments?site_id={id}
  - 응답: 모든 댓글 (삭제된 것 포함, IP 전체)
```

### 권한 검증
- 기존 `HasUserSiteAccess` 헬퍼 함수 재사용
- site_id로 소유권 확인

### 파일 구조
```
orbithall/
├── internal/
│   ├── database/
│   │   ├── sites.go (GetSiteStats 추가)
│   │   ├── sites_test.go
│   │   ├── posts.go (신규)
│   │   ├── posts_test.go (신규)
│   │   ├── comments.go (GetAdminComments 추가)
│   │   └── comments_test.go
│   └── handlers/
│       ├── admin.go (3개 핸들러 추가)
│       └── admin_test.go
└── cmd/api/
    └── main.go
```

## 구현 세부사항

### GetSiteStats SQL
```sql
SELECT
  COUNT(DISTINCT post_slug) as post_count,
  COUNT(c.id) as comment_count
FROM comments c
WHERE c.site_id = $1 AND c.is_deleted = false
```

### ListPostsBySite SQL
```sql
SELECT
  post_slug,
  COUNT(*) as comment_count,
  MAX(created_at) as last_comment_at
FROM comments
WHERE site_id = $1 AND is_deleted = false
GROUP BY post_slug
ORDER BY last_comment_at DESC
```

### GetAdminComments
- 기존 `ListCommentsByPost`와 유사하지만:
  - `is_deleted = true`인 댓글도 포함
  - IP 마스킹 없이 전체 IP 반환
  - 계층 구조 유지

## 의존성
- 작업 012 (Admin API 기본 구조) 완료 필요

## 예상 소요 시간
2-3시간

## 검증 방법
1. 유닛 테스트 모두 통과
2. Swagger UI에서 API 동작 확인
3. 삭제된 댓글이 Admin에서만 보이는지 확인
4. 권한 없는 사용자 접근 시 403 확인

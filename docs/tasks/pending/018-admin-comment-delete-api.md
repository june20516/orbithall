# 어드민 댓글 삭제 API

## 작성일
2026-09-22

## 우선순위
- [ ] 높음
- [x] 보통
- [ ] 낮음

## 작업 개요
사이트 소유자가 어드민에서 스팸·부적절한 댓글을 삭제할 수 있도록 어드민용 댓글 삭제 API를 추가합니다.
어드민 대응 과제: orbithall-admin `docs/pending/admin-comment-delete.md`

## 현재 문제
- 댓글 삭제는 공개 API `DELETE /api/comments/{id}`뿐이고 작성자 비밀번호가 필요함
- `/admin/*` 라우트에는 댓글 삭제가 없어 사이트 소유자가 댓글을 정리할 방법이 없음

## 작업 범위

### 포함
- `DELETE /admin/comments/{id}` (JWT 인증)
  - 댓글 → 포스트 → 사이트를 따라가 `HasUserSiteAccess`로 권한 확인 (권한 없으면 403, 없는 댓글 404)
  - 기존 `database.DeleteComment`(soft delete: `is_deleted`, `deleted_at`) 재사용
  - 이미 삭제된 댓글은 409 또는 멱등 처리 중 선택
- 대댓글이 달린 댓글 삭제 시 동작 확인 (soft delete라 대댓글은 유지되는지)
- 통계(`/admin/sites/{id}/stats`)·게시글 목록의 삭제 수 집계에 반영되는지 확인
- 핸들러·DB 테스트, swagger 주석

### 제외
- 삭제 취소(복구)
- 영구 삭제(hard delete)
- 여러 댓글 일괄 삭제

## 예상 시간
2-3시간

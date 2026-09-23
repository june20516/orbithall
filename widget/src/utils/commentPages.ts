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

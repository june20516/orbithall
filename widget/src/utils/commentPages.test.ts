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
    const found = pages[page];
    if (!found) {
      throw new Error(`no page ${page}`);
    }
    return found;
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

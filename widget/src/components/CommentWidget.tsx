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

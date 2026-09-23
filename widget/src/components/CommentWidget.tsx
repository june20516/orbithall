import { useEffect, useMemo, useRef, useState } from "preact/hooks";
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
  const [reloadError, setReloadError] = useState<string | null>(null);
  const { t } = useI18n();

  // 조회 요청 세대 번호. 위젯은 postSlug가 바뀌어도 언마운트되지 않고
  // prop만 갱신되므로(main.tsx의 MutationObserver 참고), 늦게 도착한 응답이
  // 최신 화면을 덮어쓰지 않도록 모든 조회 경로가 이 번호로 자신을 식별한다.
  const generationRef = useRef(0);

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
    const requestId = ++generationRef.current;
    try {
      setLoading(true);
      const data = await fetchPage(1);
      if (requestId !== generationRef.current) {
        return; // 그 사이 더 최신 조회가 시작됨 — 이 응답은 버린다
      }
      setComments(data.comments || []);
      setPagination(data.pagination);
      setError(null);
      setLoadMoreError(null);
      setReloadError(null);
    } catch (err) {
      if (requestId !== generationRef.current) {
        return;
      }
      console.error("OrbitHall: Failed to load comments", err);
      setError(toErrorMessage(err));
    } finally {
      // 세대 가드는 응답을 목록에 반영하는 것만 막는다.
      // 로딩 표시까지 가드 안에서 끄면, 뒤늦게 끝난 요청이 표시를 켠 채로 남긴다.
      setLoading(false);
    }
  };

  // 펼쳐 둔 범위를 유지한 채 다시 불러오기 (답글 작성, 수정, 삭제 후)
  // 서버에는 이미 반영된 뒤이므로, 재조회가 실패해도 화면의 목록은 지우지 않고
  // reloadError로만 알려 다시 시도할 수 있게 한다.
  const reloadOpenedPages = async () => {
    const lastPage = pagination?.currentPage ?? 1;
    const requestId = ++generationRef.current;
    try {
      const data = await reloadPages(fetchPage, lastPage);
      if (requestId !== generationRef.current) {
        return;
      }
      setComments(data.comments);
      setPagination(data.pagination);
      setError(null);
      setLoadMoreError(null);
      setReloadError(null);
    } catch (err) {
      if (requestId !== generationRef.current) {
        return;
      }
      console.error("OrbitHall: Failed to reload comments", err);
      setReloadError(toErrorMessage(err));
    }
  };

  // 다음 페이지를 이어 붙이기
  const handleLoadMore = async () => {
    if (!pagination || loadingMore) {
      return;
    }

    const requestId = ++generationRef.current;
    try {
      setLoadingMore(true);
      setLoadMoreError(null);
      const data = await fetchPage(pagination.currentPage + 1);
      if (requestId !== generationRef.current) {
        return;
      }
      setComments((current) => mergeCommentPages(current, data.comments || []));
      setPagination(data.pagination);
    } catch (err) {
      if (requestId !== generationRef.current) {
        return;
      }
      console.error("OrbitHall: Failed to load more comments", err);
      setLoadMoreError(t("comments.loadMoreError"));
    } finally {
      // 여기서도 가드를 걸면, 더 보기 응답 전에 작성·수정·삭제가 끼어들었을 때
      // 버튼이 "불러오는 중..." 상태로 굳어 다시 누를 수 없게 된다.
      setLoadingMore(false);
    }
  };

  // 컴포넌트 마운트 시, postSlug 변경 시 첫 페이지 조회
  useEffect(() => {
    setComments([]);
    setPagination(null);
    setError(null);
    setLoadMoreError(null);
    setReloadError(null);
    loadFirstPage();

    // postSlug가 다시 바뀌거나 언마운트되면 세대 번호를 올려, 이전 postSlug로
    // 진행 중이던 요청의 응답이 새 화면에 반영되지 않도록 무효화한다.
    return () => {
      generationRef.current++;
    };
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
  // 답글은 포함하지 않은, 남은 최상위 댓글 수
  const remaining = pagination ? remainingCommentCount(pagination, comments.length) : 0;
  const loadMoreLabel = loadingMore
    ? t("comments.loadingMore")
    : remaining === 1
      ? t("comments.loadMoreOne")
      : t("comments.loadMore").replace("{count}", String(remaining));

  return (
    <div className="orb-widget">
      <div className="orb-header">
        <h3>{t("comments.title")}</h3>
      </div>

      <CommentForm onSubmit={handleCommentSubmit} />

      {loading && <div className="orb-loading">{t("loading")}</div>}

      {error && <div className="orb-error">{error}</div>}

      {reloadError && (
        <div className="orb-reload-error" role="status">
          <span>{reloadError}</span>
          <Button
            label={t("comments.retry")}
            onClick={reloadOpenedPages}
            variant="clear"
            type="secondary"
            size="small"
          />
        </div>
      )}

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
            label={loadMoreLabel}
            onClick={handleLoadMore}
            variant="clear"
            type="secondary"
            disabled={loadingMore}
            aria-busy={loadingMore}
          />
          {loadMoreError && (
            <div className="orb-load-more-error" role="status">
              {loadMoreError}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

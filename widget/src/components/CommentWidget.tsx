import { useState, useEffect, useMemo } from "preact/hooks";
import { CommentList } from "./CommentList";
import { CommentForm } from "./CommentForm";
import type { Comment, CommentSubmitData } from "../types";
import { OrbitHallAPIClient } from "../api/client";
import { getErrorMessage, type ErrorResponse } from "../utils/errorMessages";
import { useI18n } from "../i18n/context";

interface CommentWidgetProps {
  apiUrl: string;
  apiKey: string;
  postSlug: string;
}

export function CommentWidget({
  apiUrl,
  apiKey,
  postSlug,
}: CommentWidgetProps) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { t } = useI18n();

  // API 클라이언트 생성
  const apiClient = useMemo(
    () => new OrbitHallAPIClient(apiUrl, apiKey),
    [apiUrl, apiKey]
  );

  // 댓글 목록 조회
  const fetchComments = async () => {
    try {
      setLoading(true);
      const data = await apiClient.getComments(postSlug);
      setComments(data.comments || []);
      setError(null);
    } catch (err) {
      console.error("OrbitHall: Failed to load comments", err);
      if (err instanceof Error && (err as any).response) {
        const errorResponse = (err as any).response as ErrorResponse;
        setError(getErrorMessage(errorResponse, t));
      } else {
        setError(t("error.NETWORK_ERROR"));
      }
    } finally {
      setLoading(false);
    }
  };

  // 컴포넌트 마운트 시 댓글 조회
  useEffect(() => {
    fetchComments();
  }, [postSlug]);

  // 댓글 작성 핸들러
  const handleCommentSubmit = async (commentData: CommentSubmitData) => {
    try {
      await apiClient.createComment(postSlug, commentData);
      // 댓글 목록 새로고침
      await fetchComments();
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
      // 댓글 목록 새로고침
      await fetchComments();
    } catch (err) {
      console.error("OrbitHall: Failed to update comment", err);
      throw err;
    }
  };

  // 댓글 삭제 핸들러
  const handleCommentDelete = async (commentId: number, password: string) => {
    try {
      await apiClient.deleteComment(commentId, password);
      // 댓글 목록 새로고침
      await fetchComments();
    } catch (err) {
      console.error("OrbitHall: Failed to delete comment", err);
      throw err;
    }
  };

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
    </div>
  );
}

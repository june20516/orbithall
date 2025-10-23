import { useState, useMemo } from "preact/hooks";
import { CommentForm } from "./CommentForm";
import { Button } from "./Button";
import { useI18n } from "../i18n/context";
import type { Comment as CommentType, CommentSubmitData } from "../types";
import { DeleteCommentButton } from "./DeleteCommentButton";
import { getErrorMessage, type ErrorResponse } from "../utils/errorMessages";
import { ErrorOverlay } from "./ErrorOverlay";

interface CommentProps {
  comment: CommentType;
  allComments: CommentType[];
  onReply: (data: CommentSubmitData) => Promise<void>;
  onUpdate: (
    commentId: number,
    content: string,
    password: string
  ) => Promise<void>;
  onDelete: (commentId: number, password: string) => Promise<void>;
}

export function Comment({
  comment,
  allComments,
  onReply,
  onUpdate,
  onDelete,
}: CommentProps) {
  const [showReplyForm, setShowReplyForm] = useState(false);
  const [showEditForm, setShowEditForm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resetDeleteInput, setResetDeleteInput] = useState(0);
  const { t } = useI18n();

  // 현재 댓글의 대댓글 찾기
  const replies = useMemo(() => comment.replies || [], [comment.replies]);

  // 수정/삭제 가능 여부 확인 (30분 이내)
  const isEditable = useMemo(() => {
    const createdAt = new Date(comment.createdAt);
    const now = new Date();
    const diffMinutes = (now.getTime() - createdAt.getTime()) / 60000;
    return diffMinutes <= 30;
  }, [comment.createdAt]);

  // 날짜 포맷팅
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now.getTime() - date.getTime();

    // 1분 미만
    if (diff < 60000) {
      return t("time.justNow");
    }

    // 1시간 미만
    if (diff < 3600000) {
      const minutes = Math.floor(diff / 60000);
      return t("time.minutesAgo").replace("{minutes}", String(minutes));
    }

    // 1일 미만
    if (diff < 86400000) {
      const hours = Math.floor(diff / 3600000);
      return t("time.hoursAgo").replace("{hours}", String(hours));
    }

    // 7일 미만
    if (diff < 604800000) {
      const days = Math.floor(diff / 86400000);
      return t("time.daysAgo").replace("{days}", String(days));
    }

    // 7일 이상: YYYY-MM-DD 형식
    return date.toLocaleDateString();
  };

  // 답글 작성 핸들러
  const handleReplySubmit = async (replyData: CommentSubmitData) => {
    await onReply(replyData);
    setShowReplyForm(false);
  };

  // 수정 핸들러
  const handleEditSubmit = async (editData: CommentSubmitData) => {
    await onUpdate(comment.id, editData.content, editData.password);
    setShowEditForm(false);
    setError(null);
  };

  // 삭제된 댓글 처리
  if (comment.isDeleted) {
    return (
      <div className="orb-comment orb-comment-deleted">
        <div className="orb-comment-content">{t("comment.deleted")}</div>
        {replies.length > 0 && (
          <div className="orb-replies">
            {replies.map((reply) => (
              <Comment
                key={reply.id}
                comment={reply}
                allComments={allComments}
                onReply={onReply}
                onUpdate={onUpdate}
                onDelete={onDelete}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="orb-comment">
      {error && <ErrorOverlay error={error} onClose={() => setError(null)} />}
      <div className="orb-comment-header">
        <div>
          <span className="orb-comment-author">{comment.authorName}</span>
          <span className="orb-comment-date">
            {formatDate(comment.createdAt)}
            {comment.updatedAt !== comment.createdAt && (
              <span className="orb-comment-edited">
                {' '}({formatDate(comment.updatedAt)} {t('time.edited')})
              </span>
            )}
          </span>
        </div>
        {isEditable && (
          <DeleteCommentButton
            comment={comment}
            onDelete={onDelete}
            onError={setError}
            resetTrigger={resetDeleteInput}
          />
        )}
      </div>

      {showEditForm ? (
        // 수정 폼
        <CommentForm
          editMode={true}
          initialContent={comment.content}
          onSubmit={handleEditSubmit}
          onCancel={() => {
            setShowEditForm(false);
            setError(null);
          }}
          onError={setError}
        />
      ) : (
        // 일반 댓글 표시
        <>
          <div className="orb-comment-content">{comment.content}</div>

          <div className="orb-comment-actions">
            {!comment.parentId && (
              <Button
                label={showReplyForm ? t("form.cancel") : t("comment.reply")}
                variant="clear"
                type="primary"
                size="small"
                onClick={() => setShowReplyForm(!showReplyForm)}
              />
            )}
            {isEditable && (
              <Button
                label={t("comment.edit")}
                variant="clear"
                type="secondary"
                size="small"
                onClick={() => {
                  setShowEditForm(true);
                  setResetDeleteInput((prev) => prev + 1);
                }}
              />
            )}
          </div>
        </>
      )}

      {(showReplyForm || replies.length > 0) && (
        <div className="orb-replies">
          {showReplyForm && (
            <div className="orb-reply-form">
              <CommentForm
                parentId={comment.id}
                onSubmit={handleReplySubmit}
                onCancel={() => setShowReplyForm(false)}
              />
            </div>
          )}
          {replies.map((reply) => (
            <Comment
              key={reply.id}
              comment={reply}
              allComments={allComments}
              onReply={onReply}
              onUpdate={onUpdate}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}
    </div>
  );
}

import { Comment } from "./Comment";
import type { Comment as CommentType, CommentSubmitData } from "../types";
import { useI18n } from "../i18n/context";
interface CommentListProps {
  comments: CommentType[];
  onReply: (data: CommentSubmitData) => Promise<void>;
  onUpdate: (
    commentId: number,
    content: string,
    password: string
  ) => Promise<void>;
  onDelete: (commentId: number, password: string) => Promise<void>;
}

export function CommentList({
  comments,
  onReply,
  onUpdate,
  onDelete,
}: CommentListProps) {
  const { t } = useI18n();
  // 최상위 댓글만 필터링 (parentId가 null인 댓글)
  const topLevelComments = comments.filter((comment) => !comment.parentId);

  if (topLevelComments.length === 0) {
    return <div className="orb-empty">{t("empty")}</div>;
  }

  return (
    <div className="orb-comment-list">
      {topLevelComments.map((comment) => (
        <Comment
          key={comment.id}
          comment={comment}
          allComments={comments}
          onReply={onReply}
          onUpdate={onUpdate}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}

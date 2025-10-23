import { useState, useEffect } from "preact/hooks";
import { Button } from "./Button";
import { useI18n } from "../i18n/context";
import type { Comment as CommentType } from "../types";
import { getErrorMessage, type ErrorResponse } from "../utils/errorMessages";
import { ErrorOverlay } from "./ErrorOverlay";

interface DeleteCommentButtonProps {
  comment: CommentType;
  onDelete: (commentId: number, password: string) => Promise<void>;
  onError: (error: string | null) => void;
  resetTrigger: number;
}

export function DeleteCommentButton({
  comment,
  onDelete,
  onError,
  resetTrigger,
}: DeleteCommentButtonProps) {
  const [showDeleteInput, setShowDeleteInput] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [loading, setLoading] = useState(false);
  const { t } = useI18n();

  // resetTrigger가 변경되면 입력창 초기화
  useEffect(() => {
    if (resetTrigger > 0) {
      setShowDeleteInput(false);
      setDeletePassword("");
      onError(null);
    }
  }, [resetTrigger]);

  // 삭제 버튼 클릭 핸들러
  const handleDeleteClick = async () => {
    // 비밀번호 입력이 안 보이면 입력창 표시
    if (!showDeleteInput) {
      setShowDeleteInput(true);
      onError(null);
      return;
    }

    // 비밀번호 입력이 보이면 삭제 실행
    if (!deletePassword) {
      onError(t("delete.error.required"));
      return;
    }

    setLoading(true);

    try {
      await onDelete(comment.id, deletePassword);
      setShowDeleteInput(false);
      setDeletePassword("");
      onError(null);
    } catch (err: any) {
      if (err.response) {
        const errorResponse = err.response as ErrorResponse;
        onError(getErrorMessage(errorResponse, t));
      } else {
        onError(t("delete.error.failed"));
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="orb-delete-actions">
        {showDeleteInput && (
          <>
            <input
              type="password"
              className="orb-input orb-delete-password-input"
              placeholder={t("delete.placeholder.password")}
              value={deletePassword}
              onInput={(e) =>
                setDeletePassword((e.target as HTMLInputElement).value)
              }
              disabled={loading}
              autoFocus
            />
            <Button
              label={t("delete.cancel")}
              variant="clear"
              type="cancel"
              size="small"
              onClick={() => setShowDeleteInput(false)}
              disabled={loading}
            />
          </>
        )}
        <Button
          label={loading ? t("comment.deleting") : t("comment.delete")}
          variant="clear"
          type="danger"
          size="small"
          onClick={handleDeleteClick}
          disabled={loading}
        />
    </div>
  );
}

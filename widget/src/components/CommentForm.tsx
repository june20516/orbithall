import { useState } from 'preact/hooks';
import type { CommentSubmitData } from '../types';
import { useI18n } from '../i18n/context';
import { Button } from './Button';
import { getErrorMessage, type ErrorResponse } from '../utils/errorMessages';

interface CommentFormProps {
  onSubmit: (data: CommentSubmitData) => Promise<void>;
  parentId?: number | null;
  onCancel?: (() => void) | null;
  editMode?: boolean;
  initialContent?: string;
  onError?: ((error: string | null) => void) | null;
}

export function CommentForm({
  onSubmit,
  parentId = null,
  onCancel = null,
  editMode = false,
  initialContent = '',
  onError = null
}: CommentFormProps) {
  const [author, setAuthor] = useState('');
  const [password, setPassword] = useState('');
  const [content, setContent] = useState(initialContent);
  const [submitting, setSubmitting] = useState(false);
  const { t } = useI18n();

  // 폼 유효성 검사
  const isValid = editMode
    ? content.trim() && password.trim()
    : author.trim() && content.trim() && password.trim();

  const handleSubmit = async (e: Event) => {
    e.preventDefault();

    // 수정 모드가 아닐 때만 author 체크
    if (!editMode && !author.trim()) {
      if (onError) {
        onError(t('error.INVALID_INPUT'));
      }
      return;
    }

    if (!content.trim() || !password.trim()) {
      if (onError) {
        onError(t('error.INVALID_INPUT'));
      }
      return;
    }

    setSubmitting(true);
    if (onError) {
      onError(null);
    }

    try {
      await onSubmit({
        authorName: author.trim(),
        password,
        content: content.trim(),
        parentId
      });

      // 폼 초기화 (수정 모드가 아닐 때만)
      if (!editMode) {
        setAuthor('');
        setPassword('');
        setContent('');
      }
    } catch (err: any) {
      if (onError) {
        if (err.response) {
          const errorResponse = err.response as ErrorResponse;
          onError(getErrorMessage(errorResponse, t));
        } else {
          onError(err.message || t('form.error'));
        }
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className={editMode ? "orb-edit-form" : "orb-comment-form"} onSubmit={handleSubmit}>
      {!editMode && (
        <input
          type="text"
          placeholder={t('form.name')}
          value={author}
          onInput={(e) => setAuthor((e.target as HTMLInputElement).value)}
          disabled={submitting}
          className="orb-input"
        />
      )}

      <textarea
        placeholder={editMode ? t('edit.placeholder.content') : t('form.content')}
        value={content}
        onInput={(e) => setContent((e.target as HTMLTextAreaElement).value)}
        disabled={submitting}
        className="orb-textarea"
        rows={4}
      />

      <div className="orb-form-actions">
        <input
          type="password"
          placeholder={t('form.password')}
          value={password}
          onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
          disabled={submitting}
          className="orb-input orb-form-password"
        />
        {onCancel && (
          <Button
            label={editMode ? t('edit.cancel') : t('form.cancel')}
            type="cancel"
            onClick={onCancel}
            disabled={submitting}
          />
        )}
        <Button
          label={submitting
            ? (editMode ? t('comment.editing') : t('form.submitting'))
            : (editMode ? t('edit.submit') : (parentId ? t('form.reply') : t('form.submit')))
          }
          type="primary"
          htmlType="submit"
          disabled={submitting || !isValid}
        />
      </div>
    </form>
  );
}

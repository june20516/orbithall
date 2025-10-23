export const locales = {
  ko: {
    // 헤더
    'comments.title': '댓글',

    // 폼
    'form.name': '이름',
    'form.password': '비밀번호',
    'form.content': '댓글을 입력하세요...',
    'form.submit': '댓글 작성',
    'form.cancel': '취소',
    'form.reply': '답글 작성',
    'form.submitting': '작성 중...',
    'form.error': '댓글 작성에 실패했습니다. 다시 시도해주세요.',

    // 댓글
    'comment.reply': '답글',
    'comment.edit': '수정',
    'comment.delete': '삭제',
    'comment.deleted': '삭제된 댓글입니다.',
    'comment.editing': '수정 중...',
    'comment.deleting': '삭제 중...',

    // 수정 폼
    'edit.title': '댓글 수정',
    'edit.placeholder.content': '댓글 내용을 입력하세요...',
    'edit.placeholder.password': '비밀번호',
    'edit.submit': '수정',
    'edit.cancel': '취소',
    'edit.error.required': '내용과 비밀번호를 입력해주세요.',
    'edit.error.failed': '수정에 실패했습니다.',

    // 삭제 확인
    'delete.title': '댓글 삭제',
    'delete.message': '정말 삭제하시겠습니까?',
    'delete.placeholder.password': '비밀번호',
    'delete.submit': '삭제',
    'delete.cancel': '취소',
    'delete.error.required': '비밀번호를 입력해주세요.',
    'delete.error.failed': '삭제에 실패했습니다.',

    // 시간
    'time.justNow': '방금 전',
    'time.minutesAgo': '{minutes}분 전',
    'time.hoursAgo': '{hours}시간 전',
    'time.daysAgo': '{days}일 전',
    'time.edited': '수정됨',

    // 상태 메시지
    'loading': '댓글을 불러오는 중...',
    'empty': '첫 댓글을 작성해보세요!',

    // 에러 오버레이
    'error.title': '오류',
    'error.close': '닫기',

    // 에러 메시지
    'error.MISSING_API_KEY': 'API 키가 필요합니다.',
    'error.INVALID_API_KEY': '유효하지 않은 API 키입니다.',
    'error.SITE_INACTIVE': '비활성화된 사이트입니다.',
    'error.INVALID_ORIGIN': '허용되지 않은 도메인입니다.',
    'error.INVALID_INPUT': '입력값이 올바르지 않습니다.',
    'error.POST_NOT_FOUND': '게시글을 찾을 수 없습니다.',
    'error.COMMENT_NOT_FOUND': '댓글을 찾을 수 없습니다.',
    'error.WRONG_PASSWORD': '비밀번호가 일치하지 않습니다.',
    'error.EDIT_TIME_EXPIRED': '댓글은 작성 후 30분 이내에만 수정/삭제할 수 있습니다.',
    'error.RATE_LIMIT_EXCEEDED': '너무 많은 요청을 보냈습니다. 잠시 후 다시 시도해주세요.',
    'error.INTERNAL_SERVER_ERROR': '서버 오류가 발생했습니다. 잠시 후 다시 시도해주세요.',
    'error.UNKNOWN_ERROR': '알 수 없는 오류가 발생했습니다.',
    'error.NETWORK_ERROR': '네트워크 연결을 확인해주세요.',
  },

  en: {
    // Header
    'comments.title': 'Comments',

    // Form
    'form.name': 'Name',
    'form.password': 'Password',
    'form.content': 'Write a comment...',
    'form.submit': 'Submit',
    'form.cancel': 'Cancel',
    'form.reply': 'Reply',
    'form.submitting': 'Submitting...',
    'form.error': 'Failed to submit comment. Please try again.',

    // Comment
    'comment.reply': 'Reply',
    'comment.edit': 'Edit',
    'comment.delete': 'Delete',
    'comment.deleted': 'This comment has been deleted.',
    'comment.editing': 'Editing...',
    'comment.deleting': 'Deleting...',

    // Edit form
    'edit.title': 'Edit Comment',
    'edit.placeholder.content': 'Write your comment...',
    'edit.placeholder.password': 'Password',
    'edit.submit': 'Update',
    'edit.cancel': 'Cancel',
    'edit.error.required': 'Please enter content and password.',
    'edit.error.failed': 'Failed to update comment.',

    // Delete confirmation
    'delete.title': 'Delete Comment',
    'delete.message': 'Are you sure you want to delete this comment?',
    'delete.placeholder.password': 'Password',
    'delete.submit': 'Delete',
    'delete.cancel': 'Cancel',
    'delete.error.required': 'Please enter password.',
    'delete.error.failed': 'Failed to delete comment.',

    // Time
    'time.justNow': 'Just now',
    'time.minutesAgo': '{minutes}m ago',
    'time.hoursAgo': '{hours}h ago',
    'time.daysAgo': '{days}d ago',
    'time.edited': 'edited',

    // Status messages
    'loading': 'Loading comments...',
    'empty': 'Be the first to comment!',

    // Error overlay
    'error.title': 'Error',
    'error.close': 'Close',

    // Error messages
    'error.MISSING_API_KEY': 'API key is required.',
    'error.INVALID_API_KEY': 'Invalid API key.',
    'error.SITE_INACTIVE': 'Site is inactive.',
    'error.INVALID_ORIGIN': 'Origin not allowed.',
    'error.INVALID_INPUT': 'Invalid input.',
    'error.POST_NOT_FOUND': 'Post not found.',
    'error.COMMENT_NOT_FOUND': 'Comment not found.',
    'error.WRONG_PASSWORD': 'Wrong password.',
    'error.EDIT_TIME_EXPIRED': 'Comments can only be edited or deleted within 30 minutes.',
    'error.RATE_LIMIT_EXCEEDED': 'Too many requests. Please try again later.',
    'error.INTERNAL_SERVER_ERROR': 'Server error occurred. Please try again later.',
    'error.UNKNOWN_ERROR': 'Unknown error occurred.',
    'error.NETWORK_ERROR': 'Please check your network connection.',
  },
} as const;

export type Locale = keyof typeof locales;
export type TranslationKey = keyof typeof locales.ko;

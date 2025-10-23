// 서버 에러 코드에 대한 사용자 친화적 메시지 맵핑
import type { TranslationKey } from '../i18n/locales';

export interface ErrorResponse {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, string[]>;
  };
}

// 에러 코드를 i18n 키로 변환
export function getErrorTranslationKey(code: string): TranslationKey {
  const key = `error.${code}` as TranslationKey;
  return key;
}

export function getErrorMessage(
  response: ErrorResponse | null,
  t: (key: TranslationKey) => string
): string {
  if (!response || !response.error) {
    return t('error.UNKNOWN_ERROR');
  }

  const { code, message, details } = response.error;

  // 서버에서 받은 에러 코드에 매핑된 메시지 반환
  if (code) {
    const translationKey = getErrorTranslationKey(code);
    const translatedMessage = t(translationKey);

    // 번역된 메시지가 키 그대로면 원본 메시지 사용
    if (translatedMessage !== translationKey) {
      return translatedMessage;
    }
  }

  // 검증 에러의 경우 details를 포함하여 메시지 구성
  if (code === 'INVALID_INPUT' && details) {
    const fieldErrors = Object.entries(details)
      .map(([field, messages]) => `${field}: ${messages.join(', ')}`)
      .join('\n');
    return `${t('error.INVALID_INPUT')}:\n${fieldErrors}`;
  }

  // 서버 메시지가 있으면 사용
  if (message) {
    return message;
  }

  return t('error.UNKNOWN_ERROR');
}

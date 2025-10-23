// camelCase와 snake_case를 상호 변환하는 유틸리티 함수

/**
 * camelCase 문자열을 snake_case로 변환
 * 예: "authorName" -> "author_name"
 */
function camelToSnake(str: string): string {
  return str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
}

/**
 * snake_case 문자열을 camelCase로 변환
 * 예: "author_name" -> "authorName"
 */
function snakeToCamel(str: string): string {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
}

/**
 * 객체의 모든 키를 camelCase에서 snake_case로 변환
 * 중첩된 객체와 배열도 재귀적으로 처리
 */
export function convertKeysToSnakeCase(obj: any): any {
  if (obj === null || obj === undefined) {
    return obj;
  }

  // 배열인 경우: 각 요소를 재귀적으로 변환
  if (Array.isArray(obj)) {
    return obj.map(item => convertKeysToSnakeCase(item));
  }

  // 객체가 아닌 경우 (string, number, boolean 등): 그대로 반환
  if (typeof obj !== 'object') {
    return obj;
  }

  // 객체인 경우: 모든 키를 snake_case로 변환
  const converted: Record<string, any> = {};

  for (const key in obj) {
    if (obj.hasOwnProperty(key)) {
      const snakeKey = camelToSnake(key);
      converted[snakeKey] = convertKeysToSnakeCase(obj[key]);
    }
  }

  return converted;
}

/**
 * 객체의 모든 키를 snake_case에서 camelCase로 변환
 * 중첩된 객체와 배열도 재귀적으로 처리
 */
export function convertKeysToCamelCase(obj: any): any {
  if (obj === null || obj === undefined) {
    return obj;
  }

  // 배열인 경우: 각 요소를 재귀적으로 변환
  if (Array.isArray(obj)) {
    return obj.map(item => convertKeysToCamelCase(item));
  }

  // 객체가 아닌 경우 (string, number, boolean 등): 그대로 반환
  if (typeof obj !== 'object') {
    return obj;
  }

  // 객체인 경우: 모든 키를 camelCase로 변환
  const converted: Record<string, any> = {};

  for (const key in obj) {
    if (obj.hasOwnProperty(key)) {
      const camelKey = snakeToCamel(key);
      converted[camelKey] = convertKeysToCamelCase(obj[key]);
    }
  }

  return converted;
}

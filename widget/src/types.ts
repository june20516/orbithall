// OrbitHall Widget 타입 정의

// 댓글 데이터 구조
export interface Comment {
  id: number;
  postId: number;
  parentId: number | null;
  authorName: string;
  content: string;
  isDeleted: boolean;
  ipAddressMasked?: string;
  replies?: Comment[];
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
}

// API 응답 타입
export interface CommentsResponse {
  comments: Comment[];
}

// 댓글 작성 데이터
export interface CommentSubmitData {
  authorName: string;
  password: string;
  content: string;
  parentId?: number | null;
}

// 위젯 타입
export type WidgetType = 'comments' | 'reactions';

// 위젯별 설정
export interface BaseWidgetConfig {
  apiUrl?: string;
  postSlug: string;
}

export interface CommentsWidgetConfig extends BaseWidgetConfig {
  type: 'comments';
}

export interface ReactionsWidgetConfig extends BaseWidgetConfig {
  type: 'reactions';
}

export type WidgetConfig = CommentsWidgetConfig | ReactionsWidgetConfig;

// OrbitHall 초기화 설정
export interface OrbitHallInitConfig {
  apiKey: string;
  locale?: 'ko' | 'en';
}

// Window 객체에 OrbitHall 타입 추가
declare global {
  interface Window {
    OrbitHall: {
      init: (config: OrbitHallInitConfig) => void;
      destroy: () => void;
    };
  }
}

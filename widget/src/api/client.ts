import type { Comment, CommentSubmitData, CommentsResponse } from "../types";
import type { ErrorResponse } from "../utils/errorMessages";
import {
  convertKeysToSnakeCase,
  convertKeysToCamelCase,
} from "../utils/caseConverter";

// RequestInit를 확장하여 body에 객체를 받을 수 있도록 함
interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: any;
}

export class OrbitHallAPIClient {
  private apiUrl: string;
  private apiKey: string;

  constructor(apiUrl: string, apiKey: string) {
    this.apiUrl = apiUrl;
    this.apiKey = apiKey;
  }

  private async request<T>(
    endpoint: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const url = `${this.apiUrl}${endpoint}`;

    // body가 있으면 snake_case로 변환 후 stringify
    const processedOptions: RequestInit = {
      ...options,
      body: options.body
        ? JSON.stringify(convertKeysToSnakeCase(options.body))
        : undefined,
    };

    const response = await fetch(url, {
      ...processedOptions,
      headers: {
        "Content-Type": "application/json",
        "X-Orbithall-API-Key": this.apiKey,
        ...options.headers,
      },
    });

    if (!response.ok) {
      const errorData = (await response.json()) as ErrorResponse;
      // 에러 객체를 그대로 throw (컴포넌트에서 i18n으로 처리)
      const error = new Error("API Error");
      (error as any).response = errorData;
      throw error;
    }

    // 204 No Content는 빈 응답 반환
    if (response.status === 204) {
      return null as T;
    }

    // 응답을 camelCase로 변환
    const data = await response.json();
    return convertKeysToCamelCase(data);
  }

  async getComments(
    postSlug: string,
    page = 1,
    limit = 50
  ): Promise<CommentsResponse> {
    return this.request<CommentsResponse>(
      `/posts/${postSlug}/comments?page=${page}&limit=${limit}`
    );
  }

  async createComment(
    postSlug: string,
    data: CommentSubmitData
  ): Promise<Comment> {
    return this.request<Comment>(`/posts/${postSlug}/comments`, {
      method: "POST",
      body: data,
    });
  }

  async updateComment(
    commentId: number,
    content: string,
    password: string
  ): Promise<Comment> {
    return this.request<Comment>(`/comments/${commentId}`, {
      method: "PUT",
      body: { content, password },
    });
  }

  async deleteComment(commentId: number, password: string): Promise<void> {
    await this.request<void>(`/comments/${commentId}`, {
      method: "DELETE",
      body: { password },
    });
  }
}

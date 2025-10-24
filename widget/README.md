# Orbithall Widget

Preact 기반의 경량 댓글 위젯입니다. 블로그나 웹사이트에 임베드하여 사용할 수 있습니다.

## 빠른 시작 (사용자용)

### 기본 사용법

HTML 페이지에 다음 코드를 추가하세요:

```html
<!DOCTYPE html>
<html>
<head>
  <!-- CSS 로드 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css">
</head>
<body>
  <!-- 위젯 컨테이너 -->
  <div data-orb-container data-widget-type="comments" data-post-slug="my-post-id"></div>

  <!-- JS 로드 및 초기화 -->
  <script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js"></script>
  <script>
    OrbitHall.init({
      apiKey: 'YOUR_API_KEY'
    });
  </script>
</body>
</html>
```

**필수 속성**:
- `data-orb-container`: 위젯 컨테이너 식별자
- `data-widget-type="comments"`: 위젯 타입 (현재는 comments만 지원)
- `data-post-slug="..."`: 게시글 고유 식별자 (URL slug, ID 등)

**선택 옵션**:
```javascript
OrbitHall.init({
  apiKey: 'YOUR_API_KEY',
  locale: 'ko' // 'ko' 또는 'en' (기본값: 'ko')
});
```

### Next.js App Router

```tsx
// app/layout.tsx
import Script from 'next/script';

export default function RootLayout({ children }) {
  return (
    <html>
      <head>
        <link
          rel="stylesheet"
          href="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css"
        />
      </head>
      <body>
        {children}

        <Script
          src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js"
          onLoad={() => {
            window.OrbitHall.init({
              apiKey: process.env.NEXT_PUBLIC_ORBITHALL_API_KEY
            });
          }}
        />
      </body>
    </html>
  );
}

// app/posts/[slug]/page.tsx
export default function PostPage({ params }: { params: { slug: string } }) {
  return (
    <article>
      <h1>Post Title</h1>
      <div>{/* Post content */}</div>

      {/* 댓글 위젯 */}
      <div
        data-orb-container
        data-widget-type="comments"
        data-post-slug={params.slug}
      />
    </article>
  );
}
```

**주의**: `data-post-slug` 속성이 변경되면 위젯이 자동으로 해당 게시글의 댓글을 로드합니다.

### React

```jsx
import { useEffect } from 'react';

function App() {
  useEffect(() => {
    // 스크립트 로드
    const script = document.createElement('script');
    script.src = 'https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js';
    script.onload = () => {
      window.OrbitHall.init({
        apiKey: process.env.REACT_APP_ORBITHALL_API_KEY
      });
    };
    document.body.appendChild(script);

    return () => {
      // 정리
      window.OrbitHall?.destroy();
      document.body.removeChild(script);
    };
  }, []);

  return (
    <div>
      <h1>My Post</h1>
      <div data-orb-container data-widget-type="comments" data-post-slug="my-post" />
    </div>
  );
}
```


---

## 개발자 가이드

### 기술 스택

- **프레임워크**: Preact 10.27 + TypeScript
- **빌드 도구**: Bun (번들러 + 런타임)
- **스타일**: CSS (CSS 변수 활용)
- **배포**: jsDelivr CDN
- **번들 포맷**: IIFE (Immediately Invoked Function Expression)

### 프로젝트 구조

```
widget/
├── src/
│   ├── main.tsx                    # 엔트리 포인트
│   ├── types.ts                    # TypeScript 타입 정의
│   ├── styles.css                  # 글로벌 스타일
│   ├── components/
│   │   ├── CommentWidget.tsx       # 최상위 컨테이너
│   │   ├── CommentList.tsx         # 댓글 목록
│   │   ├── Comment.tsx             # 댓글 아이템 (재귀 트리)
│   │   ├── CommentForm.tsx         # 댓글 작성 폼
│   │   ├── DeleteCommentButton.tsx # 삭제 버튼
│   │   ├── Button.tsx              # 공통 버튼
│   │   └── ErrorOverlay.tsx        # 에러 표시
│   ├── api/
│   │   └── client.ts               # OrbitHallAPIClient 클래스
│   ├── i18n/
│   │   ├── context.tsx             # I18n Provider
│   │   ├── index.ts                # 다국어 엔트리
│   │   └── locales/                # 언어별 번역
│   └── utils/
│       ├── constants.ts            # 상수
│       ├── caseConverter.ts        # snake_case ↔ camelCase 변환
│       └── errorMessages.ts        # 에러 메시지 처리
├── scripts/
│   └── publish.js                  # 배포 스크립트
├── .env                            # 환경변수 (개발)
├── .env.production                 # 환경변수 (프로덕션)
├── package.json
└── tsconfig.json
```

### 환경 설정

#### 1. 의존성 설치

```bash
bun install
```

#### 2. 환경변수 설정

`.env` 파일:

```bash
# API 엔드포인트
ORB_PUBLIC_API_URL=http://localhost:8080/api
```

`.env.production` 파일:

```bash
# 프로덕션 API 엔드포인트
ORB_PUBLIC_API_URL=https://orbithall.onrender.com/api
```

### 개발

#### 개발 모드 실행

```bash
bun run dev
```

파일 변경 시 자동으로 재빌드되며, 결과물은 `../static/` 폴더에 생성됩니다.

#### 수동 빌드

```bash
bun run build
```

프로덕션 빌드를 실행합니다 (minify 포함).

### 빌드 출력

빌드 결과물:
- `../static/embed.js` (~30KB minified)
- `../static/embed.css` (~8KB)

IIFE 포맷으로 빌드되어 전역 스코프에 `OrbitHall` 객체를 노출합니다.

### 배포

#### 버전 관리 배포

Orbithall은 **버전별 브랜치 배포** 방식을 사용합니다.

1. `package.json`의 `version` 수정:
   ```json
   {
     "version": "1.0.1"
   }
   ```

2. 배포 스크립트 실행:
   ```bash
   bun run publish
   ```

3. 자동으로 다음 작업이 수행됩니다:
   - main 브랜치 최신화
   - 프로덕션 빌드
   - `widget/v{version}` 브랜치 생성
   - 빌드 결과물 커밋 및 푸시
   - jsDelivr CDN에 자동 배포

#### CDN URL

```
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v{version}/static/embed.js
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v{version}/static/embed.css
```

예: `widget/v1.0.0` 브랜치는 다음 URL로 접근 가능:
```
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js
```

#### 배포 안전장치

publish 스크립트는 다음 안전장치를 포함합니다:

1. **버전 중복 체크**: 이미 배포된 버전은 재배포 차단
2. **스크립트 무결성**: publish.js 수정 시 배포 차단
3. **자동 복구**: 에러 발생 시 원래 브랜치로 자동 복귀
4. **변경사항 보존**: stash를 통한 워킹 디렉토리 보호

자세한 내용은 [ADR-006](../docs/adr/006-widget-versioning-deployment-strategy.md)을 참고하세요.

### 아키텍처

#### 컴포넌트 구조

```
CommentWidget (최상위)
  ├─ CommentForm (댓글 작성)
  └─ CommentList
      └─ Comment (재귀)
          ├─ CommentForm (대댓글 작성)
          ├─ DeleteCommentButton
          └─ Comment (자식 댓글, 재귀)
```

#### 초기화 과정

1. `OrbitHall.init({ apiKey })` 호출
2. `data-orb-container` 속성을 가진 모든 요소 탐색
3. 각 컨테이너에 MutationObserver 설정
4. `data-widget-type`에 따라 위젯 렌더링
5. `data-post-slug` 변경 감지 시 자동 리렌더링

#### 상태 관리

Preact의 `useState`, `useEffect` 훅을 사용한 로컬 상태 관리.

- `comments`: 댓글 목록
- `loading`: 로딩 상태
- `error`: 에러 메시지

#### API 통신

`OrbitHallAPIClient` 클래스 사용:

- `getComments(postSlug, page, limit)`: 댓글 조회
- `createComment(postSlug, data)`: 댓글 작성
- `updateComment(commentId, content, password)`: 댓글 수정
- `deleteComment(commentId, password)`: 댓글 삭제

자동으로 snake_case ↔ camelCase 변환을 수행합니다.

### 스타일링

CSS 변수를 사용하여 커스터마이징 가능:

```css
:root {
  /* 색상 */
  --orb-primary-color: #007bff;
  --orb-text-color: #333;
  --orb-bg-color: #fff;
  --orb-border-color: #e0e0e0;

  /* 폰트 */
  --orb-font-family: system-ui, sans-serif;
  --orb-font-size: 14px;

  /* 간격 */
  --orb-spacing: 1rem;
  --orb-border-radius: 8px;
}
```

전체 CSS 변수 목록은 `src/styles.css`를 참고하세요.

### 다국어 지원

`src/i18n/locales/`에 언어별 번역 파일 추가:

```typescript
// src/i18n/locales/ko.ts
export default {
  comments: {
    title: '댓글',
    write: '댓글 작성',
    reply: '답글',
    // ...
  }
};
```

### 문제 해결

#### 위젯이 표시되지 않는 경우

1. 브라우저 콘솔에서 에러 확인
2. `data-orb-container`, `data-widget-type`, `data-post-slug` 속성 확인
3. `OrbitHall.init()` 호출 확인

#### 빌드 에러

```bash
# 캐시 삭제 후 재설치
rm -rf node_modules
bun install
```

#### 배포 실패

```bash
# 브랜치 상태 확인
git status

# 변경사항이 있다면 커밋
git add .
git commit -m "message"
```

### 성능

#### 번들 크기

- JavaScript: ~30KB (minified)
- CSS: ~8KB
- Total: ~38KB
- Gzipped: ~12KB

#### 최적화

- Tree shaking으로 불필요한 코드 제거
- Minification 적용
- IIFE 포맷으로 전역 스코프 오염 최소화
- CSS는 별도 파일로 분리하여 캐싱 최적화

### 라이선스

MIT

### 관련 문서

- [버전 관리 및 배포 전략 ADR](../docs/adr/006-widget-versioning-deployment-strategy.md)
- [프로젝트 README](../README.md)

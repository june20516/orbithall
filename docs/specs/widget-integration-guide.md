# 위젯 임베드 시스템 명세서

## 작성일
2025-10-24

## 버전
v1.0

## 개요
웹사이트에 삽입 가능한 경량 댓글 위젯 시스템입니다. Preact 기반으로 구축되며, CDN을 통해 배포됩니다. 단일 스크립트 태그로 설치 가능하며, data-* 속성을 통한 선언적 설정을 지원합니다.

## 목적 및 배경
- 최소한의 통합 코드로 댓글 시스템 제공
- 경량 번들 크기 (~38KB, gzip ~12KB)로 성능 최적화
- 버전별 배포를 통한 안정성 및 호환성 보장
- 다양한 프레임워크 환경에서 사용 가능
- 다국어 지원으로 글로벌 서비스 대응

## 사용자 스토리
```
AS A 웹사이트 운영자
I WANT 단순한 스크립트 태그로 댓글 시스템을 설치
SO THAT 복잡한 설정 없이 빠르게 댓글 기능을 추가할 수 있다
```

## 기능 요구사항

### 필수 기능
1. **CDN 배포**
   - 설명: jsDelivr를 통한 전역 배포
   - 조건: 버전별 브랜치로 불변성 보장
   - 결과: `widget/v{version}` 브랜치에서 제공

2. **선언적 위젯 초기화**
   - 설명: HTML 속성 기반 설정
   - 조건: `data-orb-container`, `data-widget-type`, `data-post-slug` 속성 필수
   - 결과: 마크업만으로 위젯 렌더링

3. **동적 업데이트 감지**
   - 설명: MutationObserver로 속성 변경 감지
   - 조건: `data-post-slug` 변경 시 자동 리렌더링
   - 결과: SPA 환경에서도 정상 작동

4. **댓글 CRUD 기능**
   - 설명: 댓글 작성, 조회, 수정, 삭제
   - 조건: API 키 기반 인증
   - 결과: 계층형 댓글 구조 지원

5. **다국어 지원**
   - 설명: i18n 시스템
   - 조건: 초기화 시 locale 설정 (ko, en)
   - 결과: 선택된 언어로 UI 표시

### 선택 기능
1. **반응(reactions) 위젯**
   - 설명: 좋아요, 이모지 반응 등
   - 우선순위: 낮음 (추후 구현)

2. **테마 커스터마이징**
   - 설명: CSS 변수를 통한 스타일 수정
   - 우선순위: 중간

## 비기능 요구사항

### 성능
- 번들 크기: JavaScript ~30KB, CSS ~8KB (minified)
- Gzipped: ~12KB
- First Paint: 100ms 이내
- Tree shaking으로 미사용 코드 제거

### 호환성
- 브라우저: Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
- 프레임워크: Vanilla JS, React, Next.js, Vue 등 모든 환경
- TypeScript 타입 정의 포함

### 안정성
- IIFE 포맷으로 전역 스코프 오염 방지
- 에러 핸들링 및 fallback UI
- API 통신 실패 시 사용자 친화적 에러 메시지

### 확장성
- 플러그인 구조 준비 (reactions, analytics 등)
- 위젯 타입별 독립적 렌더링
- 여러 위젯 동시 사용 가능

## 기술 구성

### 프레임워크 및 빌드
```json
{
  "framework": "Preact 10.27",
  "language": "TypeScript 5+",
  "bundler": "Bun",
  "format": "IIFE",
  "target": "browser"
}
```

### 번들 출력
- `static/embed.js`: 위젯 코드 (IIFE 포맷)
- `static/embed.css`: 스타일시트
- 전역 객체: `window.OrbitHall`

### 배포 전략
1. **버전별 브랜치 배포**
   - 패턴: `widget/v{major}.{minor}.{patch}`
   - 예시: `widget/v1.0.0`, `widget/v1.0.1`
   - 불변성: 배포 후 코드 변경 불가 (브랜치 보호)

2. **CDN URL 구조**
   ```
   https://cdn.jsdelivr.net/gh/{org}/{repo}@{branch}/static/embed.js
   https://cdn.jsdelivr.net/gh/{org}/{repo}@{branch}/static/embed.css
   ```

3. **자동화 스크립트**
   - `bun run publish`: 버전 확인 → 빌드 → 브랜치 생성 → 푸시
   - 안전장치: 버전 중복 체크, 스크립트 무결성 검증

## 사용법

### 기본 HTML
```html
<!DOCTYPE html>
<html>
<head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css">
</head>
<body>
  <div data-orb-container data-widget-type="comments" data-post-slug="my-post"></div>

  <script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js"></script>
  <script>
    OrbitHall.init({
      apiKey: 'YOUR_API_KEY'
    });
  </script>
</body>
</html>
```

### Next.js App Router
```tsx
// app/layout.tsx
import Script from 'next/script';

export default function RootLayout({ children }) {
  return (
    <html>
      <head>
        <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css" />
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
```

## 아키텍처

### 초기화 흐름
```
1. OrbitHall.init({ apiKey, locale }) 호출
2. data-orb-container 요소 탐색
3. 각 컨테이너에 고유 ID 생성 (data-orb-id)
4. MutationObserver 설정
5. 위젯 렌더링 (data-widget-type 기반)
6. 속성 변경 시 자동 리렌더링
```

### 컴포넌트 구조
```
CommentWidget (root)
  ├─ I18nProvider (다국어)
  ├─ CommentForm (작성)
  ├─ CommentList
  │   └─ Comment (재귀)
  │       ├─ CommentForm (답글)
  │       ├─ DeleteCommentButton
  │       └─ Comment[] (자식)
  └─ ErrorOverlay
```

### API 통신
`OrbitHallAPIClient` 클래스:
- `getComments(postSlug)`: 댓글 조회
- `createComment(postSlug, data)`: 댓글 작성
- `updateComment(commentId, content, password)`: 댓글 수정
- `deleteComment(commentId, password)`: 댓글 삭제

자동 케이스 변환:
- 요청: camelCase → snake_case
- 응답: snake_case → camelCase

### 상태 관리
Preact 훅 기반 로컬 상태:
- `useState`: 댓글 목록, 로딩, 에러
- `useEffect`: 데이터 로드, cleanup

## 환경변수

### 개발 환경
```env
ORB_PUBLIC_API_URL=http://localhost:8080/api
```

### 프로덕션 환경
```env
ORB_PUBLIC_API_URL=https://orbithall.onrender.com/api
```

빌드 시점에 `process.env.ORB_PUBLIC_API_URL`로 치환됨.

## 제약사항 및 가정

### 제약사항
- 현재 comments 위젯만 지원
- 스타일 커스터마이징은 CSS 변수로만 가능
- 이미지 업로드 미지원

### 가정
- 사용자 브라우저에서 JavaScript 활성화
- 호스트 사이트에서 CSP가 CDN 허용
- API 서버가 CORS 설정 완료

## 의존성

### 외부 라이브러리
- `preact@^10.27.2`: UI 렌더링
- Bun 런타임: 빌드 및 개발 서버

### 내부 모듈
- `src/api/client.ts`: API 클라이언트
- `src/i18n/`: 다국어 시스템
- `src/utils/`: 유틸리티 함수

## 보안 고려사항

### 1. XSS 방지
- 댓글 내용은 텍스트로만 렌더링 (HTML 이스케이프)
- Preact의 기본 XSS 보호 활용

### 2. CSRF 방지
- API 키 기반 인증
- CORS 정책으로 허용된 도메인만 접근

### 3. 전역 스코프 오염 방지
- IIFE로 코드 격리
- `window.OrbitHall`만 노출

### 4. API 키 노출
- 클라이언트 사이드에서 사용되므로 노출됨
- CORS 정책으로 악용 방지
- 서버 측에서 Rate Limiting 필요

## 테스트 시나리오

### 정상 시나리오
1. **시나리오 1: 기본 설치**
   - 전제 조건: HTML에 스크립트 태그 추가
   - 실행 단계: 페이지 로드 → `OrbitHall.init()` 호출
   - 예상 결과: 위젯이 렌더링되고 댓글 목록 표시

2. **시나리오 2: SPA 라우팅**
   - 전제 조건: Next.js 또는 React Router 사용
   - 실행 단계: `data-post-slug` 변경
   - 예상 결과: 자동으로 새 댓글 목록 로드

3. **시나리오 3: 댓글 작성**
   - 전제 조건: 위젯 로드 완료
   - 실행 단계: 이름, 비밀번호, 내용 입력 후 제출
   - 예상 결과: 댓글 목록에 즉시 추가

4. **시나리오 4: 답글 작성**
   - 전제 조건: 기존 댓글 존재
   - 실행 단계: 답글 버튼 클릭 → 폼 작성 → 제출
   - 예상 결과: 계층 구조로 답글 표시

### 예외 시나리오
1. **시나리오 1: API 키 누락**
   - 전제 조건: `OrbitHall.init()` 호출 시 apiKey 없음
   - 실행 단계: 초기화 시도
   - 예상 결과: 콘솔 에러 메시지, 위젯 렌더링 안 됨

2. **시나리오 2: 네트워크 에러**
   - 전제 조건: API 서버 다운 또는 네트워크 끊김
   - 실행 단계: 댓글 로드 시도
   - 예상 결과: ErrorOverlay 표시, 재시도 버튼 제공

3. **시나리오 3: CORS 에러**
   - 전제 조건: 미등록 도메인에서 접근
   - 실행 단계: API 요청
   - 예상 결과: 에러 메시지 표시

### 엣지 케이스
1. **케이스 1: 여러 위젯 동시 사용**
   - 상황: 한 페이지에 여러 `data-orb-container` 존재
   - 처리 방법: 각 컨테이너에 고유 ID 부여, 독립적 렌더링

2. **케이스 2: 빠른 속성 변경**
   - 상황: `data-post-slug`가 짧은 시간에 여러 번 변경
   - 처리 방법: 각 변경마다 리렌더링 (디바운싱 미적용)

3. **케이스 3: 긴 댓글 (10KB+)**
   - 상황: 대용량 댓글 작성 시도
   - 처리 방법: 서버 측에서 제한, 클라이언트 에러 표시

## 마일스톤
- [x] Phase 1: 프로토타입 개발 (2025-10-20)
- [x] Phase 2: 배포 자동화 구현 (2025-10-23)
- [x] Phase 3: v1.0.0 릴리스 (2025-10-24)
- [ ] Phase 4: 모니터링 및 피드백 수집 (진행 중)
- [ ] Phase 5: reactions 위젯 개발 (계획)

## 참고 자료
- Preact 공식 문서: https://preactjs.com/
- jsDelivr CDN: https://www.jsdelivr.com/
- Bun 빌드 도구: https://bun.sh/
- [ADR-006: 위젯 버전 관리 및 배포 전략](../adr/006-widget-versioning-deployment-strategy.md)

## 변경 이력
| 날짜 | 버전 | 변경 내용 | 작성자 |
|------|------|-----------|--------|
| 2025-10-24 | v1.0 | 초안 작성 | Claude |

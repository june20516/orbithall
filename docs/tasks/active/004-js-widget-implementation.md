# [WIP] JS Widget 구현

## 작성일
2025-10-22

## 우선순위
- [x] 높음

## 작업 개요
Disqus처럼 블로그에 임베드 가능한 JS Widget 구현. Bun으로 개발하고 minify/난독화 후 jsDelivr CDN으로 배포.

## 작업 범위
### 포함
- Bun 프로젝트 초기화
- Preact 기반 위젯 개발
- 댓글 조회/작성 UI
- 대댓글 UI (트리 구조)
- 스타일링 (CSS 변수 + 네임스페이스 격리)
- Minify + 빌드 파이프라인
- Publish 스크립트 (브랜치별 배포)
- jsDelivr CDN 배포
- 데모 페이지
- 사용 가이드 (README 업데이트)

## 구현 단계
1. Bun 설치 및 프로젝트 초기화 (5분)
2. Preact + 기본 위젯 구조 (30분)
3. API 연동 (댓글 조회/작성) (20분)
4. 댓글 목록 UI (대댓글 트리 포함) (40분)
5. 스타일링 (CSS 변수 활용) (20분)
6. Publish 스크립트 작성 (15분)
7. 빌드 및 jsDelivr 테스트 (10분)
8. README 사용 가이드 추가 (10분)

## 주요 결정사항
- **툴킷**: Bun (패키지 매니저 + 번들러 + 런타임)
- **프레임워크**: Preact (React 호환, 경량)
- **배포**: jsDelivr (무료 CDN, GitHub 자동 연동)
- **브랜치 전략**: main (프로덕션), develop (베타)
- **CORS**: `AllowedOrigins: "*"` (블로그 도메인에서 API 호출)
- **파일 위치**: widget/ (개발), static/ (빌드 결과)
- **포맷**: IIFE (즉시 실행 함수)
- **난독화**: JavaScript만 난독화, CSS 클래스명은 유지
- **커스터마이징**: CSS 변수 제공 (사용자 스타일 오버라이드 가능)

## jsDelivr 요구사항
- Public 레포지토리 필수 (✅ 충족)
- 파일 크기 50MB 이하 (✅ ~20KB 예상)
- Git push 즉시 CDN 사용 가능 (대기시간 없음)
- URL 포맷: `https://cdn.jsdelivr.net/gh/june20516/orbithall@{branch}/static/embed.js`

## 디렉토리 구조
```
widget/
├── src/
│   ├── main.jsx         # Preact 엔트리
│   ├── components/
│   │   ├── CommentList.jsx
│   │   ├── Comment.jsx
│   │   └── CommentForm.jsx
│   └── styles.css
├── scripts/
│   └── publish.js       # 브랜치별 배포 스크립트
├── demo.html
└── package.json

static/
├── embed.js             # 빌드 결과 (Git 커밋)
└── embed.css            # 빌드 결과 (Git 커밋)
```

## 브랜치별 CDN URL
```
프로덕션 (안정 버전):
https://cdn.jsdelivr.net/gh/june20516/orbithall@main/static/embed.js

베타 (개발 버전):
https://cdn.jsdelivr.net/gh/june20516/orbithall@develop/static/embed.js
```

## Publish 워크플로우
```bash
# 개발
cd widget
bun install
bun run dev

# develop 브랜치 배포 (베타)
bun run publish:develop

# main 브랜치 배포 (프로덕션)
bun run publish:main
```

## 의존성
- 선행: 없음
- 후속: 005-post-reaction-api (위젯에서 반응 기능 추가)

## 예상 시간
2-3시간

---

## 작업 이력
### [2025-10-22] 문서 작성
### [2025-10-22] Bun + Preact + Publish 스크립트 추가

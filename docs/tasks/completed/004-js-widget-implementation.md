# [완료] JS Widget 구현

## 작성일
2025-10-22

## 완료일
2025-10-24

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
- [x] 1. Bun 설치 및 프로젝트 초기화 (5분)
- [x] 2. Preact + 기본 위젯 구조 (30분)
- [x] 3. API 연동 (댓글 조회/작성) (20분)
- [x] 4. 댓글 목록 UI (대댓글 트리 포함) (40분)
- [x] 5. 스타일링 (CSS 변수 활용) (20분)
- [x] 6. Publish 스크립트 작성 (15분)
- [x] 7. 빌드 및 jsDelivr 테스트 (10분)
- [x] 8. README 사용 가이드 추가 (10분)

## 주요 결정사항
- **툴킷**: Bun (패키지 매니저 + 번들러 + 런타임)
- **프레임워크**: Preact 10.27 + TypeScript
- **배포**: jsDelivr (무료 CDN, GitHub 자동 연동)
- **브랜치 전략**: 버전별 배포 (widget/v{version})
- **CORS**: `AllowedOrigins: "*"` (블로그 도메인에서 API 호출)
- **파일 위치**: widget/ (개발), static/ (빌드 결과)
- **포맷**: IIFE (즉시 실행 함수)
- **난독화**: JavaScript만 minify, CSS 클래스명은 유지
- **커스터마이징**: CSS 변수 제공 (사용자 스타일 오버라이드 가능)
- **다국어**: i18n 지원 (ko, en)

## jsDelivr 요구사항
- Public 레포지토리 필수 (✅ 충족)
- 파일 크기 50MB 이하 (✅ ~20KB 예상)
- Git push 즉시 CDN 사용 가능 (대기시간 없음)
- URL 포맷: `https://cdn.jsdelivr.net/gh/june20516/orbithall@{branch}/static/embed.js`

## 디렉토리 구조
```
widget/
├── src/
│   ├── main.tsx                    # Preact 엔트리
│   ├── types.ts                    # TypeScript 타입 정의
│   ├── styles.css                  # 글로벌 스타일
│   ├── components/
│   │   ├── CommentWidget.tsx       # 최상위 위젯 컨테이너
│   │   ├── CommentList.tsx         # 댓글 목록
│   │   ├── Comment.tsx             # 댓글 아이템 (대댓글 트리)
│   │   ├── CommentForm.tsx         # 댓글 작성 폼
│   │   ├── DeleteCommentButton.tsx # 삭제 버튼
│   │   ├── Button.tsx              # 공통 버튼
│   │   └── ErrorOverlay.tsx        # 에러 표시
│   ├── api/                        # API 호출 로직
│   ├── i18n/                       # 다국어 지원
│   └── utils/                      # 유틸리티 함수
├── scripts/
│   └── publish.js                  # 버전별 배포 스크립트
└── package.json

static/
├── embed.js                        # 빌드 결과 (Git 커밋)
└── embed.css                       # 빌드 결과 (Git 커밋)
```

## 버전별 CDN URL
```
버전별 배포 (예: v1.0.0):
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css

버전은 package.json의 version 필드로 관리
```

## Publish 워크플로우
```bash
# 개발
cd widget
bun install
bun run dev

# 버전 배포
# 1. package.json의 version 수정 (예: 1.0.0 -> 1.0.1)
# 2. publish 실행
bun run publish

# 자동으로 수행되는 작업:
# - main 브랜치 최신화
# - 프로덕션 빌드 (minify)
# - widget/v{version} 브랜치 생성
# - jsDelivr CDN 배포
# - 원래 브랜치로 복귀
```

## 의존성
- 선행: 없음
- 후속: 없음 (독립적인 댓글 위젯 구현)

## 예상 시간
2-3시간

---

## 구현 완료 내용
### 핵심 기능
- Preact + TypeScript 기반 위젯
- 댓글 CRUD (생성, 조회, 수정, 삭제)
- 대댓글 트리 구조 (무한 depth)
- 실시간 UI 업데이트
- 다국어 지원 (ko, en)
- 에러 핸들링 및 표시

### 배포 시스템
- 버전 기반 배포 (widget/v{version})
- 버전 중복 체크
- 안전한 브랜치 전환 (stash/복원)
- jsDelivr CDN 자동 배포
- 빌드 크기: JS ~30KB, CSS ~8KB (minified)

### 추가 구현 사항
- TypeScript 타입 안정성
- 컴포넌트 모듈화 (7개 컴포넌트)
- API 레이어 분리
- 유틸리티 함수 분리

## 작업 이력
### [2025-10-22] 문서 작성
### [2025-10-23] Bun + Preact + 기본 컴포넌트 구현
### [2025-10-24] Publish 스크립트 개선 (버전 기반 배포)
### [2025-10-24] 문서화 완료 및 프로젝트 마무리

## 완료 요약
- ✅ 모든 구현 단계 완료 (8/8)
- ✅ jsDelivr CDN 배포 시스템 구축
- ✅ 상세 문서 작성 (README, ADR)
- ✅ 버전 관리 전략 수립 및 적용
- 📦 배포 버전: v1.0.0
- 📄 관련 문서:
  - [ADR-006: Widget 버전 관리 및 배포 전략](../adr/006-widget-versioning-deployment-strategy.md)
  - [Widget README](../../widget/README.md)

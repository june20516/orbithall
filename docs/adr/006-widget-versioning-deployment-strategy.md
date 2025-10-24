# ADR-006: Widget 버전 관리 및 배포 전략

## 상태
승인됨

## 작성일
2025-10-24

## 컨텍스트
Orbithall 위젯을 jsDelivr CDN을 통해 배포하기 위한 버전 관리 및 배포 전략이 필요하다.

### 요구사항
1. 사용자가 특정 버전의 위젯을 사용할 수 있어야 함
2. 배포 과정이 자동화되고 안전해야 함
3. 버전 간 충돌이 발생하지 않아야 함
4. jsDelivr CDN의 캐싱 정책과 호환되어야 함

### 고려한 옵션

#### 옵션 1: 브랜치 기반 배포 (main/develop)
- 장점: 간단한 구조, 명확한 환경 분리
- 단점: 버전 관리 불가, 특정 버전 롤백 어려움, 캐시 무효화 필요

#### 옵션 2: Git 태그 기반 배포
- 장점: 표준 버전 관리 방식
- 단점: jsDelivr는 태그보다 브랜치를 선호, 자동화 복잡도 증가

#### 옵션 3: 버전별 브랜치 배포 (선택됨)
- 장점:
  - 명확한 버전 관리 (widget/v1.0.0, widget/v1.0.1...)
  - jsDelivr 브랜치 캐싱과 완벽 호환
  - 특정 버전으로 쉬운 롤백
  - 여러 버전 동시 제공 가능
- 단점: 브랜치 수 증가

## 결정
**버전별 브랜치 배포 전략**을 채택한다.

### 브랜치 네이밍 규칙
```
widget/v{major}.{minor}.{patch}
```
예: `widget/v1.0.0`, `widget/v1.0.1`, `widget/v2.0.0`

### 배포 프로세스
1. `package.json`의 `version` 필드를 수정
2. `bun run publish` 실행
3. 자동으로 다음 작업 수행:
   - main 브랜치 최신 코드 pull
   - 프로덕션 빌드 (minify)
   - `widget/v{version}` 브랜치 생성
   - 빌드 결과물 커밋 및 푸시
   - 원래 브랜치로 복귀

### CDN URL 형식
```
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v{version}/static/embed.js
https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v{version}/static/embed.css
```

### 안전장치
1. **버전 중복 체크**: 동일 버전 재배포 방지
2. **스크립트 무결성 검증**: publish.js 수정 시 배포 차단
3. **자동 복구**: 에러 발생 시 자동으로 원래 브랜치로 복귀
4. **변경사항 보존**: stash를 통한 워킹 디렉토리 보호

## 결과

### 장점
- 사용자가 특정 버전을 명시적으로 선택 가능
- 버전별 독립적인 캐싱으로 안정성 향상
- 자동화된 배포로 인적 오류 감소
- 브랜치 네임스페이스(`widget/`)로 다른 브랜치와 명확히 구분

### 단점
- 브랜치 수가 버전에 비례하여 증가
- 오래된 버전 브랜치 정리 필요 (수동)

### 사용 예시
```html
<!-- 특정 버전 고정 사용 (권장) -->
<script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.js"></script>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.css">

<!-- 마이너 버전 자동 업데이트 -->
<script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1/static/embed.js"></script>

<!-- 메이저 버전 자동 업데이트 -->
<script src="https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0/static/embed.js"></script>
```

## 참고사항
- jsDelivr는 브랜치별로 24시간 캐싱 적용
- Semantic Versioning 2.0.0 준수
- 호환성 깨지는 변경: major 버전 증가
- 기능 추가: minor 버전 증가
- 버그 수정: patch 버전 증가

## 관련 문서
- [jsDelivr 문서](https://www.jsdelivr.com/documentation)
- [Semantic Versioning](https://semver.org/)
- [Widget README](../../widget/README.md)

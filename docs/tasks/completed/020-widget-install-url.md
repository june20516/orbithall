# README 위젯 설치 주소 수정

## 작성일
2026-09-22

## 우선순위
- [x] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
README "JS Widget > 빠른 시작"의 설치 주소가 404라, 문서대로 설치하면 위젯 스크립트와 스타일을 불러오지 못합니다.

## 현재 문제 (2026-09-22 확인)
- README가 `https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.0.0/static/embed.{js,css}`를 안내함 → **404**
- 원격에 `widget/v1.0.0` 브랜치가 없음 (`widget/v1.0.0`~`v1.0.3`은 로컬에만 있음)
- 원격 릴리스 브랜치: `v1.0.0`, `widget/v1.1.0`, `widget/v1.1.1` (모두 `static/embed.js`, `static/embed.css` 포함)
  - `@widget/v1.1.1` → 200 (정상)
  - `@v1.0.0` → 404 (브랜치에 파일은 있지만 jsDelivr가 `v1.0.0`을 버전 태그로 해석하는 것으로 추정 — 확인 필요)
- ADR 006은 `widget/v{major}.{minor}.{patch}` 브랜치로 배포하는 규칙

## 작업 범위

### 포함
- README 설치 주소를 동작하는 버전으로 수정 (현재 최신은 `widget/v1.1.1`)
- 안내할 버전 정책 결정: 특정 버전 고정 안내 vs 최신 버전 안내
- `v1.0.0` 브랜치를 ADR 규칙(`widget/v1.0.0`)에 맞게 정리할지, 기존 사용자를 위해 `widget/v1.0.x`를 원격에 다시 올릴지 결정
- 릴리스 절차(ADR 006 "배포 프로세스")에 README 주소 갱신 단계 추가 검토

### 제외
- 위젯 기능 변경

## 예상 시간
30분-1시간

## 주요 결정사항
- 원인: README가 존재하지 않는 브랜치 주소(`@widget/v1.0.0`)를 안내했다. 더 근본적으로는 ADR-006의 버전별 브랜치 방식이 jsDelivr 동작과 맞지 않았다(ADR-007 참고).
- 배포 방식을 semver 태그로 바꿨다([ADR-007](../../adr/007-widget-semver-tag-release.md)). 설치 주소는 `@1.1.1`이다.
- 안내 버전 정책: 문서는 고정 버전을 기본으로 안내한다. 범위 주소(`@1`)는 반영이 늦을 수 있다는 점을 함께 적었다.
- 원격 `v1.0.0` 브랜치는 삭제했다(로컬에는 남김). `widget/v1.0.x`는 다시 올리지 않는다. 설치 주소가 404였으므로 쓰는 곳이 없다.
- `@v1.0.0`이 404였던 이유: jsDelivr가 `v1.0.0`을 버전으로 해석해 같은 이름의 태그를 찾았기 때문이다(추정이 맞았음).
- 릴리스 절차에 "설치 주소 버전 갱신" 단계를 넣었다(ADR-007, widget/README.md).

## 작업 이력
### [2026-09-22] semver 태그 전환과 첫 태그 v1.1.1 발행
- `v1.1.1` 태그를 `widget/v1.1.1` 빌드 커밋(`8d2c33c`)에 붙여 push
- CDN 확인 결과 (embed.js, embed.css 동일):

| 주소 | 응답 | x-jsd-version | 유형 | cache-control |
|---|---|---|---|---|
| `@1.1.1` | 200 | 1.1.1 | version | `public, max-age=31536000, s-maxage=31536000, immutable` |
| `@v1.1.1` | 200 | 1.1.1 | version | `public, max-age=31536000, s-maxage=31536000, immutable` |
| `@1` | 200 | 1.1.1 | version | `public, max-age=604800, s-maxage=43200` |
| `@1.1` | 200 | 1.1.1 | version | `public, max-age=604800, s-maxage=43200` |
| `@latest` | 200 | 1.1.1 | version | `public, max-age=604800, s-maxage=43200` |

- `@1.1.1`, `@widget/v1.1.1`, git 태그 파일의 sha256 일치 (embed.js `27a7ca86…`, embed.css `cf9c3c6e…`)
- 원격 `v1.0.0` 브랜치 삭제
- 남은 정리: 블로그(codeverse)가 `@1.1.1`로 옮긴 뒤 원격 `widget/v1.1.0`, `widget/v1.1.1` 삭제

### [2026-09-22] 작업 완료

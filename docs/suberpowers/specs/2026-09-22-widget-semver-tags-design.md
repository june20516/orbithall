# 위젯 CDN 배포: 버전별 브랜치 → semver 태그 설계

## 작성일
2026-09-22

## 배경 (2026-09-22 실측)

ADR-006은 "jsDelivr는 태그보다 브랜치를 선호"를 근거로 버전별 브랜치(`widget/vX.Y.Z`)를 택했지만, 실제 jsDelivr 동작은 반대다.

| 주소 | 응답 | `x-jsd-version-type` | `cache-control` |
|---|---|---|---|
| `@widget/v1.1.1`, `@widget/v1.1.0` | 200 | branch | `max-age=604800, s-maxage=43200` |
| `@widget/v1`, `@widget/v1.0`, `@widget/v1.0.0`, `@v1.0.0` | 404 | - | - |
| (대조) bootstrap `@v5.3.0`, `@5.3.0` | 200 | version | `max-age=31536000, s-maxage=31536000, immutable` |
| (대조) bootstrap `@5`, `@v5`, `@5.3` | 200 (→ 5.3.8) | version | `max-age=604800, s-maxage=43200` |

- 브랜치 주소는 CDN 12시간 캐시이며 내용이 바뀔 수 있다. 불변 보장은 CDN이 아니라 publish.js의 "이미 있는 브랜치면 중단" 검사가 대신 지켜 왔다.
- v 접두 semver 태그는 `@1.1.1`과 `@v1.1.1` 모두로 받을 수 있고, 1년 불변 캐시가 적용된다.
- 범위 주소(`@1`, `@1.1`)는 semver 태그에서만 동작하며, CDN 12시간·브라우저 7일 캐시라 새 버전 반영이 늦을 수 있다.
- 공식 README 기준 purge API는 비공개(이메일 요청)이므로 이점으로 보지 않는다.
- 원격 브랜치 `v1.0.0`은 jsDelivr가 버전으로 해석해 태그를 찾으므로 404다.
- main에는 2025-10 빌드(`02ab1fc`)의 `static/embed.{js,css}`가 커밋되어 있어 `@main`으로 옛 코드가 나간다.

## 결정

- 위젯 릴리스는 `v{version}` semver 태그로 낸다. 이 저장소의 태그는 위젯 전용이다.
- 빌드 결과물은 main에 두지 않는다. 릴리스마다 `origin/main` 위에 결과물만 추가한 커밋을 만들고, 어느 브랜치에도 머지하지 않은 채 그 커밋에 태그를 붙여 태그만 push한다.
- 배포는 작업 도중이 아니라 배포만을 목적으로 실행한다고 전제한다. 그래서 작업 트리 stash 대신 "작업 트리가 깨끗해야 실행"을 요구한다.
- 위젯 코드는 바꾸지 않는다.

```
main:   A ── B ── C              (static/embed.* 없음)
                   \
                    R  ← tag v{version}   (C + static/embed.js, static/embed.css)
```

## 구성 요소

### 1. `widget/scripts/publish.js`

사용법은 그대로 `bun run publish`(`bun --env-file=.env.production run scripts/publish.js`)이며, package.json 스크립트는 바꾸지 않는다.

흐름:
1. 작업 트리에 변경이 있으면 중단한다. publish.js 수정도 여기서 걸린다.
2. `ORB_PUBLIC_API_URL`이 비어 있으면 중단한다. 빌드 시 이 값이 코드에 들어가므로, 값이 있으면 실행 시작 때 출력해 운영 주소인지 눈으로 확인할 수 있게 한다.
3. `git fetch origin`을 실행한다.
4. `git show origin/main:widget/package.json`에서 버전을 읽는다. 현재 브랜치의 package.json이 아니라 배포할 main 기준이다.
5. `refs/tags/v{version}`이 로컬(`git rev-parse -q --verify`)이나 원격(`git ls-remote --tags origin`)에 있으면 중단한다.
6. 현재 위치를 기억한다(브랜치 이름, detached면 커밋 해시).
7. `git checkout --detach origin/main`
8. `bun run build`
9. `git add -f ../static/embed.js ../static/embed.css` 후 `git commit -m "build: widget v{version} [skip ci]"`
10. `git tag -a v{version} -m "widget v{version}"`
11. `git push origin refs/tags/v{version}`을 실행한다. force는 쓰지 않는다. push가 실패하면 로컬 태그를 지워 재시도할 수 있게 한다.
12. 성공과 실패에 관계없이 6에서 기억한 위치로 복귀한다.
13. 고정 버전 주소와 범위 주소를 출력한다.

### 2. main 정리

- `git rm --cached static/embed.js static/embed.css`
- 루트 `.gitignore`에 `static/embed.js`, `static/embed.css`를 추가한다. `static/test.html`은 계속 추적한다.
- main 머지 후 `@main/static/embed.js`는 404가 된다. 사용처는 없다.

### 3. 문서

| 파일 | 변경 |
|---|---|
| `docs/adr/006-...md` | 상태를 Superseded by ADR-007로 바꾸고 상단에 사유 한 줄 |
| `docs/adr/007-widget-semver-tag-release.md` (신규) | 브랜치·태그의 jsDelivr 동작 차이(실측 표), 선택 이유, 릴리스 커밋 구조, 설치 주소 형식, 범위 주소 정책 |
| `docs/adr/README.md` | 목록에 007 추가, 006 상태 갱신 |
| `README.md` | 빠른 시작 주소를 `@1.1.1`로 |
| `widget/README.md` | 설치 주소를 `@1.1.1`로, "배포" 섹션을 태그 방식으로 |
| `docs/specs/widget-integration-guide.md` | 배포 전략 설명과 예시 주소 |
| `docs/tasks/pending/020-...md` | `completed/`로 옮기고 결과 기록 |

`docs/tasks/completed/004-...md`는 당시 기록이므로 수정하지 않는다.

설치 주소 형식:
```
https://cdn.jsdelivr.net/gh/june20516/orbithall@{version}/static/embed.js
https://cdn.jsdelivr.net/gh/june20516/orbithall@{version}/static/embed.css
```

범위 주소 정책: 문서의 기본 안내는 고정 버전(`@1.1.1`)으로 한다. 범위 주소(`@1`)는 쓸 수 있지만 새 버전이 CDN에 최대 12시간, 브라우저에 최대 7일 늦게 반영된다고 명시한다.

### 4. 첫 태그 v1.1.1

- 원격 `widget/v1.1.1`의 커밋 `8d2c33c`에 annotated 태그 `v1.1.1`을 붙인다. 빌드하지 않는다.
- publish.js를 거치지 않는 1회성 수동 작업이다.

### 5. 원격 브랜치 정리

- 원격 `v1.0.0` 브랜치를 삭제한다. 고유 커밋 `a3583f9`는 로컬 `v1.0.0` 브랜치에 남는다.
- `widget/v1.1.0`, `widget/v1.1.1`은 블로그(codeverse)가 새 주소로 옮긴 뒤 지운다. 이번에는 유지한다.
- 로컬에만 있는 `widget/v1.0.0`~`v1.0.3`은 건드리지 않는다.

## 되돌리기 어려운 작업 (실행 직전 사용자 확인)

- `v1.1.1` 태그 push
- 원격 `v1.0.0` 브랜치 삭제

## 에러 처리

- 모든 중단 조건은 git 상태를 바꾸기 전(1~5단계)에 검사한다.
- 7단계 이후 실패하면 원래 위치로 복귀한다. 태그 push 전에 실패하면 원격에는 아무것도 남지 않는다. 로컬에 만들어진 태그는 11단계 실패 시 지운다.
- 커밋 R은 태그 push 실패 시 어디서도 참조하지 않게 되어 git gc 대상이 된다.

## 검증

- publish.js: 실제 push 없이 확인한다.
  - 작업 트리가 깨끗하지 않을 때 중단되는지
  - `ORB_PUBLIC_API_URL`이 비어 있을 때 중단되는지
  - 이미 있는 태그 버전일 때 중단되는지(첫 태그 push 후 1.1.1로 실행)
  - 각 중단 뒤 원래 브랜치와 작업 트리가 그대로인지
- 첫 태그 push 후 curl 확인:
  - `@1.1.1`, `@v1.1.1`은 200, `version`, 1년 immutable이어야 한다
  - `@1`, `@1.1`은 200, `version`이어야 한다
  - `embed.js`와 `embed.css` 모두
  - `@1.1.1`과 `@widget/v1.1.1` 파일의 sha256이 같아야 한다
- 백엔드 동작은 바뀌지 않는다. main 머지 후 Render 재배포 시 `/health`를 확인한다.

## 범위 밖

- 위젯 코드 변경
- 블로그(codeverse) 주소 변경 (codeverse 쪽 작업)
- `widget/v1.1.x` 원격 브랜치 삭제 (블로그 이전 후)

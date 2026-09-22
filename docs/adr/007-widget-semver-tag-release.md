# ADR-007: Widget semver 태그 릴리스

## Status
Accepted (2026-09-22). [ADR-006](006-widget-versioning-deployment-strategy.md)을 대체한다.

## Context
ADR-006은 "jsDelivr는 태그보다 브랜치를 선호"한다는 근거로 버전별 브랜치(`widget/vX.Y.Z`) 배포를 택했다. 2026-09-22에 실측하고 jsDelivr 공식 문서를 확인해 보니, 이 근거는 실제와 반대였다.

### jsDelivr가 브랜치와 태그를 다루는 방식 (2026-09-22 실측)

| 요청 주소 | 응답 | `x-jsd-version-type` | `cache-control` |
|---|---|---|---|
| `@widget/v1.1.1` (브랜치) | 200 | `branch` | `public, max-age=604800, s-maxage=43200` |
| `@widget/v1`, `@widget/v1.0` (ADR-006의 자동 업데이트 예시) | 404 | - | - |
| `@v1.0.0` (브랜치, 같은 이름의 태그 없음) | 404 | - | - |
| bootstrap `@v5.3.0`, `@5.3.0` (v 접두 태그) | 200 | `version` | `public, max-age=31536000, s-maxage=31536000, immutable` |
| bootstrap `@5`, `@v5`, `@5.3` (범위) | 200, 5.3.8로 연결 | `version` | `public, max-age=604800, s-maxage=43200` |

- **브랜치**: CDN 캐시가 12시간이고 내용이 바뀔 수 있다. 범위 주소를 쓸 수 없다. 이름이 semver처럼 보이면(`v1.0.0`) jsDelivr가 버전으로 해석해 태그를 찾기 때문에 404가 된다.
- **semver 태그**: 정확한 버전 주소는 1년 immutable 캐시라 한 번 받으면 바뀌지 않는다. `v` 접두 태그도 `@1.1.1`, `@v1.1.1` 두 형식 모두로 받을 수 있다.
- **범위 주소**(`@1`, `@1.1`): semver 태그에서만 동작한다. CDN 캐시 12시간, 브라우저 캐시 7일이라 새 버전이 늦게 반영된다.
- **purge API**: semver 릴리스에서만 동작한다. 다만 공식 README에 따르면 공개되어 있지 않고, 쓰려면 이메일로 요청해야 한다.

ADR-006 방식에서 "한 번 낸 버전은 바뀌지 않는다"는 보장은 CDN이 아니라 publish.js의 "이미 있는 브랜치면 중단" 검사가 대신 지키고 있었다.

## Decision
위젯 릴리스는 `v{version}` semver git 태그로 낸다. 이 저장소의 태그는 위젯 전용이다. 백엔드는 태그를 쓰지 않는다.

### 릴리스 커밋 구조
빌드 결과물(`static/embed.js`, `static/embed.css`)은 main에 두지 않는다(`.gitignore`에 등록). 릴리스할 때마다 `origin/main` 최신 커밋 위에 빌드 결과물만 추가한 커밋을 만들고, 그 커밋에 태그를 붙여 태그만 push한다. 릴리스 커밋은 어느 브랜치에도 머지하지 않으며, 태그가 가리키고 있어서 사라지지 않는다.

```
main:   A ── B ── C              (static/embed.* 없음)
                   \
                    R  ← tag v{version}   (C + static/embed.js, static/embed.css)
```

jsDelivr `gh` 경로에는 업로드 API가 없고, GitHub 저장소의 ref가 가리키는 파일을 그대로 제공한다. 그래서 배포는 곧 "결과물을 담은 커밋에 태그를 달아 push하는 일"이다.

### 릴리스 절차
1. `widget/package.json`의 `version`을 올리고 main까지 머지한다.
2. 깨끗한 작업 트리에서 `widget/` 디렉터리로 이동해 `bun run publish`를 실행한다. 배포는 작업 도중에 끼워 넣지 않고, 배포만을 목적으로 실행한다.
3. `README.md`, `widget/README.md`, `docs/specs/widget-integration-guide.md`의 설치 주소 버전을 새 버전으로 갱신한다.

### publish 스크립트 안전장치 (`widget/scripts/publish.js`)
- 작업 트리에 변경이 있으면 중단한다. publish.js 수정도 여기에 포함된다.
- `ORB_PUBLIC_API_URL`이 비어 있으면 중단하고, 값이 있으면 출력한다.
- 버전은 현재 브랜치가 아니라 `origin/main`의 `widget/package.json`에서 읽는다.
- 같은 버전 태그가 로컬이나 원격에 있으면 중단한다.
- force push를 쓰지 않는다. 태그 push에 실패하면 로컬 태그를 지운다.
- 성공 여부와 관계없이, 스크립트가 만든 변경을 버리고 실행 전 브랜치로 돌아간다.

### 설치 주소
```
https://cdn.jsdelivr.net/gh/june20516/orbithall@{version}/static/embed.js
https://cdn.jsdelivr.net/gh/june20516/orbithall@{version}/static/embed.css
```

- 문서는 기본으로 고정 버전(예: `@1.1.1`)을 안내한다. 한 번 받으면 바뀌지 않으므로 사용하는 쪽이 동작을 예측할 수 있다.
- 범위 주소(`@1`)도 쓸 수 있다. 다만 새 버전이 CDN에는 최대 12시간, 브라우저에는 최대 7일 늦게 반영된다. 호환성이 깨지는 변경은 major를 올리므로, `@1`은 1.x 안에서만 새 버전을 따라간다.

### 전환
- 첫 태그 `v1.1.1`은 기존 `widget/v1.1.1` 브랜치의 빌드 커밋(`8d2c33c`)에 붙였다. 파일이 같으므로 새 주소로 옮겨도 동작이 바뀌지 않는다.
- 원격 `v1.0.0` 브랜치는 CDN에서 404이고 태그 이름과 헷갈리므로 삭제했다.
- `widget/v1.1.0`, `widget/v1.1.1` 브랜치는 블로그(codeverse)가 새 주소로 옮긴 뒤 삭제한다.

## Consequences
### Positive
- 정확한 버전 주소가 바뀌지 않는다는 보장을 CDN이 직접 해 준다(1년 immutable).
- 범위 주소를 쓸 수 있다.
- 릴리스용 브랜치가 쌓이지 않는다.
- main에 옛 빌드 결과물이 남지 않는다.

### Negative
- 릴리스 커밋이 어느 브랜치에도 없어 `git log main`에는 보이지 않는다. `git tag -l`이나 `git log v{version}`으로 확인해야 한다.
- 문서가 고정 버전을 안내하므로 릴리스할 때마다 설치 주소를 갱신해야 한다.
- `@main/static/embed.js` 주소는 더 이상 쓸 수 없다(404).

## Alternatives Considered
### 버전별 브랜치 유지 (ADR-006)
불변성과 범위 주소를 CDN에서 얻을 수 없어서 제외했다.

### 빌드 결과물을 main에 커밋하고 main 커밋에 태그
태그 방식은 같지만, main 이력에 빌드 커밋이 쌓이고 main 최신 파일이 마지막 릴리스와 어긋날 수 있어서 제외했다.

### 임시 git worktree에서 릴리스 커밋 생성
작업 트리를 건드리지 않는다는 장점이 있다. 하지만 배포를 깨끗한 작업 트리에서 배포 목적으로만 실행한다는 전제에서는 이점이 작고, 릴리스할 때마다 의존성을 새로 설치해야 해서 제외했다.

### npm 배포 후 jsDelivr npm 경로 사용
git 명령 없이 배포할 수 있지만, npm 계정과 공개 패키지가 새로 필요해서 제외했다.

## Related Decisions
- [ADR-006](006-widget-versioning-deployment-strategy.md): 버전별 브랜치 배포 (이 ADR로 대체됨)

## References
- [jsDelivr README](https://github.com/jsdelivr/jsdelivr#readme): GitHub 버전과 범위, 캐싱, purge
- [Semantic Versioning 2.0.0](https://semver.org/)

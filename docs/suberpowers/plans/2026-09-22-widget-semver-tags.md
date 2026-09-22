# 위젯 semver 태그 릴리스 Implementation Plan

> **agentic worker에게:** REQUIRED SUB-SKILL: 이 plan을 task 단위로 구현하려면 suberpower:subagent-driven-development(권장) 또는 suberpower:executing-plans를 사용하세요. Step은 추적을 위해 checkbox(`- [ ]`) 문법을 사용합니다.

**Goal:** 위젯 CDN 배포를 버전별 브랜치(`widget/vX.Y.Z`)에서 semver 태그(`vX.Y.Z`)로 바꾸고, 첫 태그 v1.1.1을 발행해 README 설치 주소 404(작업 020)를 해결한다.

**Architecture:** `publish.js`가 `origin/main`을 detached로 checkout해 빌드하고, 결과물만 `git add -f`로 담은 릴리스 커밋에 annotated 태그를 붙여 태그만 push한다. 릴리스 커밋은 어느 브랜치에도 머지하지 않는다. main은 빌드 결과물을 추적하지 않는다.

**Tech Stack:** Bun 1.3 (Bun Shell `$`), git, jsDelivr `gh` 경로

**Spec:** `docs/suberpowers/specs/2026-09-22-widget-semver-tags-design.md`

**작업 브랜치:** `chore/widget-semver-tags` (origin/develop 기준, upstream 없음)

---

## 파일 구조

| 파일 | 변경 | 책임 |
|---|---|---|
| `.gitignore` | 수정 | 빌드 결과물 무시 |
| `static/embed.js`, `static/embed.css` | 추적 해제 | main에서 제거 (로컬 파일은 유지) |
| `widget/scripts/publish.js` | 전면 교체 | 태그 기반 릴리스 |
| `docs/adr/007-widget-semver-tag-release.md` | 생성 | 새 배포 결정 기록 |
| `docs/adr/006-widget-versioning-deployment-strategy.md` | 수정 | Superseded 표시 |
| `docs/adr/README.md` | 수정 | ADR 목록 |
| `README.md` | 수정 | 설치 주소 |
| `widget/README.md` | 수정 | 설치 주소, 배포 섹션 |
| `docs/specs/widget-integration-guide.md` | 수정 | 배포 전략, 예시 주소 |
| `docs/tasks/pending/020-widget-install-url.md` → `docs/tasks/completed/` | 이동·수정 | 결과 기록 |

커밋 메시지는 모두 다음 두 줄로 끝낸다.

```
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
```

---

### Task 1: main에서 빌드 결과물 추적 해제

**Files:**
- Modify: `.gitignore` (끝에 추가)
- 추적 해제: `static/embed.js`, `static/embed.css`

- [ ] **Step 1: `.gitignore` 끝에 추가**

```gitignore

# widget build output (릴리스 태그 커밋에만 포함, ADR-007)
/static/embed.js
/static/embed.css
```

- [ ] **Step 2: 추적 해제**

실행: `git rm --cached static/embed.js static/embed.css`
기대: `rm 'static/embed.css'`, `rm 'static/embed.js'`

- [ ] **Step 3: 확인**

실행: `git status --porcelain && ls static && git check-ignore -v static/embed.js static/test.html`
기대:
- status에 `D  static/embed.css`, `D  static/embed.js`, ` M .gitignore`
- `ls static`에 embed.css, embed.js, test.html이 남아 있음
- check-ignore는 `static/embed.js`만 출력 (`static/test.html`은 무시되지 않음)

- [ ] **Step 4: Commit**

```bash
git add .gitignore
git commit -F - <<'EOF'
chore: main에서 위젯 빌드 결과물 추적 해제

빌드 결과물은 릴리스 태그 커밋에만 담는다 (ADR-007).

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 2: publish.js를 태그 방식으로 교체

**Files:**
- Modify: `widget/scripts/publish.js` (전체 교체)
- `widget/package.json`의 `publish` 스크립트는 그대로 둔다 (`bun --env-file=.env.production run scripts/publish.js`)

- [ ] **Step 1: 파일 전체를 아래 내용으로 교체**

```js
#!/usr/bin/env bun

// 위젯 릴리스 스크립트 (ADR-007)
// origin/main 최신 코드를 빌드하고, 빌드 결과물만 추가한 릴리스 커밋에 v{version} 태그를 붙여 태그만 push한다.
// 릴리스 커밋은 어느 브랜치에도 머지하지 않는다.
// 사용법: bun run publish (작업 도중이 아니라 배포만을 목적으로, 깨끗한 작업 트리에서 실행)

import { $ } from "bun";

const REPO = "june20516/orbithall";
const BUILD_OUTPUTS = ["../static/embed.js", "../static/embed.css"];

// git 상태를 바꾸기 전에 모든 중단 조건을 검사하고, 배포할 버전과 태그를 돌려준다
async function checkPreconditions() {
  const workingTreeStatus = await $`git status --porcelain`.text();
  if (workingTreeStatus.trim()) {
    throw new Error(
      `작업 트리에 변경사항이 있습니다. 커밋하거나 되돌린 뒤 실행하세요.\n${workingTreeStatus}`
    );
  }

  const apiUrl = process.env.ORB_PUBLIC_API_URL;
  if (!apiUrl) {
    throw new Error(
      "ORB_PUBLIC_API_URL이 비어 있습니다. .env.production을 읽는 `bun run publish`로 실행하세요."
    );
  }
  console.log(`🔗 빌드에 들어갈 API URL: ${apiUrl}`);

  await $`git fetch origin`;

  const mainPackageJson =
    await $`git show origin/main:widget/package.json`.text();
  const version = JSON.parse(mainPackageJson).version;
  const tag = `v${version}`;
  console.log(`📌 배포 버전: ${version} (origin/main 기준)`);

  const localTag = await $`git rev-parse -q --verify refs/tags/${tag}`
    .nothrow()
    .quiet();
  const remoteTag = await $`git ls-remote --tags origin refs/tags/${tag}`.text();
  if (localTag.exitCode === 0 || remoteTag.trim()) {
    throw new Error(
      `${tag} 태그가 이미 있습니다. 한 번 낸 버전은 다시 내지 않습니다. widget/package.json 버전을 올려 main에 반영한 뒤 실행하세요.`
    );
  }

  return { version, tag };
}

// 현재 위치: 브랜치 이름, detached HEAD면 커밋 해시
async function getCurrentLocation() {
  const branch = (await $`git branch --show-current`.text()).trim();
  if (branch) {
    return branch;
  }
  return (await $`git rev-parse HEAD`.text()).trim();
}

// origin/main 위에 빌드 결과물만 추가한 릴리스 커밋을 만들고 태그를 붙인다
async function createReleaseCommit(tag) {
  console.log("\n🔄 origin/main으로 전환 (detached)");
  await $`git checkout --detach origin/main`;

  console.log("\n📦 빌드");
  await $`bun run build`;

  await $`git add -f ${BUILD_OUTPUTS}`;
  await $`git commit -m ${`build: widget ${tag} [skip ci]`}`;
  await $`git tag -a ${tag} -m ${`widget ${tag}`}`;
  console.log(`\n🏷️  릴리스 커밋에 ${tag} 태그를 붙였습니다`);
}

// 태그만 push한다. 실패하면 로컬 태그를 지워 다시 실행할 수 있게 한다
async function pushTag(tag) {
  console.log(`\n📤 ${tag} 태그 push`);
  const result = await $`git push origin refs/tags/${tag}`.nothrow();
  if (result.exitCode !== 0) {
    await $`git tag -d ${tag}`;
    throw new Error(
      `${tag} 태그 push에 실패해 로컬 태그를 지웠습니다. 원인을 해결한 뒤 다시 실행하세요.`
    );
  }
}

async function returnTo(location) {
  if ((await getCurrentLocation()) === location) {
    return;
  }
  const result = await $`git checkout ${location}`.nothrow();
  if (result.exitCode === 0) {
    console.log(`\n↩️  ${location}(으)로 돌아왔습니다`);
  } else {
    console.error(
      `\n⚠️  ${location}(으)로 돌아가지 못했습니다. 직접 checkout 하세요.`
    );
  }
}

function printCdnUrls(version) {
  const major = version.split(".")[0];
  const files = ["embed.js", "embed.css"];

  console.log(`\n✨ v${version} 배포 완료`);
  console.log("\n📍 고정 버전 (권장, 1년 불변 캐시):");
  for (const file of files) {
    console.log(`https://cdn.jsdelivr.net/gh/${REPO}@${version}/static/${file}`);
  }
  console.log(
    `\n📍 범위 버전 @${major} (새 버전 반영이 CDN 최대 12시간, 브라우저 최대 7일 늦음):`
  );
  for (const file of files) {
    console.log(`https://cdn.jsdelivr.net/gh/${REPO}@${major}/static/${file}`);
  }
  console.log(
    "\n📝 README.md, widget/README.md, docs/specs/widget-integration-guide.md의 설치 주소 버전을 갱신하세요."
  );
}

async function publish() {
  let originalLocation = "";

  try {
    const { version, tag } = await checkPreconditions();
    originalLocation = await getCurrentLocation();
    await createReleaseCommit(tag);
    await pushTag(tag);
    printCdnUrls(version);
  } catch (error) {
    console.error(`\n❌ 배포 중단: ${error.message}`);
    process.exitCode = 1;
  } finally {
    if (originalLocation) {
      await returnTo(originalLocation);
    }
  }
}

publish();
```

설계 메모:
- 중단 조건은 모두 `checkPreconditions`에서 git 상태를 바꾸기 전에 검사한다. 그래서 `originalLocation`은 검사를 통과한 뒤에 기록하고, 이 값이 있을 때만 복귀한다.
- `process.exit()` 대신 `throw`와 `process.exitCode`를 쓴다. `process.exit()`를 쓰면 `finally`의 복귀가 실행되지 않는다.
- 기본 `git fetch`는 브랜치에서 닿지 않는 태그를 가져오지 않는다. 릴리스 태그는 브랜치 밖 커밋에 붙으므로, 원격 태그 검사(`ls-remote`)가 반드시 필요하다.

- [ ] **Step 2: 문법 확인**

실행: `cd widget && bun build scripts/publish.js --target bun --outfile /dev/null`
기대: 에러 없이 종료

- [ ] **Step 3: Commit**

```bash
git add widget/scripts/publish.js
git commit -F - <<'EOF'
feat: 위젯 publish를 semver 태그 릴리스로 변경

origin/main을 빌드해 결과물만 담은 릴리스 커밋에 v{version} 태그를 붙이고
태그만 push한다. 버전별 브랜치와 force push, stash를 없애고
깨끗한 작업 트리를 요구한다.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 3: 격리 환경에서 publish.js 검증 (GitHub push 없음)

scratchpad에 로컬 bare 저장소를 원격으로 둔 복제본을 만들어 전체 흐름을 실행한다. 커밋은 없다.

`S=/private/tmp/claude-501/-Users-bran-personal-orbithall/ce5fe0f8-7c5c-4818-8f17-4378f094d655/scratchpad/publish-sandbox`

- [ ] **Step 1: 격리 환경 구성**

```bash
S=/private/tmp/claude-501/-Users-bran-personal-orbithall/ce5fe0f8-7c5c-4818-8f17-4378f094d655/scratchpad/publish-sandbox
rm -rf "$S" && mkdir -p "$S"
git clone -q --bare /Users/bran/personal/orbithall "$S/remote.git"
# 머지 후의 main을 흉내: 격리 원격의 main을 작업 브랜치로 맞춘다
git -C "$S/remote.git" update-ref refs/heads/main refs/heads/chore/widget-semver-tags
git clone -q "$S/remote.git" "$S/work"
cd "$S/work"
git checkout -q -b sandbox-start --no-track origin/chore/widget-semver-tags
cp /Users/bran/personal/orbithall/widget/.env.production widget/
(cd widget && bun install --frozen-lockfile)
git remote -v
git status --porcelain
```

기대:
- `git remote -v`의 fetch/push가 모두 `$S/remote.git`임 (GitHub 아님)
- `git status --porcelain` 출력이 비어 있음 (`.env.production`, `node_modules`는 무시됨)
- `widget/.env`가 없음 (API URL 빈 값 시나리오용)

- [ ] **Step 2: 정상 배포**

실행: `cd "$S/work/widget" && bun run publish`
기대: API URL 출력, `배포 버전: 1.1.1`, 빌드, `v1.1.1 태그 push`, `v1.1.1 배포 완료`, `sandbox-start(으)로 돌아왔습니다`

확인:
```bash
cd "$S/work"
git -C ../remote.git tag -l                                   # v1.1.1
git -C ../remote.git ls-tree -r --name-only v1.1.1 -- static/ # embed.css, embed.js, test.html
test "$(git -C ../remote.git rev-parse 'v1.1.1^{commit}^')" = "$(git -C ../remote.git rev-parse main)" && echo "parent is main"
git -C ../remote.git branch --contains v1.1.1                 # 출력 없음 (어느 브랜치에도 없음)
git branch --show-current                                     # sandbox-start
git status --porcelain                                        # 출력 없음
git -C ../remote.git show v1.1.1:static/embed.js | grep -cF "$(grep '^ORB_PUBLIC_API_URL=' widget/.env.production | cut -d= -f2-)"  # 1 이상
```

- [ ] **Step 3: 같은 버전 재배포 차단**

실행: `cd "$S/work/widget" && bun run publish; echo "exit=$?"`
기대: `❌ 배포 중단: v1.1.1 태그가 이미 있습니다...`, `exit=1`
확인: `git branch --show-current` → `sandbox-start`, `git status --porcelain` → 비어 있음

- [ ] **Step 4: 작업 트리 변경 시 차단**

```bash
cd "$S/work/widget" && touch dirty.txt && bun run publish; echo "exit=$?"; rm dirty.txt
```
기대: `❌ 배포 중단: 작업 트리에 변경사항이 있습니다...`와 `?? widget/dirty.txt`, `exit=1`, 브랜치는 `sandbox-start`

- [ ] **Step 5: API URL 빈 값 차단**

실행: `cd "$S/work/widget" && bun run scripts/publish.js; echo "exit=$?"`
기대: `❌ 배포 중단: ORB_PUBLIC_API_URL이 비어 있습니다...`, `exit=1`, 브랜치는 `sandbox-start`

- [ ] **Step 6: 태그 push 실패 시 로컬 태그 삭제와 복귀**

```bash
cd "$S/work"
git checkout -q -b bump --no-track origin/main
sed -i '' 's/"version": "1.1.1"/"version": "1.1.2"/' widget/package.json
git commit -qam "bump 1.1.2" && git push -q origin bump:main
git checkout -q sandbox-start && git branch -q -D bump
git remote set-url --push origin /nonexistent/remote.git
cd widget && bun run publish; echo "exit=$?"; cd ..
git tag -l v1.1.2                     # 출력 없음
git branch --show-current             # sandbox-start
git status --porcelain                # 출력 없음
git remote set-url --push origin "$S/remote.git"
```
기대: push 에러 후 `❌ 배포 중단: v1.1.2 태그 push에 실패해 로컬 태그를 지웠습니다...`, `exit=1`

- [ ] **Step 7: 정리**

실행: `rm -rf "$S"`

검증 중 하나라도 기대와 다르면 Task 2로 돌아가 수정한 뒤 Task 3 전체를 다시 실행한다.

---

### Task 4: ADR-007 작성, ADR-006 Superseded 표시

**Files:**
- Create: `docs/adr/007-widget-semver-tag-release.md`
- Modify: `docs/adr/006-widget-versioning-deployment-strategy.md:3-4`
- Modify: `docs/adr/README.md` (ADR 목록 표)

- [ ] **Step 1: ADR-007 생성**

````markdown
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
- 성공 여부와 관계없이 실행 전 브랜치로 돌아간다.

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
````

- [ ] **Step 2: ADR-006 상태 수정**

old:
```markdown
## 상태
승인됨
```
new:
```markdown
## 상태
Superseded by [ADR-007](007-widget-semver-tag-release.md) (2026-09-22)

> 이 ADR의 근거였던 "jsDelivr는 태그보다 브랜치를 선호"는 실제와 반대였다. 브랜치 주소는 CDN 캐시가 12시간이고 내용이 바뀔 수 있으며, 아래 "사용 예시"의 범위 주소(`@widget/v1`, `@widget/v1.0`)는 404다. 현재 배포 방식은 ADR-007을 따른다.
```

- [ ] **Step 3: ADR README 목록 수정**

old:
```markdown
| [006](006-widget-versioning-deployment-strategy.md) | Widget 버전 관리 및 배포 전략 | Accepted | 2025-10-24 |
```
new:
```markdown
| [006](006-widget-versioning-deployment-strategy.md) | Widget 버전 관리 및 배포 전략 | Superseded by 007 | 2025-10-24 |
| [007](007-widget-semver-tag-release.md) | Widget semver 태그 릴리스 | Accepted | 2026-09-22 |
```

- [ ] **Step 4: Commit**

```bash
git add docs/adr/007-widget-semver-tag-release.md docs/adr/006-widget-versioning-deployment-strategy.md docs/adr/README.md
git commit -F - <<'EOF'
docs: 위젯 semver 태그 릴리스 ADR-007 추가, ADR-006 대체 표시

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 5: README, widget README, 통합 가이드 주소 수정

**Files:**
- Modify: `README.md:221,230`
- Modify: `widget/README.md:16,23,58,65,106,227-273`
- Modify: `docs/specs/widget-integration-guide.md:29-32,104-118,127,132,151,156,329`

- [ ] **Step 1: widget/README.md 배포 섹션 교체**

old: 227행 `### 배포`부터 273행 `자세한 내용은 [ADR-006](../docs/adr/006-widget-versioning-deployment-strategy.md)을 참고하세요.`까지 전체

new:
````markdown
### 배포

#### 릴리스 절차

위젯은 **semver 태그**(`v{version}`)로 릴리스합니다.

1. `package.json`의 `version`을 올리고 main까지 머지합니다:
   ```json
   {
     "version": "1.1.2"
   }
   ```

2. 깨끗한 작업 트리에서 배포 스크립트를 실행합니다:
   ```bash
   bun run publish
   ```

3. 스크립트가 다음을 수행합니다:
   - `origin/main` 최신 코드를 detached 상태로 checkout
   - 프로덕션 빌드
   - 빌드 결과물만 추가한 릴리스 커밋 생성 (어느 브랜치에도 머지하지 않음)
   - 릴리스 커밋에 `v{version}` 태그를 붙여 태그만 push
   - 원래 브랜치로 복귀

4. `README.md`, `widget/README.md`, `docs/specs/widget-integration-guide.md`의 설치 주소 버전을 갱신합니다.

#### CDN URL

```
https://cdn.jsdelivr.net/gh/june20516/orbithall@{version}/static/embed.js
https://cdn.jsdelivr.net/gh/june20516/orbithall@{version}/static/embed.css
```

- 고정 버전(예: `@1.1.1`)을 권장합니다. jsDelivr가 1년 immutable 캐시로 제공하므로 한 번 받으면 바뀌지 않습니다.
- 범위 버전(예: `@1`)은 1.x의 최신 버전을 따라갑니다. 다만 새 버전이 CDN에는 최대 12시간, 브라우저에는 최대 7일 늦게 반영됩니다.

#### 배포 안전장치

1. **깨끗한 작업 트리 요구**: 변경사항이 있으면 중단 (publish.js 수정 포함)
2. **API URL 확인**: `ORB_PUBLIC_API_URL`이 비어 있으면 중단, 값이 있으면 출력
3. **버전 중복 체크**: 같은 버전 태그가 로컬이나 원격에 있으면 중단
4. **force push 없음**: 태그 push에 실패하면 로컬 태그를 지워 다시 시도 가능
5. **자동 복귀**: 성공 여부와 관계없이 원래 브랜치로 복귀

자세한 내용은 [ADR-007](../docs/adr/007-widget-semver-tag-release.md)을 참고하세요.
````

- [ ] **Step 2: 통합 가이드 CDN 배포 요구사항(29-32행) 수정**

old:
```markdown
   - 조건: 버전별 브랜치로 불변성 보장
   - 결과: `widget/v{version}` 브랜치에서 제공
```
new:
```markdown
   - 조건: semver 태그(`v{version}`)로 불변성 보장
   - 결과: `@{version}` 주소로 제공 (예: `@1.1.1`)
```

- [ ] **Step 3: 통합 가이드 배포 전략(104-118행) 수정**

old:
````markdown
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
````
new:
````markdown
### 배포 전략
1. **semver 태그 릴리스**
   - 태그: `v{major}.{minor}.{patch}` (예: `v1.1.1`)
   - 릴리스 커밋: `origin/main` 위에 빌드 결과물만 추가한 커밋, 어느 브랜치에도 머지하지 않음
   - 불변성: jsDelivr가 정확한 버전을 1년 immutable 캐시로 제공

2. **CDN URL 구조**
   ```
   https://cdn.jsdelivr.net/gh/{org}/{repo}@{version}/static/embed.js
   https://cdn.jsdelivr.net/gh/{org}/{repo}@{version}/static/embed.css
   ```
   - 범위 주소(`@1`)도 동작하지만, 새 버전이 CDN에 최대 12시간, 브라우저에 최대 7일 늦게 반영됨

3. **자동화 스크립트**
   - `bun run publish`: 사전 검사 → origin/main 빌드 → 릴리스 커밋·태그 → 태그 push
   - 안전장치: 깨끗한 작업 트리 요구, 버전 태그 중복 체크, force push 없음
````

- [ ] **Step 4: 통합 가이드 참고 자료(329행) 수정**

old: `- [ADR-006: 위젯 버전 관리 및 배포 전략](../adr/006-widget-versioning-deployment-strategy.md)`
new: `- [ADR-007: 위젯 semver 태그 릴리스](../adr/007-widget-semver-tag-release.md)`

- [ ] **Step 5: 남은 설치 주소 일괄 치환**

실행:
```bash
sed -i '' 's#orbithall@widget/v1\.0\.0/#orbithall@1.1.1/#g' README.md widget/README.md docs/specs/widget-integration-guide.md
```

- [ ] **Step 6: 확인**

실행: `grep -rn "widget/v\|@{branch}\|버전별 브랜치" README.md widget/README.md docs/specs/widget-integration-guide.md; grep -c "orbithall@1.1.1/" README.md widget/README.md docs/specs/widget-integration-guide.md`
기대: 첫 grep 출력 없음. 개수 `README.md:2`, `widget/README.md:5`, `docs/specs/widget-integration-guide.md:4`

- [ ] **Step 7: Commit**

```bash
git add README.md widget/README.md docs/specs/widget-integration-guide.md
git commit -F - <<'EOF'
docs: 위젯 설치 주소와 배포 안내를 semver 태그 방식으로 수정

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 6: 첫 태그 v1.1.1 발행과 CDN 확인 (⚠️ 사용자 확인 필수)

- [ ] **Step 1: 사용자에게 확인 받기**

아래 두 원격 작업을 보여주고 승인받는다. 승인 전에는 실행하지 않는다.
- `git push origin refs/tags/v1.1.1` (태그가 가리키는 커밋: `8d2c33c build: update widget v1.1.1 [skip ci]`)
- `git push origin --delete refs/heads/v1.0.0` (Task 7)

- [ ] **Step 2: 태그 생성과 확인**

```bash
git tag -a v1.1.1 8d2c33c -m "widget v1.1.1"
git rev-parse 'v1.1.1^{commit}'                 # 8d2c33c...
git ls-tree -r --name-only v1.1.1 -- static/    # embed.css, embed.js, test.html
```

- [ ] **Step 3: 태그 push**

실행: `git push origin refs/tags/v1.1.1`
기대: `* [new tag]         v1.1.1 -> v1.1.1`

- [ ] **Step 4: CDN 응답 확인**

```bash
for v in 1.1.1 v1.1.1 1 1.1 latest; do for f in embed.js embed.css; do
  printf "%-7s %-10s " "$v" "$f"
  curl -s -o /dev/null -D - "https://cdn.jsdelivr.net/gh/june20516/orbithall@$v/static/$f" \
    | grep -iE '^(HTTP|x-jsd-version:|x-jsd-version-type|cache-control)' | tr -d '\r' | tr '\n' ' '
  echo
done; done
```
기대:
- `1.1.1`, `v1.1.1`: `200`, `x-jsd-version: 1.1.1`, `version`, `max-age=31536000, s-maxage=31536000, immutable`
- `1`, `1.1`, `latest`: `200`, `x-jsd-version: 1.1.1`, `version`, `max-age=604800, s-maxage=43200`

범위 주소가 처음에 404면 jsDelivr가 버전 목록을 아직 갱신하지 않은 것이다. 몇 분 뒤 다시 확인한다(purge는 비공개라 쓰지 않는다). 결과는 Task 8에 기록한다.

- [ ] **Step 5: 파일 동일성 확인**

```bash
for f in embed.js embed.css; do
  echo "## $f"
  curl -s "https://cdn.jsdelivr.net/gh/june20516/orbithall@1.1.1/static/$f" | shasum -a 256
  curl -s "https://cdn.jsdelivr.net/gh/june20516/orbithall@widget/v1.1.1/static/$f" | shasum -a 256
  git show "v1.1.1:static/$f" | shasum -a 256
done
```
기대: 파일마다 해시 세 개가 같음

- [ ] **Step 6: 실제 저장소에서 재배포 차단 확인**

실행: `cd widget && bun run publish; echo "exit=$?"; cd ..`
기대: API URL 출력, `배포 버전: 1.1.1`, `❌ 배포 중단: v1.1.1 태그가 이미 있습니다...`, `exit=1`
확인: `git branch --show-current` → `chore/widget-semver-tags`, `git status --porcelain` → 비어 있음

---

### Task 7: 원격 v1.0.0 브랜치 삭제 (⚠️ Task 6 Step 1에서 승인받은 경우에만)

- [ ] **Step 1: 삭제**

실행: `git push origin --delete refs/heads/v1.0.0`
기대: `- [deleted]         v1.0.0`

- [ ] **Step 2: 확인**

실행: `git fetch origin --prune && git branch -r && git branch --list v1.0.0`
기대: 원격 목록에 `origin/v1.0.0`이 없고 `origin/widget/v1.1.0`, `origin/widget/v1.1.1`은 남아 있음. 로컬 `v1.0.0` 브랜치(커밋 `a3583f9` 보존)도 남아 있음

---

### Task 8: 작업 문서 020 완료 처리

**Files:**
- Move: `docs/tasks/pending/020-widget-install-url.md` → `docs/tasks/completed/020-widget-install-url.md`

- [ ] **Step 1: 이동**

실행: `git mv docs/tasks/pending/020-widget-install-url.md docs/tasks/completed/020-widget-install-url.md`

- [ ] **Step 2: 파일 끝에 결과 기록 추가**

Task 6 Step 4·5의 실제 출력으로 표를 채운다. 기대와 다른 값이 나오면 그대로 적는다.

```markdown

## 주요 결정사항
- 원인: README가 존재하지 않는 브랜치 주소(`@widget/v1.0.0`)를 안내했다. 더 근본적으로는 ADR-006의 버전별 브랜치 방식이 jsDelivr 동작과 맞지 않았다(ADR-007 참고).
- 배포 방식을 semver 태그로 바꿨다([ADR-007](../../adr/007-widget-semver-tag-release.md)). 설치 주소는 `@1.1.1`이다.
- 안내 버전 정책: 문서는 고정 버전을 기본으로 안내한다. 범위 주소(`@1`)는 반영이 늦을 수 있다는 점을 함께 적었다.
- 원격 `v1.0.0` 브랜치는 삭제했다(로컬에는 남김). `widget/v1.0.x`는 다시 올리지 않는다. 설치 주소가 404였으므로 쓰는 곳이 없다.
- 릴리스 절차에 "설치 주소 버전 갱신" 단계를 넣었다(ADR-007, widget/README.md).

## 작업 이력
### [2026-09-22] semver 태그 전환과 첫 태그 v1.1.1 발행
- `v1.1.1` 태그를 `widget/v1.1.1` 빌드 커밋(`8d2c33c`)에 붙여 push
- CDN 확인 결과:

| 주소 | embed.js | embed.css | x-jsd-version | 유형 | cache-control |
|---|---|---|---|---|---|
| `@1.1.1` | (응답 코드) | (응답 코드) | (값) | (값) | (값) |
| `@v1.1.1` | ... | ... | ... | ... | ... |
| `@1` | ... | ... | ... | ... | ... |
| `@1.1` | ... | ... | ... | ... | ... |
| `@latest` | ... | ... | ... | ... | ... |

- `@1.1.1`과 `@widget/v1.1.1`의 sha256 일치 여부: (결과)
- 남은 정리: 블로그(codeverse)가 `@1.1.1`로 옮긴 뒤 원격 `widget/v1.1.0`, `widget/v1.1.1` 삭제

### [2026-09-22] 작업 완료
```

(표의 괄호 칸은 이 step을 실행할 때 실제 curl 출력으로 채운다. 파일에 괄호가 남으면 안 된다.)

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/completed/020-widget-install-url.md
git commit -F - <<'EOF'
docs: 작업 020 완료 처리 (위젯 설치 주소를 semver 태그로 전환)

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
EOF
```

---

### Task 9: PR과 배포 확인

- [ ] **Step 1: 작업 브랜치 push와 develop 대상 PR 생성**

```bash
git status -sb | head -1    # upstream 없음 확인: "## chore/widget-semver-tags"
git push -u origin chore/widget-semver-tags
gh pr create --repo june20516/orbithall --base develop --head chore/widget-semver-tags \
  --title "chore: 위젯 CDN 배포를 semver 태그로 전환" --body-file <본문 파일>
```

본문에 넣을 것: 배경(ADR-006 근거가 실제와 반대), 변경 사항(publish.js, main 빌드 결과물 추적 해제, ADR-007, 문서 주소), 검증(Task 3 격리 환경 시나리오 5개, Task 6 curl 결과), 원격 작업(v1.1.1 태그 push, v1.0.0 삭제), 백엔드 영향 없음. 끝에 다음을 붙인다.

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_014QmRdV5igLj62aeTC9NCrh
```

- [ ] **Step 2: 머지와 develop → main PR은 사용자 확인 후 진행**

머지 방식은 기존 이력(Merge pull request)을 따른다. main에 머지되면 Render가 재배포되므로 배포 완료 후 확인한다.

실행: `curl -s -o /dev/null -w "%{http_code}\n" https://orbithall.onrender.com/health`
기대: `200`

- [ ] **Step 3: 최종 요약 보고**

- 블로그에 넣을 주소: 고정 `@1.1.1`과 범위 `@1`의 embed.js·embed.css
- curl 확인 결과
- 기존 브랜치 정리 계획: 블로그 이전 후 원격 `widget/v1.1.0`, `widget/v1.1.1` 삭제. 로컬 `widget/v1.0.0`~`v1.0.3`, `v1.0.0`은 사용자가 판단

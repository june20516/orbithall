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

// 시작 전에 작업 트리가 깨끗했으므로 남은 변경은 모두 이 스크립트가 만든 빌드 결과물이다.
// 커밋 전에 실패했을 때 스테이징된 결과물이 원래 위치로 따라가지 않도록 먼저 버린다.
async function returnTo(location) {
  await $`git reset --hard -q`.nothrow();
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

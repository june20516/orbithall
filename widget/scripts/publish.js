#!/usr/bin/env bun

// 버전별 배포 스크립트
// package.json의 version을 읽어서 v{version} 브랜치에 main 브랜치 코드를 빌드/배포
// 사용법: bun run scripts/publish.js

import { $ } from "bun";
import pkg from "../package.json";

// package.json의 version에서 배포 브랜치 이름 생성
const version = pkg.version;
const targetBranch = `v${version}`;

console.log(`\n📌 Publishing version: ${version}`);
console.log(`📍 Target branch: ${targetBranch}`);

async function publish() {
  let stashed = false;
  let currentBranch = "";

  try {
    // 1. 현재 브랜치 저장
    currentBranch = (await $`git branch --show-current`.text()).trim();
    console.log(`\n📍 Current branch: ${currentBranch}`);

    // 2. publish.js가 수정되어 있는지 확인
    const publishJsStatus =
      await $`git status --porcelain scripts/publish.js`.text();
    if (publishJsStatus.trim()) {
      console.error("\n❌ Error: scripts/publish.js has uncommitted changes");
      console.error(
        "Please commit or discard changes to publish.js before running publish"
      );
      process.exit(1);
    }

    // 3. 워킹 디렉토리에 변경사항이 있으면 stash
    const statusCheck = await $`git status --porcelain`.text();
    if (statusCheck.trim()) {
      console.log(`\n💾 Stashing current changes...`);
      await $`git stash push -m "publish script auto-stash"`;
      stashed = true;
      console.log("✅ Changes stashed");
    }

    // 4. main 브랜치로 전환
    console.log(`\n🔄 Switching to main branch...`);
    await $`git checkout main`;
    await $`git pull origin main`;
    console.log("✅ Updated to latest main");

    // 5. 빌드 실행
    console.log(`\n📦 Building widget for version ${version}...`);
    await $`bun run build`;
    console.log("✅ Build complete");

    // 6. 타겟 브랜치 체크아웃 또는 생성
    const branchExists = await $`git rev-parse --verify ${targetBranch}`
      .nothrow()
      .quiet();

    if (branchExists.exitCode === 0) {
      console.log(`\n🔄 Switching to existing ${targetBranch} branch...`);
      await $`git checkout ${targetBranch}`;
      await $`git merge main --no-edit`;
    } else {
      console.log(`\n🆕 Creating new ${targetBranch} branch from main...`);
      await $`git checkout -b ${targetBranch}`;
    }

    // 7. static 디렉토리 변경사항 스테이징
    await $`git add ../static/embed.js ../static/embed.css`;
    console.log("✅ Staged embed.js and embed.css");

    // 8. 변경사항이 있는 경우에만 커밋
    const status = await $`git status --porcelain`.text();
    if (status.trim()) {
      const commitMessage = `build: update widget v${version} [skip ci]`;
      await $`git commit -m ${commitMessage}`;
      console.log(`✅ Committed: ${commitMessage}`);
    } else {
      console.log("⚠️  No changes to commit");
    }

    // 9. Push
    console.log(`\n📤 Pushing to ${targetBranch}...`);
    await $`git push origin ${targetBranch} --force`;

    console.log("\n✨ Publish complete!");
    console.log(`\n📍 CDN URLs:`);
    console.log(
      `https://cdn.jsdelivr.net/gh/june20516/orbithall@${targetBranch}/static/embed.js`
    );
    console.log(
      `https://cdn.jsdelivr.net/gh/june20516/orbithall@${targetBranch}/static/embed.css`
    );
  } catch (error) {
    console.error("\n❌ Publish failed:", error.message);
    process.exit(1);
  } finally {
    // 10. 항상 원래 브랜치로 복귀 시도
    if (currentBranch) {
      try {
        const current = (await $`git branch --show-current`.text()).trim();
        if (current !== currentBranch) {
          console.log(`\n🔄 Returning to ${currentBranch}...`);
          await $`git checkout ${currentBranch}`;
          console.log(`✅ Switched back to ${currentBranch}`);
        }
      } catch (checkoutError) {
        console.error(
          `⚠️  Could not return to ${currentBranch}. Please checkout manually.`
        );
      }
    }

    // 11. 항상 stash 복원 시도
    if (stashed) {
      try {
        console.log(`\n📦 Restoring stashed changes...`);
        await $`git stash pop`;
        console.log("✅ Changes restored");
      } catch (stashError) {
        console.error(
          "⚠️  Could not restore stash automatically. Run: git stash pop"
        );
      }
    }
  }
}

publish();

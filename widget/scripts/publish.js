#!/usr/bin/env bun

// 브랜치별 배포 스크립트
// 사용법: bun run scripts/publish.js [develop|main]

import { $ } from "bun";

const targetBranch = process.argv[2];

if (!targetBranch || !['develop', 'main'].includes(targetBranch)) {
  console.error('Usage: bun run scripts/publish.js [develop|main]');
  process.exit(1);
}

async function publish() {
  try {
    console.log(`\n📦 Building widget for ${targetBranch} branch...`);

    // 1. 빌드 실행
    await $`bun run build`;
    console.log('✅ Build complete');

    // 2. Git 상태 확인
    const status = await $`git status --porcelain`.text();
    if (!status.includes('static/embed.js')) {
      console.log('⚠️  No changes in static/embed.js');
      return;
    }

    // 3. 현재 브랜치 확인
    const currentBranch = (await $`git branch --show-current`.text()).trim();
    console.log(`Current branch: ${currentBranch}`);

    // 4. static 디렉토리 변경사항 스테이징
    await $`git add ../static/embed.js ../static/embed.css`;
    console.log('✅ Staged static/embed.js and embed.css');

    // 5. 커밋
    const commitMessage = `build: update widget for ${targetBranch} [skip ci]`;
    await $`git commit -m ${commitMessage}`;
    console.log(`✅ Committed: ${commitMessage}`);

    // 6. 타겟 브랜치로 체크아웃 (필요시)
    if (currentBranch !== targetBranch) {
      console.log(`\n🔄 Switching to ${targetBranch} branch...`);
      await $`git checkout ${targetBranch}`;
      await $`git merge ${currentBranch}`;
    }

    // 7. Push
    console.log(`\n📤 Pushing to ${targetBranch}...`);
    await $`git push origin ${targetBranch}`;

    console.log('\n✨ Publish complete!');
    console.log(`\n📍 CDN URLs:`);
    console.log(`https://cdn.jsdelivr.net/gh/june20516/orbithall@${targetBranch}/static/embed.js`);
    console.log(`https://cdn.jsdelivr.net/gh/june20516/orbithall@${targetBranch}/static/embed.css`);

    // 원래 브랜치로 복귀
    if (currentBranch !== targetBranch) {
      await $`git checkout ${currentBranch}`;
      console.log(`\n✅ Switched back to ${currentBranch}`);
    }

  } catch (error) {
    console.error('\n❌ Publish failed:', error.message);
    process.exit(1);
  }
}

publish();

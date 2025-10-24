---
description: 세션 시작 시 프로젝트 컨텍스트 로드 (프로젝트 규칙, 현재 작업, git 상태)
---

프로젝트 warmup을 수행합니다:

1. 현재 디렉토리 확인 (`pwd`)
2. Git 루트 디렉토리 확인 (`git rev-parse --show-toplevel`)
3. Git 루트 기준으로 다음 파일/폴더 확인:
   - `.claude-project-rules.md` 파일 읽기
   - `docs/tasks/active/` 폴더의 작업 문서 목록
   - 서브디렉토리인 경우 해당 디렉토리의 README.md도 확인
4. `git status`로 현재 변경사항 확인
5. 최근 커밋 5개 확인 (`git log --oneline -5`)
6. 현재 작업 컨텍스트를 요약하여 사용자에게 제공

**중요**: 이 프로젝트는 문서 기반 Task-Driven 개발을 사용합니다.
- 작업 문서: `docs/tasks/active/`, `docs/tasks/pending/`
- TodoWrite는 세션 내 임시 메모용만 사용
- 향후 작업은 반드시 `docs/tasks/pending/`에 마크다운 문서로 생성

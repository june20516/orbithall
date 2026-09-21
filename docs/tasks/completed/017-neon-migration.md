# 운영 DB를 Neon으로 이전

## 작성일
2026-09-18

## 우선순위
- [x] 높음

## 작업 개요
운영 DB를 Supabase 무료 플랜에서 Neon 무료 플랜으로 이전한다. Supabase는 1주일 미사용 시 프로젝트를 일시정지해 수동 복구가 필요하지만, Neon은 컴퓨트만 자동으로 정지/재개되어 관리가 필요 없다. 기존 데이터는 이전하지 않는다.

## 작업 범위
### 포함
- Neon 프로젝트 생성 및 연결 설정
- Render 환경변수 교체 및 재배포 (배포 시 마이그레이션 자동 실행)
- 기존 API 키로 사이트 재등록
- 코드 주석과 문서에서 Supabase 전용 서술 제거
- 로컬 Docker PostgreSQL을 18로 업그레이드

### 제외
- 기존 댓글 데이터 이전
- Neon 전용 도구 도입 (neon.ts, neon CLI, MCP)

## 구현 단계
1. Neon 프로젝트 생성: Singapore 리전, 결제수단 미등록, 오토스케일 상한 0.25 CU (5분)
2. Render `DATABASE_URL` 교체 후 재배포, 로그에서 마이그레이션 완료 확인 (5분)
3. Neon SQL Editor에서 기존 API 키로 사이트 재등록 (5분)
4. 관리자 로그인 후 `user_sites` 연결 (5분)
5. 로컬 Docker PostgreSQL 18 업그레이드 (15분)

## 주요 결정사항
- Neon 무료 플랜: 결제수단이 없으면 초과 과금 경로가 없고, 한도 초과 시 과금 대신 차단된다
  - 저장 0.5GB 초과: 저장을 늘리는 쓰기가 실패 (읽기는 정상)
  - 컴퓨트 월 100 CU-hours 초과: 다음 주기까지 컴퓨트 정지 (앱 기동 실패)
- Singapore 리전: Render(API)와 같은 리전. Neon에는 한국/일본 리전이 없고, 리전은 생성 후 변경 불가
- direct 연결 사용: golang-migrate가 세션 단위 advisory lock을 사용하므로 transaction 모드 풀러(`-pooler`)와 호환되지 않음
- 업타임 핑은 `/health` 사용: `/health/db`는 쿼리를 실행해 컴퓨트를 계속 깨우므로 무료 한도를 소진함
- 오토스케일 상한 0.25 CU: 상한이 높으면 같은 시간에 CU-hours가 더 빨리 소모됨
- Neon 전용 도구 미사용: 표준 `DATABASE_URL`만 사용해 다른 PostgreSQL로 옮길 때 환경변수만 바꾸면 되도록 유지
- 로컬 PostgreSQL 18: Neon(18.6)과 로컬/테스트 환경의 메이저 버전을 맞춤

## 환경변수
```
DATABASE_URL=postgresql://neondb_owner:<PASSWORD>@<ENDPOINT_ID>.c-4.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require
```

## 사이트 재등록 SQL
```sql
INSERT INTO sites (name, domain, api_key, cors_origins)
VALUES ('<사이트명>', '<도메인>', '<기존 API 키>', ARRAY['<허용 Origin>']);

-- 관리자로 한 번 로그인해 users 행이 생긴 뒤 실행
-- Neon SQL Editor에서는 INSERT ... SELECT 형태가 문법 에러로 실패하므로 id를 먼저 조회해 값으로 넣음
SELECT
  (SELECT id FROM users WHERE email = '<관리자 이메일>') AS user_id,
  (SELECT id FROM sites WHERE domain = '<도메인>') AS site_id;

INSERT INTO user_sites (user_id, site_id, role)
VALUES (<user_id>, <site_id>, 'owner');
```

## 의존성
- 선행: 003-deployment

## 예상 시간
30분

---

## 작업 이력
### [2026-09-18] 작업 문서 작성, 코드 주석과 문서의 Supabase 서술 제거
### [2026-09-18] 로컬 PostgreSQL 16 → 18 업그레이드
- 로컬 데이터 백업 후 볼륨 재생성, 복원 후 테이블별 건수 일치 확인
- PG18에서 전체 테스트 통과, 테스트 DB 마이그레이션 버전 4 적용 확인
- 운영 이미지와 같은 migrate CLI v4.17.0으로 PG18 빈 DB에 마이그레이션 1~4 적용 확인
### [2026-09-18] 사이트 재등록 SQL 수정
- Neon SQL Editor에서 INSERT ... SELECT가 `syntax error at or near "INTO"`로 실패하여 id 조회 후 값으로 넣는 방식으로 변경
### [2026-09-18] 운영 전환 완료
- Render가 Neon에 연결되어 배포 시 마이그레이션 적용, 운영 `/health/db` 정상 응답 확인
- 관리자 사용자, codeverse 사이트, 소유 관계 등록 완료
- 기존 Supabase 프로젝트는 비활성 정책으로 이미 삭제된 상태라 별도 정리 불필요
### [2026-09-18] 작업 완료

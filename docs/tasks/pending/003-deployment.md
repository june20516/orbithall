# [WIP] Render + Supabase 배포

## 작성일
2025-10-22

## 우선순위
- [x] 높음

## 작업 개요
무료 영구 플랫폼(Render + Supabase)에 댓글 시스템 배포

## 배포 전략

### 선택: Supabase (DB) + Render (API)

| 플랫폼 | 역할 | 제한 | 기간 |
|--------|------|------|------|
| Supabase | PostgreSQL | 500MB, 1주일 미사용 시 일시정지 | 영구 무료 |
| Render | Go API | 15분 유휴 시 슬립, 750시간/월 | 영구 무료 |

**선택 이유**:
- 기간 제한 없음 (Railway 30일 trial, Fly.io 무료 플랜 없음)
- 표준 PostgreSQL (벤더락인 없음)
- 코드 수정 불필요 (DATABASE_URL만 변경)

## 작업 범위

### 포함
- Supabase 프로젝트 생성 및 마이그레이션
- Render Web Service 배포 (Dockerfile)
- 환경변수 설정 (DATABASE_URL, ENV)
- 테스트 사이트 등록 및 API 테스트

### 제외
- 커스텀 도메인
- Rate Limiting (별도 작업 002)
- 모니터링 도구

## 체크리스트

### 수동 작업 (사용자)
- [ ] Supabase 계정 생성 및 로그인
- [ ] Supabase 프로젝트 생성 (Region: Seoul, Plan: Free)
- [ ] DATABASE_URL 복사 및 저장
- [ ] Render 계정 생성 및 GitHub 연동
- [ ] Render Web Service 생성 (저장소: june20516/orbithall, Branch: main)
- [ ] Render 환경변수 설정 (DATABASE_URL, ENV)

### 코드 작업
- [ ] 로컬 .env 파일에 Supabase DATABASE_URL 설정
- [ ] Supabase에 마이그레이션 실행
- [ ] Supabase에 테스트 사이트 등록 (SQL)
- [ ] Render 배포 로그 확인 (마이그레이션 성공 여부)
- [ ] API 헬스체크 테스트
- [ ] 댓글 CRUD API 테스트

### 문서화
- [ ] Supabase 프로젝트 URL 기록
- [ ] Render 서비스 URL 기록
- [ ] 테스트 API 키 기록

## 구현 단계

### 1. Supabase 설정 (10분)
1. 프로젝트 생성 (https://supabase.com)
   - Region: Northeast Asia (Seoul)
   - Plan: Free
2. DATABASE_URL 복사 (Settings → Database → URI)
3. 마이그레이션 실행
4. 테스트 사이트 등록

### 2. Render 설정 (15분)
1. Web Service 생성 (https://render.com)
   - GitHub 저장소 연결: `june20516/orbithall`
   - Branch: `main` (자동 배포)
   - Runtime: Docker
   - Region: Singapore
   - Instance: Free
2. 환경변수 설정
   - `DATABASE_URL`: (Supabase에서 복사)
   - `ENV`: `production`
3. Auto-Deploy 확인 (Settings → Build & Deploy)
   - Auto-Deploy: `Yes` (main 브랜치 푸시 시 자동 배포)

### 3. 검증 (10분)
- 헬스체크 (`/health`)
- 댓글 CRUD API 테스트
- Supabase 데이터 확인

## 주요 결정사항

### CORS 전략
- **동적 CORS**: 사이트별 `cors_origins` 배열 검증 (AuthMiddleware)
- **글로벌 CORS**: `AllowedOrigins: ["*"]` (preflight만 처리)
- **환경변수 불필요**: DB에서 동적 관리

### 환경변수
**Render**:
- `DATABASE_URL`: Supabase URI
- `ENV`: `production`

**로컬(.env)**:
- `DATABASE_URL`: Supabase URI (테스트용)
- `ENV`: `development`

### 테스트 사이트 등록 (Supabase SQL Editor)
```sql
INSERT INTO sites (name, domain, cors_origins, api_key, is_active)
VALUES (
  'Test Blog',
  'localhost:3000',
  ARRAY['http://localhost:3000']::TEXT[],
  'test-api-key-replace-later',
  TRUE
);
```

## 제약사항 및 대응

### Render
- **슬립 (15분 유휴)**: 블로그 로드 시 헬스체크 호출로 대응
- **750시간/월**: 충분 (슬립으로 실사용 적음)

### Supabase
- **일시정지 (1주일 미사용)**: 주 1회 접속으로 대응
- **500MB 제한**: 100만 댓글 가능 (충분)
- **프로젝트 2개 제한**: 한 프로젝트에 여러 사이트 등록

## 의존성
- 선행: 댓글 CRUD API (완료)
- 후속: Rate Limiting (002), 프로덕션 환경변수 관리 (004)

## 예상 시간
50분 (Supabase 10분 + Render 15분 + 로컬 테스트 10분 + 검증 10분 + 문서 5분)

## 참고 자료
- Supabase 문서: https://supabase.com/docs
- Render 문서: https://render.com/docs
- Render Dockerfile: https://render.com/docs/deploy-from-dockerfile

---

## 실제 배포 정보

### Supabase
- **프로젝트 ID**: dinwpsvokpkbdfwqvjhz
- **Region**: ap-northeast-2 (Seoul)
- **DATABASE_URL**: `postgresql://postgres.dinwpsvokpkbdfwqvjhz:[PASSWORD]@aws-1-ap-northeast-2.pooler.supabase.com:5432/postgres`
- **Connection Type**: Session Pooler (IPv4 호환)

### 사이트 정보
- **Name**: Bran's codeverse
- **Domain**: june20516.github.io
- **CORS Origins**:
  - `http://localhost:3000` (개발)
  - `https://june20516.github.io` (프로덕션)
- **API Key**: `orb_live_429eaf8d5c35a8684dc2eb1df86cb3bcd546abde5d6341feabb85ed957dc9dfb`

### 마이그레이션
- ✅ 001_initial_schema: sites, posts, comments 테이블 생성
- ✅ 002_change_api_key_to_varchar: api_key UUID → VARCHAR(100) 변경

---

## 작업 이력

### [2025-10-22] Supabase 배포 완료
- Session Pooler 사용 (IPv4 호환)
- api_key 타입 변경 (UUID → VARCHAR)
- Prefixed API key 적용 (orb_live_)
- 프로덕션 사이트 등록 완료

### [2025-10-22] CORS 전략 반영 및 문서 간소화
- 동적 CORS 적용 (사이트별 검증)
- 글로벌 CORS 환경변수 제거
- 중복 제거 및 간결화 (630줄 → 150줄)

### [2025-10-22] Railway → Render + Supabase로 전면 수정
- 무료 영구 플랫폼 조사 및 선택
- 단계별 배포 가이드 작성

### [2025-10-14] 초안 작성
- Railway 기준 작성

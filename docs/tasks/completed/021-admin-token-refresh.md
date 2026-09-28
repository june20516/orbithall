# 어드민 토큰 갱신 (Refresh Token)

## 작성일
2026-09-28

## 우선순위
- [x] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
어드민 로그인에 회전되는 Refresh Token을 도입해 재로그인 없이 세션을 연장하고, 서버에서 세션을 폐기할 수 있게 한다.

## 작업 범위
### 포함
- `refresh_tokens` 테이블, `POST /auth/refresh`, `POST /auth/logout`
- 로그인 응답을 토큰 쌍 필드로 교체 (`token` 필드 제거)
- Access Token에 `typ`/`iss`/`aud`/`jti` 클레임 추가, `typ`·`iss`·`aud` 검증 필수 (기존 토큰은 배포 즉시 무효, 재로그인)
- JWT 미들웨어와 `/auth/google/verify`의 에러 본문을 객체 형식으로 통일

### 제외
- 프론트엔드(orbithall-admin) 변경
- `/admin/*` 핸들러 본문의 평문 에러 통일

## 주요 결정사항
- 회전은 단일 SQL 문장(조건부 UPDATE + INSERT): 트랜잭션 주입 구조에서도 원자성 보장
- 후속 토큰 = HMAC(REFRESH_TOKEN_SECRET, 이전 토큰): 원문 저장 없이 유예 시간 재반환
- rate limit은 계열(family) 기준: 모든 요청이 어드민 서버 IP 하나에서 옴
- 오래된 토큰 정리는 로그인 시: 스케줄러 없이 사용자별로 정리

## 의존성
- 선행: 009, 010
- 후속: 023 (보강 과제), orbithall-admin 갱신 구현

## 배포 전 필수
- Render 환경변수 `REFRESH_TOKEN_SECRET` 설정 (32자 이상, `JWT_SECRET`과 다른 값)
- 배포 즉시 기존 Access Token이 무효가 되어 로그인 사용자 전원이 재로그인해야 함 (사용자 공지)

---

## 작업 이력
### [2026-09-28] 작업 시작

### [2026-09-28] 구현 완료
- refresh_tokens 테이블(006), `/auth/refresh`, `/auth/logout` 추가
- 로그인 응답에 토큰 쌍 추가, 인증 에러 본문 객체 형식 통일
- Access Token 클레임(typ·iss·aud·exp) 필수, HS256 고정, JWT_SECRET·REFRESH_TOKEN_SECRET 시작 시 검증
- 리뷰 반영: 계열 폐기와 회전 경합 차단(NOT EXISTS, 유예 재반환 전 계열 폐기 확인), 회전 전 사용자 조회·토큰 발급, 동시성 테스트
- 호환용 `token` 필드 제거, 백엔드·클라이언트 함께 배포로 전환 계획 통합
- 후속: p9-023 (보강 과제), orbithall-admin 갱신 구현

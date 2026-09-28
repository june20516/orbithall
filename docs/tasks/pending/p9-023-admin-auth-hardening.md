# 어드민 인증 코드 리뷰 후속 보강

## 작성일
2026-09-28

## 우선순위
- [ ] 높음
- [x] 보통
- [ ] 낮음

## 작업 개요
어드민 토큰 갱신 구현(`docs/specs/admin-auth-token-refresh.md`) 코드 리뷰에서 나온 후속 보강 항목을 모은다. 기능 동작에는 영향이 없지만 방치하면 남용·경쟁 조건·운영 문제로 이어질 수 있는 항목들이다.

## 작업 범위

### 포함
- **로그인 요청 제한 없음**: `/auth/google/verify` 호출마다 `refresh_tokens` 행이 새로 쌓이는데 요청 자체에는 제한이 없다. 사용자당 활성 계열(family) 수 제한 또는 `google_id` 기준 요청 제한을 검토한다. BFF(Next.js 서버) 한 곳에서만 요청이 오므로 IP 기준 제한은 부적합하다.
- **첫 로그인 사용자 생성 경쟁 조건**: `internal/handlers/auth.go`의 `GoogleVerify`가 `GetUserByGoogleID` 조회 후 없으면 `CreateUser`로 INSERT하는 구조라, 같은 `google_id`로 동시에 두 요청이 오면 한쪽이 UNIQUE 제약 위반으로 500을 받는다. `internal/database/posts.go`의 `GetOrCreatePost`처럼 `ON CONFLICT DO NOTHING` + 재조회 패턴으로 바꾼다.
- **검증되지 않은 이메일 저장**: 사용자 생성 시 Google이 검증한 `payload.Email`이 아니라 요청 본문의 `req.Email`을 저장한다(`auth.go` 128행). 클라이언트가 임의의 이메일을 보낼 수 있으므로 `payload.Email`을 우선 사용하도록 고친다.
- **`/admin/*` 에러 응답 형식 불일치**: `internal/handlers/admin.go`의 핸들러들이 `http.Error`로 평문 에러를 반환해, 명세가 정한 객체 형식(`{"error":{"code","message"}}`)과 다르다. 응답을 통일하고, 핸들러가 직접 내는 400·403·404·500의 Swagger 주석(`{string} string`)도 통일된 형식에 맞춰 갱신한다. (JWT 미들웨어가 내는 401은 이미 객체 형식이며 Swagger에도 `ErrorResponse`로 반영됨)
- **rate limiter 키 누적**: 계열(family) 기준 rate limiter(`internal/ratelimit`)가 `sync.Map`에 키를 계속 쌓아 프로세스 수명 동안 해제되지 않는다. 기존 IP 기준 limiter와 같은 한계이며, 인스턴스를 늘리면 제한이 인스턴스별로 나뉘어 실질 한도가 배수로 늘어난다.

### 제외
- 신규 인증 방식 도입(비대칭 서명, DPoP 등) — 명세 9장 범위 외 항목과 동일
- "모든 기기에서 로그아웃", 활성 세션 목록 조회 — 별도 기능 작업

## 의존성
- 선행: 021

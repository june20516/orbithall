# 어드민 토큰 전환 3단계 (호환 코드 제거)

## 작성일
2026-09-28

## 우선순위
- [x] 높음

## 작업 개요
orbithall-admin이 `access_token`을 읽도록 바뀐 뒤, 로그인 응답의 호환용 `token` 필드를 제거한다. 명세 `docs/specs/admin-auth-token-refresh.md` 7장 3단계다.

## 작업 범위
### 포함
- 로그인 응답의 `token` 필드 제거 (`GoogleVerifyResponse.Token`)
- README·명세 갱신

### 제외
- `/admin/*` 핸들러 본문의 평문 에러 통일 (별도 작업)

## 착수 조건
- orbithall-admin이 `token` 대신 `access_token`을 읽도록 배포 완료

## 의존성
- 선행: 021, orbithall-admin 토큰 갱신 구현

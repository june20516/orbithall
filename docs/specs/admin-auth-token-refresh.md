# 어드민 인증 토큰 갱신 명세서

## 작성일
2026-09-28

## 버전
v1.8 (동시 로그인 직렬화, 로그인 요청 필드 설명 정리, 사이트 검증 에러 객체 형식 통일)

## 개요
어드민 로그인 세션에 **회전(rotation)되는 Refresh Token**을 도입해, Access Token이 만료되면 재로그인 없이 연장할 수 있게 한다.
Google은 최초 로그인 시 신원 확인(ID Token)에만 쓰고, 이후 세션 연장은 Orbithall 백엔드가 자체 발급한 Refresh Token으로 처리한다.
OAuth 2.0 Refresh Token 규격(RFC 6749 §6)과 보안 권고(RFC 9700, OAuth 2.0 Security BCP)를 따른다.

## 목적 및 배경
- 현재 백엔드 JWT는 유효기간이 7일이고 갱신 수단이 없다. 만료되면 Google 재로그인 외에 방법이 없다.
- Refresh Token을 DB에서 관리해 **자동 세션 연장**과 **서버 측 세션 폐기**를 함께 얻는다.
- Access Token 수명은 현재와 같은 7일로 유지한다. 그래서 세션을 폐기해도 이미 발급된 Access Token은 최대 7일간 유효하다(2.3, 3.3 참고).

## 사용자 스토리
```
AS A 사이트 관리자
I WANT 로그인한 뒤 작업하는 동안 다시 로그인하라는 요구를 받지 않고
SO THAT 끊김 없이 사이트와 댓글을 관리할 수 있다

AS A 사이트 관리자
I WANT 로그아웃하면 세션이 서버에서도 즉시 무효화되어
SO THAT 토큰이 유출되어도 계속 악용되지 않는다
```

---

## 1. 전체 구조

```
[브라우저] ──(Auth.js 세션 쿠키, HttpOnly/암호화)──> [orbithall-admin: Next.js 서버]
                                                          │
                                                          │ Authorization: Bearer <access_token>
                                                          │ POST /auth/refresh {refresh_token}
                                                          ▼
                                                 [orbithall API (Go)]
```

- 백엔드 토큰(Access/Refresh)은 **브라우저 JS에 절대 노출하지 않는다.** Auth.js의 암호화된 JWT 쿠키 안에만 저장한다(현재 방식 유지).
- 백엔드 호출은 Next.js 서버(Server Action, Route Handler, proxy)에서만 한다.
- 그래서 Refresh Token은 **응답 본문(JSON)으로 전달**한다. 백엔드는 인증용 쿠키를 설정하지 않으며, CORS 설정도 바꾸지 않는다.

### 토큰 종류

| 구분 | 형식 | 유효기간(기본값) | 용도 | 저장 위치(클라이언트) |
|------|------|------------------|------|----------------------|
| Access Token | JWT (HS256) | 7일 | `/admin/*` API 호출 | Auth.js JWT 쿠키 |
| Refresh Token | 불투명 랜덤 문자열 | 14일 (사용 시 연장), 최대 30일 | Access Token 재발급 | Auth.js JWT 쿠키 |

- **Refresh Token은 1회용이다.** 사용할 때마다 새 Refresh Token이 발급되고, 이전 토큰은 무효가 된다(rotation).
- 14일간 한 번도 쓰지 않으면 만료된다(유휴 만료).
- 계속 쓰더라도 최초 로그인 후 30일이 지나면 만료되고, Google 재로그인이 필요하다(절대 만료).
- 클라이언트는 Refresh Token의 내부 구조를 해석하지 않는다(불투명 값). 만료 시각은 응답 필드로 받는다.

---

## 2. API 명세

모든 에러 응답은 위젯 API(`respondError`)에서 쓰는 **객체 형식**으로 통일한다.
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "사람이 읽을 수 있는 설명"
  }
}
```
- `code`: 분기용 상수. 클라이언트는 이 값만 해석한다.
- `message`: 디버깅용. 사용자에게 그대로 보여주지 않는다.
- `details`: 선택 필드. `/admin/sites` 생성·수정의 입력 검증 실패(400 `INVALID_INPUT`, `message`는 `"Validation failed"`)에서만 필드별 검증 메시지를 담는다(예: `{"name":"Name is required"}`). 그 밖의 에러에서는 쓰지 않는다.

**이번 작업에서 형식이 바뀌는 곳** (현재 클라이언트는 상태 코드만 보므로 호환된다)

| 위치 | 현재 | 변경 후 |
|------|------|---------|
| JWT 미들웨어 (`/admin/*`의 401) | `{"error":"CODE","message":"..."}` | 객체 형식 |
| `POST /auth/google/verify` | 평문(`text/plain`) | 객체 형식 |
| `POST /auth/refresh`, `/auth/logout` | - | 객체 형식 |
| `/admin/*` 핸들러 본문 | 평문(`text/plain`) | 객체 형식 |
| `/admin/sites` 생성·수정 검증 실패 | `{"error":"<검증 메시지>"}` | 객체 형식(`details`에 필드별 메시지) |

`/admin/*` 핸들러 본문의 에러도 같은 객체 형식이다(코드: INVALID_INPUT, UNAUTHORIZED, FORBIDDEN, SITE_NOT_FOUND, POST_NOT_FOUND, COMMENT_NOT_FOUND, INTERNAL_SERVER_ERROR).

### 2.1 로그인 (변경)
```
POST /auth/google/verify
Content-Type: application/json
```

**요청** (필드 구성은 기존과 같고, `id_token` 외에는 선택)
```json
{
  "id_token": "<Google ID Token>",
  "email": "user@example.com",
  "name": "홍길동",
  "picture": "https://..."
}
```
- `id_token`만 필수다. `email`·`name`·`picture`는 선택이다.
- `email`은 사용하지 않는다(항상 ID Token의 검증된 이메일을 저장한다). `name`·`picture`는 ID Token에 없을 때만 사용한다. 이름은 100자를 넘으면 앞 100자로 잘라 저장한다.
- 검증되지 않은 이메일의 ID Token은 401 `INVALID_ID_TOKEN`이다.

**응답 200**
```json
{
  "token_type": "Bearer",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "access_token_expires_at": "2026-10-01T12:15:00Z",
  "refresh_token": "ohrt_Q2x1ZGUgcmVmcmVzaCB0b2tlbiBleGFtcGxl...",
  "refresh_token_expires_at": "2026-10-15T12:00:00Z",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "홍길동",
    "picture_url": "https://...",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

| 필드 | 설명 |
|------|------|
| `token_type` | 항상 `"Bearer"` |
| `access_token` | `/admin/*` 호출용 JWT |
| `access_token_expires_at` | Access Token 만료 시각 (RFC 3339, UTC) |
| `refresh_token` | 갱신용 토큰. `ohrt_` 접두사 + base64url |
| `refresh_token_expires_at` | Refresh Token 만료 시각 (유휴·절대 만료 중 빠른 쪽) |
| `user` | 기존과 동일 |

**에러** (상태 코드는 유지하고 본문만 객체 형식으로 변경)

| 상태 | `code` | 의미 |
|------|--------|------|
| 400 | `INVALID_INPUT` | JSON 오류, 필수 필드 누락, Content-Type 오류 |
| 401 | `INVALID_ID_TOKEN` | Google ID Token 검증 실패 (이메일 미검증 포함) |
| 500 | `INTERNAL_SERVER_ERROR` | 서버 오류 |

### 2.2 토큰 갱신 (신규)
```
POST /auth/refresh
Content-Type: application/json
```

**요청**
```json
{ "refresh_token": "ohrt_..." }
```

**응답 200**: 로그인 응답에서 `user`를 뺀 형태
```json
{
  "token_type": "Bearer",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "access_token_expires_at": "2026-10-01T12:30:00Z",
  "refresh_token": "ohrt_새로운값...",
  "refresh_token_expires_at": "2026-10-15T12:15:00Z"
}
```

- 응답에는 `Cache-Control: no-store` 헤더가 붙는다(로그인 응답도 동일). 클라이언트도 이 응답을 캐시하거나 로그에 남기지 않는다.
- 클라이언트는 응답의 `refresh_token`으로 기존 값을 **항상 교체**해 저장한다.
- 보통은 새 값이다. 다만 유예 시간 안에 같은 토큰으로 다시 요청하면, 앞서 발급한 값과 **같은** `refresh_token`을 돌려준다(3.2). `access_token`은 매번 새로 발급한다.

**에러**

| 상태 | `code` | 의미 | 클라이언트 처리 |
|------|---------|------|----------------|
| 400 | `INVALID_INPUT` | JSON 오류, `refresh_token` 누락 | 버그. 로그 후 재로그인 |
| 401 | `INVALID_REFRESH_TOKEN` | 존재하지 않거나 폐기된 토큰 | 재로그인 |
| 401 | `REFRESH_TOKEN_EXPIRED` | 유휴 또는 절대 만료 | 재로그인 |
| 401 | `REFRESH_TOKEN_REUSED` | 이미 사용된 토큰이 유예 시간 이후 재사용됨 → **해당 세션 전체 폐기** | 재로그인 |
| 401 | `USER_NOT_FOUND` | 사용자 삭제됨 | 재로그인 |
| 429 | `RATE_LIMIT_EXCEEDED` | 과도한 갱신 요청 | `Retry-After` 이후 재시도 |
| 500 | `INTERNAL_SERVER_ERROR` | 서버 오류 | 기존 세션 유지, **30초 안에** 같은 토큰으로 한 번 재시도 |

> 클라이언트는 **401이면 코드와 관계없이 재로그인**으로 처리하면 된다. 코드 구분은 로깅·디버깅용이다.
> 500이나 네트워크 오류는 Refresh Token이 소비되지 않았을 수 있으므로 **세션을 지우지 않는다.**
> 서버에서는 회전됐지만 응답을 받지 못한 경우는 유예 시간(30초) 안의 재시도로만 복구된다. 30초가 지난 뒤 같은 토큰을 보내면 재사용으로 판정되어 세션이 폐기된다.

### 2.3 로그아웃 (신규)
```
POST /auth/logout
Content-Type: application/json
```

**요청**
```json
{ "refresh_token": "ohrt_..." }
```

**응답 204** (본문 없음)

- 해당 Refresh Token이 속한 **세션(토큰 계열) 전체를 폐기**한다.
- 이미 폐기되었거나, 만료되었거나, 존재하지 않는 토큰이어도 **204**를 반환한다(멱등). 토큰이 존재하는지 알려주지 않기 위해서다.
- 이미 발급된 Access Token은 만료될 때까지(**최대 7일**) 유효하다. Access Token은 서버에 상태를 두지 않기 때문이다. 클라이언트는 로그아웃 시 저장된 토큰을 모두 지워야 한다.

### 2.4 `/admin/*` 인증 (변경 없음 + 보강)
- 헤더: `Authorization: Bearer <access_token>`
- 기존 에러 코드를 유지하되, 본문은 객체 형식(`error.code`)으로 바뀐다: `MISSING_TOKEN`, `INVALID_TOKEN`, `EXPIRED_TOKEN`, `USER_NOT_FOUND`
- `UNAUTHORIZED`: 미들웨어를 통과했지만 컨텍스트에 사용자가 없을 때 핸들러가 반환하는 방어 코드다(정상 흐름에서는 발생하지 않는다).
- Access Token에 `typ: "access"`, `iss: "orbithall"`, `aud: "orbithall-admin"`, `jti` 클레임을 추가한다. 백엔드는 `typ`·`iss`·`aud` 중 하나라도 없거나 다르면 `INVALID_TOKEN`으로 거부한다.
  - 이 클레임이 없는 기존 토큰은 배포 즉시 거부된다. 기존 로그인 사용자는 한 번 다시 로그인해야 한다(7장).
  - `exp`도 필수이며, 서명 알고리즘은 HS256만 받는다.
  - 서명이 유효하고 만료된 토큰은 다른 클레임 오류와 관계없이 `EXPIRED_TOKEN`으로 응답한다. 클라이언트의 갱신 가능 여부는 Refresh Token으로만 판단되므로, 만료를 먼저 알려 갱신을 시도하게 한다.

---

## 3. 세션 수명과 회전 규칙

### 3.1 Refresh Token Rotation
1. 클라이언트가 Refresh Token `R1`으로 `/auth/refresh`를 호출한다.
2. 서버가 `R1`을 "사용됨"으로 표시하고 `R2`와 새 Access Token을 발급한다.
3. 이후 `R2`만 유효하다.

### 3.2 재사용 탐지와 유예 시간
- 이미 사용된 토큰이 다시 제출되면 탈취로 간주하고, **같은 로그인에서 파생된 토큰 전체(family)를 폐기**한다 → `REFRESH_TOKEN_REUSED`
- **예외(유예 시간 30초)**: `R1`이 사용된 지 30초가 지나지 않았고 그 후속 토큰 `R2`가 아직 사용되지 않았다면, **이미 발급한 `R2`를 그대로** 돌려준다. `access_token`만 새로 발급한다.
  - 목적: 동시 요청 두 개가 같은 토큰으로 갱신하거나, 갱신 응답을 받은 뒤 쿠키 저장에 실패하는 경우를 흡수한다.
  - 새 갈래를 만들지 않으므로 계열이 항상 한 줄로 유지된다. 공격자가 유예 시간 안에 `R1`을 재사용해 `R2`를 얻더라도, 정상 사용자와 공격자 중 나중에 `R2`를 쓰는 쪽에서 재사용 탐지가 걸린다.
  - `R2`가 이미 사용되었다면(→ `R3` 발급됨) 유예 시간 안이라도 재사용으로 처리한다.
  - 유예 시간 안이지만 `R2`가 만료되었으면 `REFRESH_TOKEN_EXPIRED`, 폐기되었으면 `INVALID_REFRESH_TOKEN`으로 응답하고 계열은 폐기하지 않는다. 비밀키 교체로 `R2`를 다시 계산할 수 없는 경우도 `INVALID_REFRESH_TOKEN`이다. 이들은 탈취 신호가 아니기 때문이다.
  - 유예 시간은 안전장치일 뿐이다. 클라이언트는 4장의 규칙을 지켜 재사용 자체가 생기지 않게 해야 한다.

```
R1 ──사용──> R2 ──사용──> R3
 │
 └─ 30초 안에 R1 재제출 → R2 재반환 (R2 미사용일 때만)
 └─ 30초 후 R1 재제출    → REFRESH_TOKEN_REUSED, 계열 전체 폐기
```

### 3.3 서버 측 폐기 트리거
> 폐기는 Refresh Token에만 적용된다. 이미 발급된 Access Token은 만료(최대 7일)까지 유효하다.

| 트리거 | 폐기 범위 |
|--------|-----------|
| `/auth/logout` | 해당 family |
| 재사용 탐지 | 해당 family |
| 사용자 삭제 | 해당 사용자 전체 |
| (추후) "모든 기기에서 로그아웃" | 해당 사용자 전체 |

---

## 4. 클라이언트(orbithall-admin) 구현 가이드

### 4.1 저장
- Auth.js `jwt` 콜백의 `token`에 다음을 저장한다(기존 `backendToken`을 대체).
  ```ts
  interface JWT {
    backendAccessToken?: string;
    backendAccessTokenExpiresAt?: string;   // RFC 3339
    backendRefreshToken?: string;
    backendRefreshTokenExpiresAt?: string;
    backendUser?: GoogleVerifyResponse["user"];
    backendAuthError?: "RefreshFailed";
  }
  ```
- `session` 콜백으로 토큰을 브라우저에 노출하지 않는다(현재 원칙 유지).
- `backendAuthError`만 세션에 노출해, UI가 재로그인을 유도하는 데 쓸 수 있다.

### 4.2 갱신 시점
- **선제 갱신**: `access_token_expires_at`까지 **60초 미만** 남았으면 API를 호출하기 전에 갱신한다.
  - 만료 시각은 JWT를 디코딩하지 말고 응답 필드 값을 쓴다.
- **사후 갱신**: `/admin/*`이 `401 EXPIRED_TOKEN`을 반환하면 **한 번만** 갱신한 뒤 재시도한다. 재시도도 401이면 재로그인한다.

### 4.3 갱신 위치 제약 (중요)
- 회전된 새 Refresh Token은 **반드시 쿠키에 저장되어야 한다.** 저장하지 못하면 다음 요청이 옛 토큰을 보내고, 유예 시간이 지나면 재사용 탐지에 걸려 세션이 폐기된다.
- **Server Component 렌더링 중에는 쿠키를 쓸 수 없다.** 그러므로 갱신은 쿠키를 쓸 수 있는 곳에서만 한다.
  - proxy(구 middleware), Route Handler, Server Action
- 권장 방식: proxy에서 요청마다 만료가 임박했는지 확인하고, 필요하면 갱신한 뒤 새 세션 쿠키를 응답에 싣는다. 같은 요청의 하위 렌더링에서도 새 토큰이 보이도록 처리한다.
- 같은 사용자의 요청이 동시에 갱신을 시도할 수 있다. 가능하면 한 번만 갱신하도록 막는다(유예 시간이 최후 안전장치다).

### 4.4 로그인 / 로그아웃
- 로그인: 기존 `jwt` 콜백 흐름을 유지하고, 응답의 새 필드를 저장한다.
- 로그아웃: Auth.js `events.signOut`에서 `POST /auth/logout`을 호출한다. 호출에 실패하더라도 로컬 세션은 삭제한다.
- 갱신이 401로 실패하면 저장된 백엔드 토큰을 지우고 `backendAuthError = "RefreshFailed"`로 표시한 뒤 `/login`으로 보낸다.

### 4.5 로깅
- `refresh_token`, `access_token` 필드는 로그에서 반드시 마스킹한다.
- `lib/utils/redact.ts`에 `access_token`, `refresh_token`은 이미 있다. 새 필드명(`backendAccessToken` 등)을 쓴다면 추가한다.

### 4.6 시퀀스
```
[로그인]
Browser → Next: Google 로그인
Next → API: POST /auth/google/verify {id_token}
API → Next: {access_token(7d), refresh_token(14d)}
Next → Browser: Set-Cookie authjs.session-token (암호화)

[API 호출 - 정상]
Next → API: GET /admin/sites (Bearer access)

[API 호출 - 만료 임박]
proxy: expires_at - now < 60s
Next → API: POST /auth/refresh {R1}
API → Next: {access', R2}
Next → Browser: Set-Cookie (R2로 교체)
Next → API: GET /admin/sites (Bearer access')

[갱신 실패]
API → Next: 401 REFRESH_TOKEN_*
Next: 토큰 삭제 → /login 리다이렉트
```

---

## 5. 보안 요구사항

### 백엔드
- **Refresh Token 생성**: 값은 base64url로 인코딩하고 `ohrt_` 접두사를 붙인다. 접두사는 유출 스캐닝을 쉽게 하기 위한 것이다.
  - 로그인 시 첫 토큰: `crypto/rand` 32바이트
  - 회전 시 후속 토큰: `HMAC-SHA256(REFRESH_TOKEN_SECRET, 이전 토큰 원문)`
    - 같은 이전 토큰에서 항상 같은 후속 토큰이 나온다. 그래서 원문을 저장하지 않고도 유예 시간 안에 `R2`를 재반환할 수 있다(3.2).
    - 비밀키 없이는 `R1`에서 `R2`를 예측할 수 없다.
- **저장**: DB에는 **SHA-256 해시만** 저장한다. 평문은 저장하지 않고 로그에도 남기지 않는다.
- **비밀키 분리**: `REFRESH_TOKEN_SECRET`은 `JWT_SECRET`과 다른 값(32자 이상)을 쓴다. 이 키를 교체하면 유예 시간 중 재반환만 영향을 받는다(기존 토큰 검증은 해시 비교라 영향 없음).
  - 이 키가 유출되면 탈취한 과거 토큰 하나로 이후 후속 토큰을 모두 계산할 수 있다. 유출이 의심되면 즉시 교체하고, 필요하면 해당 사용자의 세션을 폐기한다.
- **조회와 회전**: 해시로 조회한다. 회전은 "아직 사용되지 않았고 폐기되지 않은 경우에만 사용 처리하고, 후속 토큰을 삽입"하는 **SQL 한 문장**(조건부 UPDATE + INSERT)으로 처리한다. 동시에 회전 요청이 와도 한 요청만 성공하고, 나머지는 유예 시간 규칙(3.2)으로 판단한다.
  - 같은 계열에 폐기된 토큰이 하나라도 있으면 회전하지 않고, 유예 시간 재반환도 하지 않는다. 계열 폐기 UPDATE는 문장 시작 시점에 보이는 행만 폐기하므로, 동시에 진행된 회전이 만든 토큰은 폐기 표시 없이 남을 수 있기 때문이다.
- **Access Token**: HS256 서명 알고리즘을 고정한다(현재 유지). `typ=access`, `iss`, `aud`, `exp`, `iat`, `jti` 클레임을 포함한다.
- **Rate limit**: `/auth/refresh`에 적용한다. `/auth/logout`은 멱등이고 실패해도 클라이언트가 무시하므로 제외한다.
  - 모든 요청이 Next 서버 IP 하나에서 오므로 **IP 기준 제한은 쓰지 않는다.** Refresh Token 해시 또는 family 기준으로 제한한다.
  - 기준치: family당 분당 10회
- **만료 토큰 정리**: 로그인할 때 그 사용자의 토큰 중 절대 만료 시각 또는 폐기 시각으로부터 7일이 지난 행을 삭제한다. 별도 스케줄러는 두지 않는다.
- **세션 수 상한**: 사용자당 활성 세션(폐기·만료되지 않은 계열)은 최근 10개까지 유지하고, 초과하면 가장 오래된 활성 세션을 삭제한다(해당 Refresh Token은 `INVALID_REFRESH_TOKEN`). 폐기된 계열이나 이미 만료된 계열은 이 상한 계산에 포함하지 않으며, 삭제 대상도 아니다(만료 토큰 정리 규칙으로 별도 정리됨). 로그인할 때 새 계열을 만든 뒤 적용하며, 최근 여부는 활성 계열에서 가장 나중에 저장된 토큰 기준이다. 로그인은 사용자 단위로 직렬화되어 상한을 넘지 않는다. 상한 초과로 삭제되는 가장 오래된 세션이 동시에 회전 중이면 그 세션은 곧바로 무효가 될 수 있다.
- 전송은 HTTPS로만 한다(Render 기본).

### 클라이언트
- 백엔드 토큰을 `localStorage`, `sessionStorage`, 클라이언트 컴포넌트 props, `/api/auth/session` 응답에 두지 않는다.
- Auth.js 세션 쿠키는 `HttpOnly`, `Secure`, `SameSite=Lax`를 유지한다(Auth.js 기본값).
- `AUTH_SECRET`이 바뀌면 모든 세션이 무효화되는 것을 전제로 운영한다.

---

## 6. 데이터 모델 (백엔드 내부, 참고용)

```sql
-- migrations/006_create_refresh_tokens_table.up.sql 과 동일
CREATE TABLE refresh_tokens (
    id                 BIGSERIAL   PRIMARY KEY,
    user_id            BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id          UUID        NOT NULL,              -- 최초 로그인 1회 = 1 family
    parent_id          BIGINT      REFERENCES refresh_tokens(id) ON DELETE CASCADE, -- 이 토큰을 회전해 만든 이전 토큰
    token_hash         BYTEA       NOT NULL UNIQUE,       -- SHA-256(refresh_token)
    expires_at         TIMESTAMPTZ NOT NULL,              -- 유휴 만료 (발급 + 14일, 절대 만료를 넘지 않음)
    family_expires_at  TIMESTAMPTZ NOT NULL,              -- 절대 만료 (최초 로그인 + 30일)
    used_at            TIMESTAMPTZ,                       -- 회전에 사용된 시각
    revoked_at         TIMESTAMPTZ,
    revoked_reason     VARCHAR(30),                       -- logout | reuse_detected (사용자 삭제 시에는 행이 CASCADE로 삭제됨)
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_refresh_tokens_expiry CHECK (expires_at <= family_expires_at)
);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_user_id   ON refresh_tokens(user_id);
-- 부모 하나에 자식은 하나뿐이다 (계열이 갈라지지 않음을 DB가 보장)
CREATE UNIQUE INDEX idx_refresh_tokens_parent_id ON refresh_tokens(parent_id) WHERE parent_id IS NOT NULL;
```

### 환경변수
| 변수 | 기본값 | 설명 |
|------|--------|------|
| `JWT_EXPIRATION_HOURS` | `168` (7일) | Access Token 수명 (기존 변수 유지) |
| `REFRESH_TOKEN_IDLE_TTL` | `336h` (14일) | Refresh Token 유휴 만료 |
| `REFRESH_TOKEN_ABSOLUTE_TTL` | `720h` (30일) | 세션 절대 만료 |
| `REFRESH_TOKEN_REUSE_GRACE` | `30s` | 재사용 유예 시간 |
| `REFRESH_TOKEN_SECRET` | (필수) | 후속 토큰 파생용 HMAC 키, 32자 이상 |

---

## 7. 배포 (호환성)

호환 기간 없이 백엔드와 클라이언트를 **함께 배포**한다. 사용자가 적어 짧은 중단과 재로그인을 감수한다.

| 대상 | 변경 |
|------|------|
| 백엔드 | 토큰 쌍 응답, `/auth/refresh`·`/auth/logout`, 에러 본문 객체 형식, Access Token 클레임 검증 필수 |
| 클라이언트 | 로그인 응답의 `access_token`·`refresh_token` 저장(기존 `token` 필드는 없음), 갱신·로그아웃 구현, `error.code` 기반 사후 갱신 |

- 로그인 응답에 기존 `token` 필드가 없다. 클라이언트가 바뀌기 전에 백엔드가 먼저 배포되면, 그 사이에는 로그인이 실패한다.
- 기존 Access Token은 클레임이 없어 배포 즉시 `INVALID_TOKEN`으로 거부된다. 로그인해 있던 사용자는 다시 로그인해야 한다.
- 이메일이 검증되지 않은(`email_verified`가 true가 아닌) Google 계정은 배포 후 로그인이 401 `INVALID_ID_TOKEN`으로 거부된다(이전에는 로그인할 수 있었다).

---

## 8. 테스트 시나리오

### 정상
1. 로그인하면 네 가지 토큰 필드와 `user`가 반환된다.
2. `R1`로 갱신하면 새 Access Token과 `R2`가 발급되고, `R2`로 다시 갱신할 수 있다.
3. 로그아웃 후 해당 Refresh Token으로 갱신하면 401 `INVALID_REFRESH_TOKEN`이 반환된다.

### 예외
1. `R1`이 사용된 뒤 30초가 지나 다시 `R1`을 제출하면 401 `REFRESH_TOKEN_REUSED`가 반환되고, `R2`도 무효가 된다.
2. 14일간 사용하지 않은 토큰은 401 `REFRESH_TOKEN_EXPIRED`
3. 계속 갱신하더라도 최초 로그인 후 30일이 지나면 401 `REFRESH_TOKEN_EXPIRED`
4. 사용자가 삭제되면 401이 반환된다.
5. `/admin/*`에 Refresh Token을 Bearer로 보내면 401 `INVALID_TOKEN`이 반환된다.

### 엣지 케이스
1. 같은 `R1`으로 동시에 두 번 갱신 요청이 오면 둘 다 200이고, 두 응답의 `refresh_token`은 같은 `R2`다.
2. `R1` → `R2` → `R3`로 회전한 뒤 30초 안에 `R1`을 다시 제출하면 401 `REFRESH_TOKEN_REUSED`가 반환된다.
3. 갱신 중 500이 반환되면 클라이언트는 세션을 유지하고 재시도한다.
4. 존재하지 않는 토큰으로 로그아웃하면 204가 반환된다.
5. `/admin/*`에 만료된 Access Token을 보내면 `{"error":{"code":"EXPIRED_TOKEN",...}}`이 반환된다.

---

## 9. 범위 외 (추후)
- "모든 기기에서 로그아웃" UI/API
- 활성 세션 목록 조회
- 비대칭 서명(RS256/EdDSA)과 JWKS 공개
- 발신자 제한 토큰(DPoP)

## 참고 자료
- RFC 6749 §6 Refreshing an Access Token
- RFC 9700 Best Current Practice for OAuth 2.0 Security (§4.14 Refresh Token Protection)
- Auth.js Refresh Token Rotation 가이드
- `docs/tasks/completed/009-admin-auth-system.md`, `010-admin-jwt-middleware.md`

## 변경 이력
| 날짜 | 버전 | 변경 내용 | 작성자 |
|------|------|-----------|--------|
| 2026-09-28 | v1.0 | 초안 작성 | Bran |
| 2026-09-28 | v1.1 | Access Token 7일로 변경, 에러 본문 객체 형식 통일, 유예 시간 재사용 시 기존 R2 재반환 | Bran |
| 2026-09-28 | v1.2 | 회전을 단일 SQL 문장으로, `replaced_by_id` → `parent_id`, rate limit은 refresh만, 로그인 시 정리, 클레임 필수화는 3단계, 에러 코드를 기존 상수(INVALID_INPUT·INTERNAL_SERVER_ERROR·RATE_LIMIT_EXCEEDED)에 맞춤 | Bran |
| 2026-09-28 | v1.3 | Access Token 클레임(`typ`·`iss`·`aud`) 검증을 1단계에서 바로 필수로 적용 | Bran |
| 2026-09-28 | v1.4 | 호환용 `token` 필드 제거, 전환 단계를 함께 배포 한 번으로 통합 | Bran |
| 2026-09-28 | v1.5 | 로그인 시 ID Token의 검증된 이메일·이름·사진 우선 사용(요청 본문 `email`·`name` 선택), 이메일 미검증 토큰 거부, 첫 로그인 동시 요청 시 사용자 생성 경합 제거, 사용자당 세션 10개 상한 | Bran |
| 2026-09-28 | v1.6 | 세션 수 상한 계산에서 폐기·만료된 계열 제외(활성 계열만 상한 대상) | Bran |
| 2026-09-28 | v1.7 | `/admin/*` 핸들러 본문의 평문 에러(`http.Error`)를 객체 형식으로 통일 | Bran |
| 2026-09-28 | v1.8 | 로그인을 사용자 단위로 직렬화해 동시 로그인에도 세션 상한 유지, 로그인 요청 `email`은 사용하지 않고 `name`·`picture`는 ID Token에 없을 때만 사용함을 명확히 하고 이메일 미검증 계정 거부를 배포 행동 변경으로 기록, `/admin/sites` 생성·수정 검증 실패를 객체 형식(`INVALID_INPUT`, `details`)으로 통일 | Bran |

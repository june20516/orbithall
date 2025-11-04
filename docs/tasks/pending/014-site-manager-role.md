# 사이트 매니저 역할 추가

## 작성일
2025-11-04

## 우선순위
- [ ] 높음
- [x] 보통
- [ ] 낮음

## 작업 개요
사이트에 manager 역할을 추가하여 owner가 다른 사용자에게 조회 권한 부여 가능

## 작업 범위

### 포함
- **Manager 추가 API**
  - `POST /admin/sites/:id/managers` - owner만 호출 가능
  - 요청: `{ "email": "user@example.com" }`
  - 응답: 추가된 manager 정보

- **Manager 목록 조회 API**
  - `GET /admin/sites/:id/managers` - owner만 호출 가능

- **Manager 삭제 API**
  - `DELETE /admin/sites/:id/managers/:user_id` - owner만 호출 가능

- **권한 로직 수정**
  - `HasUserSiteAccess()`: owner 또는 manager 확인하도록 변경
  - 조회(GET): owner, manager 모두 가능
  - 수정/삭제(PUT/DELETE): owner만 가능

- **네이밍 정리**
  - `HasUserSiteAccess()`: 조회 권한 확인 (owner + manager)
  - `IsUserSiteOwner()` 새로 추가: owner 여부만 확인
  - 수정/삭제 API에서는 `IsUserSiteOwner()` 사용

### 제외
- manager의 수정/삭제 권한 (추후 확장 가능)
- 역할별 세분화된 권한 (viewer, editor 등)

## 예상 시간
2시간

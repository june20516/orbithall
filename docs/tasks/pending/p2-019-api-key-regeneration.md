# 사이트 API Key 재발급

## 작성일
2026-09-22

## 우선순위
- [ ] 높음
- [ ] 보통
- [x] 낮음

## 작업 개요
API Key가 유출되었을 때 사이트 소유자가 새 키로 교체할 수 있도록 재발급 API를 추가합니다.
어드민 대응 과제: orbithall-admin `docs/pending/api-key-regeneration.md`

## 현재 문제
- API Key는 사이트 생성 시(`database.CreateSiteForUser`의 `GenerateAPIKey("orb_live_")`)에만 발급되고, `PUT /admin/sites/{id}`로는 바꿀 수 없음
- 키가 유출되면 사이트를 지우고 다시 만드는 것 외에 방법이 없음 (삭제 시 게시글·댓글 cascade 삭제)

## 작업 범위

### 포함
- `POST /admin/sites/{id}/api-key` (owner만) → 새 키 발급 후 반환
- `database.GetSiteByAPIKey`의 API Key → 사이트 캐시(TTL 1분, `internal/database/cache.go`)에서 이전 키 무효화
- 재발급 즉시 이전 키로 들어오는 요청은 401 (캐시를 무효화하지 않으면 최대 1분간 이전 키가 통과함)

### 제외
- 이전 키 유예 기간(grace period), 키 여러 개 동시 운영

## 예상 시간
1-2시간

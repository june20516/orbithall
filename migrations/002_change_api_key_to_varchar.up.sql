-- api_key 타입을 UUID에서 VARCHAR로 변경
-- Prefixed API key 지원 (orb_live_, orb_test_)

BEGIN;

-- 기존 UUID 타입을 VARCHAR로 변경
ALTER TABLE sites
ALTER COLUMN api_key TYPE VARCHAR(100);

-- DEFAULT 제거 (앱에서 생성하도록)
ALTER TABLE sites
ALTER COLUMN api_key DROP DEFAULT;

COMMIT;

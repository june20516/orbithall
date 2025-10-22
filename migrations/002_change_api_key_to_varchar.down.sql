-- api_key 타입을 VARCHAR에서 UUID로 되돌림

BEGIN;

-- VARCHAR를 UUID로 변경
ALTER TABLE sites
ALTER COLUMN api_key TYPE UUID USING api_key::uuid;

-- DEFAULT 복원
ALTER TABLE sites
ALTER COLUMN api_key SET DEFAULT gen_random_uuid();

COMMIT;

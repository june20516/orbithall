-- refresh_tokens 테이블 생성
-- 어드민 로그인 세션을 연장하는 Refresh Token 저장소
BEGIN;

-- ============================================
-- refresh_tokens 테이블
-- ============================================
-- 토큰 원문은 저장하지 않고 SHA-256 해시만 저장한다
-- 한 번의 로그인에서 회전으로 이어진 토큰들은 같은 family_id를 가진다
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL,
    parent_id BIGINT REFERENCES refresh_tokens(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    family_expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(30),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 토큰 만료는 계열의 절대 만료를 넘을 수 없다 (IsExpired가 expires_at만 비교하는 전제)
    CONSTRAINT chk_refresh_tokens_expiry CHECK (expires_at <= family_expires_at)
);

-- refresh_tokens 테이블 인덱스
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
-- 부모 토큰 하나에서는 후속 토큰이 하나만 나온다 (계열이 갈라지지 않음을 DB가 보장)
CREATE UNIQUE INDEX idx_refresh_tokens_parent_id ON refresh_tokens(parent_id) WHERE parent_id IS NOT NULL;

COMMIT;

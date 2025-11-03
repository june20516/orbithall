-- user_sites 테이블 생성
-- 사용자와 사이트 간의 다대다 관계 (소유권)
BEGIN;

-- ============================================
-- user_sites 테이블
-- ============================================
-- 사용자-사이트 관계 (한 사용자가 여러 사이트를 소유 가능)
CREATE TABLE user_sites (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'owner' NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, site_id)
);

-- user_sites 테이블 인덱스
CREATE INDEX idx_user_sites_user_id ON user_sites(user_id);
CREATE INDEX idx_user_sites_site_id ON user_sites(site_id);

COMMIT;

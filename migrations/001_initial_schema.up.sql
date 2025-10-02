-- 초기 스키마 생성
-- 트랜잭션 시작 (전체 성공 또는 전체 실패)
BEGIN;

-- ============================================
-- sites 테이블
-- ============================================
-- Orbithall을 사용하는 사이트 정보 (멀티 테넌시)
CREATE TABLE sites (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL UNIQUE,
    api_key UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    cors_origins TEXT[] NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- sites 테이블 인덱스
CREATE INDEX idx_sites_api_key ON sites(api_key);
CREATE INDEX idx_sites_domain ON sites(domain);
CREATE INDEX idx_sites_is_active ON sites(is_active) WHERE is_active = TRUE;

-- ============================================
-- posts 테이블
-- ============================================
-- 블로그 포스트 메타데이터 (댓글 연결용)
CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    slug VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    comment_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- slug는 사이트 내에서만 unique
    UNIQUE(site_id, slug)
);

-- posts 테이블 인덱스
CREATE INDEX idx_posts_site_id ON posts(site_id);
CREATE INDEX idx_posts_site_slug ON posts(site_id, slug);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);

-- ============================================
-- comments 테이블
-- ============================================
-- 댓글 데이터
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES comments(id) ON DELETE CASCADE,

    -- 작성자 정보
    author_name VARCHAR(100) NOT NULL,
    author_password VARCHAR(255) NOT NULL,

    -- 댓글 내용
    content TEXT NOT NULL,

    -- 메타데이터
    is_deleted BOOLEAN DEFAULT FALSE,
    ip_address INET,
    user_agent TEXT,

    -- 타임스탬프
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- comments 테이블 인덱스
CREATE INDEX idx_comments_post_id ON comments(post_id, created_at DESC);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);
CREATE INDEX idx_comments_is_deleted ON comments(is_deleted) WHERE is_deleted = FALSE;

-- 트랜잭션 커밋
COMMIT;

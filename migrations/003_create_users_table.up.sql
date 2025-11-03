-- users 테이블 생성
-- Google OAuth 로그인 사용자 정보 저장
BEGIN;

-- ============================================
-- users 테이블
-- ============================================
-- Admin 사용자 정보 (Google OAuth 기반)
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    picture_url TEXT,
    google_id VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- users 테이블 인덱스
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);

COMMIT;

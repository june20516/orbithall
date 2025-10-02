-- 초기 스키마 롤백
BEGIN;

-- 역순으로 테이블 삭제 (외래키 제약조건 때문)
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS sites;

COMMIT;

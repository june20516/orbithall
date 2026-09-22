-- 엔티티로 저장된 댓글 본문과 작성자 이름을 평문으로 되돌림
-- 이전 sanitizer는 태그를 지우면서 텍스트의 특수문자를 엔티티로 바꿔 저장했고,
-- 그때 생기는 엔티티는 &lt; &gt; &#34; &#39; &amp; 다섯 가지다.
-- (사용자가 &#13;을 직접 입력한 경우만 &#13;으로 남으며, 무시할 수 있는 수준이다)
-- 이전 sanitizer는 사용자가 입력한 엔티티(&lt; 등)를 이미 문자로 풀어 저장했으므로 그 원문은 복구할 수 없다.
-- &amp;를 마지막에 바꿔야 저장값 "&amp;lt;"(원래 텍스트 "&lt;")가 "<"로 두 번 풀리지 않는다.
-- updated_at은 건드리지 않는다 (위젯이 updated_at으로 "수정됨"을 표시함).
--
-- 이 변환은 두 번 적용하면 평문을 한 번 더 풀어 버린다.
-- force 4 후 다시 up 하는 경우(롤백, dirty 복구)에도 한 번만 적용되도록 data_fixups에 기록하고,
-- 기록이 이미 있으면 아무것도 바꾸지 않는다.
BEGIN;

-- 한 번만 실행해야 하는 데이터 변환의 적용 기록
CREATE TABLE IF NOT EXISTS data_fixups (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);

WITH marker AS (
    INSERT INTO data_fixups (name) VALUES ('005_unescape_comment_entities')
    ON CONFLICT DO NOTHING
    RETURNING 1
)
UPDATE comments
SET
    content = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(content,
        '&lt;', '<'), '&gt;', '>'), '&#34;', '"'), '&#39;', ''''), '&amp;', '&'),
    author_name = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(author_name,
        '&lt;', '<'), '&gt;', '>'), '&#34;', '"'), '&#39;', ''''), '&amp;', '&')
WHERE EXISTS (SELECT 1 FROM marker)
  AND (content ~ '&(lt|gt|#34|#39|amp);'
       OR author_name ~ '&(lt|gt|#34|#39|amp);');

COMMIT;

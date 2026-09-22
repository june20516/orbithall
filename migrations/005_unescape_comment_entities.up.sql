-- 엔티티로 저장된 댓글 본문과 작성자 이름을 평문으로 되돌림
-- 이전 sanitizer는 태그를 지우면서 텍스트의 특수문자를 엔티티로 바꿔 저장했고,
-- 그때 생기는 엔티티는 &lt; &gt; &#34; &#39; &amp; 다섯 가지뿐이다.
-- &amp;를 마지막에 바꿔야 사용자가 직접 입력한 "&lt;"(저장값 "&amp;lt;")가 "<"로 두 번 풀리지 않는다.
-- updated_at은 건드리지 않는다 (위젯이 updated_at으로 "수정됨"을 표시함).
BEGIN;

UPDATE comments
SET
    content = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(content,
        '&lt;', '<'), '&gt;', '>'), '&#34;', '"'), '&#39;', ''''), '&amp;', '&'),
    author_name = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(author_name,
        '&lt;', '<'), '&gt;', '>'), '&#34;', '"'), '&#39;', ''''), '&amp;', '&')
WHERE content ~ '&(lt|gt|#34|#39|amp);'
   OR author_name ~ '&(lt|gt|#34|#39|amp);';

COMMIT;

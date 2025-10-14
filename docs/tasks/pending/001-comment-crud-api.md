# [WIP] 댓글 CRUD API 구현

## 작성일
2025-10-14

## 우선순위
- [x] 긴급
- [ ] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
댓글 시스템의 핵심 기능인 댓글 생성, 조회, 수정, 삭제 API 엔드포인트 구현

## 작업 목적
블로그 사이트에서 댓글을 작성하고 관리할 수 있는 REST API 제공. API 키 기반 인증으로 멀티 테넌시 지원.

## 작업 범위

### 포함 사항
- 댓글 생성 API (POST /api/comments)
- 댓글 목록 조회 API (GET /api/comments)
- 댓글 수정 API (PUT /api/comments/:id)
- 댓글 삭제 API (DELETE /api/comments/:id)
- API 키 인증 미들웨어
- CORS 검증
- 입력 검증 (길이 제한, 필수 필드 등)
- 대댓글(중첩 댓글) 지원

### 제외 사항
- Rate Limiting (다음 작업)
- 스팸 필터링 (추후)
- 실시간 알림 (추후)
- 파일 첨부 (추후)

## 기술적 접근

### 사용할 기술/라이브러리
- Chi 라우터 (이미 사용 중)
- database/sql (이미 설정됨)
- bcrypt (비밀번호 해싱)
- 기존 models, database 패키지 활용

### 파일 구조
```
orbithall/
├── internal/
│   ├── handlers/
│   │   ├── comments.go          # 댓글 CRUD 핸들러
│   │   └── middleware.go        # API 키 인증 미들웨어
│   └── validators/
│       └── comment.go            # 입력 검증
└── cmd/api/
    └── main.go                   # 라우팅 추가
```

## 구현 단계

### 1. API 명세 정의
- 각 엔드포인트의 요청/응답 스키마
- 에러 코드 정의
- 인증 헤더 규칙

### 2. 미들웨어 구현
- API 키 검증
- CORS 확인
- 사이트 정보 캐싱 활용

### 3. 댓글 생성 API
- 입력 검증 (author_name, content, password)
- 비밀번호 해싱
- IP 주소, User-Agent 저장
- posts 테이블 comment_count 증가

### 4. 댓글 조회 API
- site_id + post_slug 필터링
- 대댓글 트리 구조 반환
- 페이지네이션 (선택)
- 삭제된 댓글 처리

### 5. 댓글 수정 API
- 비밀번호 확인
- 수정 가능 시간 제한 (30분?)
- 내용만 수정 가능

### 6. 댓글 삭제 API
- 비밀번호 확인
- Soft delete (is_deleted = true)
- posts 테이블 comment_count 감소
- 대댓글 있으면 "삭제된 댓글입니다" 표시

## 검증 방법

### 테스트 시나리오
1. **댓글 생성**
   - curl로 댓글 작성 요청
   - DB에 저장 확인
   - 비밀번호 해싱 확인

2. **댓글 조회**
   - 특정 포스트의 댓글 목록 조회
   - 대댓글 구조 확인
   - 다른 사이트 데이터 격리 확인

3. **댓글 수정**
   - 올바른 비밀번호로 수정 성공
   - 잘못된 비밀번호로 수정 실패
   - 다른 사람 댓글 수정 불가

4. **댓글 삭제**
   - 올바른 비밀번호로 삭제 성공
   - soft delete 확인
   - comment_count 감소 확인

5. **인증 실패**
   - API 키 없이 요청 → 401
   - 잘못된 API 키 → 403
   - 비활성 사이트 → 403

## 의존성
- 선행 작업: 데이터베이스 연결 및 스키마 구현 (완료)
- 후속 작업: Rate Limiting 구현

## 예상 소요 시간
- 예상: 4-6시간

## 주의사항
- 비밀번호는 절대 응답에 포함하지 않음
- IP 주소, User-Agent도 비공개
- SQL Injection 방어 (prepared statement)
- XSS 방어 (입력 검증)
- 대댓글 깊이 제한 필요 (무한 중첩 방지)

## 참고 자료
- Chi 라우터: https://github.com/go-chi/chi
- bcrypt: https://pkg.go.dev/golang.org/x/crypto/bcrypt

---

## 작업 이력

### [2025-10-14] 작업 문서 초안 작성
- 러프한 구조 작성
- 구체적인 API 명세는 작업 시작 시 추가 예정

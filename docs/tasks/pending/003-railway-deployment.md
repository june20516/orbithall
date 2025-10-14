# [WIP] Railway 배포 설정

## 작성일
2025-10-14

## 우선순위
- [ ] 긴급
- [x] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
Railway 플랫폼에 Orbithall 댓글 시스템 배포 및 프로덕션 환경 구성

## 작업 목적
개발 완료된 댓글 시스템을 실제 운영 환경에 배포하여 블로그에서 사용 가능하게 함

## 작업 범위

### 포함 사항
- Railway PostgreSQL 서비스 생성
- Railway 앱 서비스 생성 및 배포
- 환경변수 설정
- 도메인 연결 (Railway 제공 도메인)
- HTTPS 자동 설정 확인
- 마이그레이션 자동 실행 확인

### 제외 사항
- 커스텀 도메인 (추후 필요시)
- CDN 설정 (현재 필요 없음)
- 모니터링 도구 (추후)
- 백업 자동화 (Railway 기본 기능 사용)

## 기술적 접근

### Railway 구조
```
Railway Project: orbithall
├── PostgreSQL Service
│   └── DATABASE_URL (자동 생성)
└── API Service
    ├── Dockerfile 사용
    ├── 환경변수 설정
    └── 마이그레이션 자동 실행
```

### 필요한 환경변수
```
DATABASE_URL    # Railway에서 자동 주입
CORS_ORIGIN     # 블로그 도메인
ENV=production
PORT            # Railway에서 자동 할당
```

## 구현 단계

### 1. Railway 프로젝트 생성
- Railway 계정 로그인
- 새 프로젝트 생성
- GitHub 연동 (선택)

### 2. PostgreSQL 서비스 추가
- Add Service → PostgreSQL
- DATABASE_URL 자동 생성 확인

### 3. API 서비스 추가
- Add Service → GitHub Repo (또는 Empty Service)
- Dockerfile 사용 설정
- production stage 지정

### 4. 환경변수 설정
- CORS_ORIGIN 설정
- ENV=production 설정
- DATABASE_URL 연결 확인

### 5. 배포 및 검증
- 자동 배포 트리거
- 로그에서 마이그레이션 실행 확인
- Health check 엔드포인트 확인
- 댓글 API 테스트

### 6. 블로그 연동
- Railway 도메인 복사
- 블로그 Comments 컴포넌트 수정
- CORS 확인

## 검증 방법

### 테스트 시나리오
1. **배포 성공**
   - 빌드 로그 확인
   - 컨테이너 실행 확인
   - 헬스체크 200 응답

2. **데이터베이스 연결**
   - 마이그레이션 로그 확인
   - 테이블 생성 확인
   - 테스트 사이트 등록

3. **HTTPS 통신**
   - Railway 도메인이 HTTPS인지 확인
   - 블로그에서 API 호출 테스트

4. **CORS 검증**
   - 블로그 도메인에서 요청 성공
   - 다른 도메인에서 요청 차단

## 의존성
- 선행 작업: 댓글 CRUD API, Rate Limiting 구현
- 후속 작업: 프로덕션 환경변수 관리

## 예상 소요 시간
- 예상: 1-2시간

## 주의사항
- Railway 무료 플랜 제한 확인 ($5 credit/month)
- DATABASE_URL에 sslmode=require 설정 필요
- Dockerfile production stage 최적화 확인
- 민감한 정보 로그에 출력 금지

## 참고 자료
- Railway 공식 문서: https://docs.railway.app/
- Railway PostgreSQL: https://docs.railway.app/databases/postgresql

---

## 작업 이력

### [2025-10-14] 작업 문서 초안 작성
- 러프한 구조 작성
- Railway 무료 플랜 기준으로 작성

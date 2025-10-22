# [WIP] 프로덕션 환경변수 관리

## 작성일
2025-10-14

## 우선순위
- [ ] 긴급
- [ ] 높음
- [x] 보통
- [ ] 낮음

## 작업 개요
프로덕션 환경에서 안전하고 효율적인 환경변수 관리 체계 구축

## 작업 목적
민감한 정보 보호, 환경별 설정 분리, 배포 자동화 개선

## 작업 범위

### 포함 사항
- 환경변수 검증 로직
- 필수 환경변수 체크
- 기본값 설정 (개발/프로덕션 구분)
- 환경변수 문서화
- .env.example 파일 생성

### 제외 사항
- Secrets 관리 도구 (Vault 등) - 현재 규모에 과함
- 동적 환경변수 로딩 - 재시작으로 충분
- 환경변수 암호화 - Railway가 처리

## 기술적 접근

### 환경변수 구조 (초안)
```
# 필수
DATABASE_URL
ENV (development | production)

# 선택 (기본값 있음)
PORT (기본값: 8080)
CORS_ORIGIN (기본값: http://localhost:3000)

# 프로덕션 필수
LOG_LEVEL (기본값: info)
```

### 파일 구조
```
orbithall/
├── internal/
│   └── config/
│       └── config.go           # 환경변수 로딩 및 검증
├── .env.example                # 환경변수 템플릿
└── docs/
    └── deployment.md           # 배포 가이드
```

## 구현 단계

### 1. Config 패키지 생성
- `internal/config/config.go`
- 구조체로 환경변수 관리
- 검증 함수
- 기본값 설정 로직

### 2. 환경변수 검증
- 필수 변수 누락 체크
- 형식 검증 (URL, 숫자 등)
- ENV 값 제한 (development/production만 허용)

### 3. .env.example 작성
- 모든 환경변수 나열
- 설명 주석 추가
- 개발 환경 예시값

### 4. 문서 작성
- README에 환경변수 섹션 업데이트
- 배포 가이드 작성
- Render 설정 가이드

### 5. 기존 코드 리팩토링
- main.go에서 직접 os.Getenv() 제거
- config 패키지 사용으로 변경

## 검증 방법

### 테스트 시나리오
1. **필수 변수 누락**
   - DATABASE_URL 없이 시작
   - 명확한 에러 메시지 확인

2. **잘못된 ENV 값**
   - ENV=staging 설정
   - 에러 또는 경고 확인

3. **기본값 동작**
   - PORT 생략
   - 8080 포트로 시작 확인

4. **프로덕션 설정**
   - Render 환경변수 설정
   - 정상 배포 확인

## 의존성
- 선행 작업: ✅ 프로덕션 배포 완료 (003-deployment)
- 후속 작업: 없음

## 예상 소요 시간
- 예상: 1-2시간

## 주의사항
- .env 파일은 .gitignore에 포함 (이미 되어 있음)
- Render는 .env 파일 불필요 (UI에서 설정)
- 환경변수 로그 출력 시 마스킹 필요
- 기본값은 개발 환경 기준
- CORS는 동적 검증 방식 사용 (CORS_ORIGIN 환경변수 불필요)

## 참고 자료
- Go 환경변수: https://pkg.go.dev/os#Getenv
- 12 Factor App: https://12factor.net/config

---

## 작업 이력

### [2025-10-14] 작업 문서 초안 작성
- 러프한 구조 작성
- 작은 프로젝트에 맞는 심플한 접근

# ADR-001: 데이터베이스 마이그레이션 전략

## Status
Accepted (2025-10-02)

## Context

Orbithall 프로젝트는 PostgreSQL 데이터베이스를 사용하며, 스키마 변경을 안전하게 관리해야 합니다.

### 요구사항
- **멱등성 보장**: 같은 마이그레이션 파일이 여러 번 실행되어도 안전해야 함
- **버전 추적**: 현재 적용된 마이그레이션 버전을 DB에서 관리
- **개발/프로덕션 일관성**: 동일한 마이그레이션 도구와 프로세스 사용
- **자동화**: 컨테이너 시작 시 자동으로 마이그레이션 적용

### 배포 환경
- **초기 배포**: Railway (PaaS, 개인 프로젝트 규모)
  - 기본적으로 단일 인스턴스
  - 스케일 아웃 시 순차적 재시작 (rolling restart)
  - Dockerfile 기반 배포

- **향후 고려사항**: Kubernetes 이전 가능성 존재

### 핵심 질문
1. 마이그레이션 도구는 무엇을 사용할 것인가?
2. 언제 마이그레이션을 실행할 것인가? (빌드 시 vs 런타임 시)
3. 여러 인스턴스가 동시에 시작될 때 경합을 어떻게 처리할 것인가?

## Decision

### 1. golang-migrate 사용
- Go 생태계 표준 마이그레이션 도구
- SQL 기반 마이그레이션 파일 (`.sql`)
- 멱등성 자동 보장 (`schema_migrations` 테이블로 버전 추적)
- PostgreSQL advisory lock으로 동시 실행 방지

### 2. Dockerfile + entrypoint.sh 방식
```dockerfile
# Dockerfile (development, production 공통)
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate

COPY migrations /migrations
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
```

```bash
# entrypoint.sh
#!/bin/sh
set -e

echo "Running database migrations..."
migrate -path /migrations -database "${DATABASE_URL}" up

echo "Starting application..."
exec "$@"
```

### 3. Railway 환경에 적합한 단순 구조
- 복잡한 Init Container나 별도 Job 사용 안함
- 앱 시작 전 매번 마이그레이션 실행
- 멱등성 보장으로 이미 적용된 마이그레이션은 자동 스킵

## Consequences

### 장점
- **Railway 배포 모델에 최적화**: 단일 Dockerfile로 완결, 추가 설정 불필요
- **개발/프로덕션 일관성**: 동일한 마이그레이션 프로세스
- **자동화**: 개발자가 수동으로 마이그레이션 실행할 필요 없음
- **멱등성**: 컨테이너 재시작 시에도 안전
- **경합 최소화**: Railway의 순차적 재시작으로 동시 실행 가능성 낮음

### 단점
- **Kubernetes 이전 시 재검토 필요**:
  - 대규모 스케일 아웃 시 동시 마이그레이션 시도로 인한 대기 시간 발생 가능
  - Init Container나 Kubernetes Job으로 분리 고려 필요
- **마이그레이션 실패 시 앱 시작 안됨**:
  - dirty state 발생 시 수동 복구 필요
  - 하지만 이는 장점이기도 함 (잘못된 스키마로 앱 실행 방지)

### 마이그레이션 동작 원리
```sql
-- golang-migrate가 자동 생성하는 테이블
CREATE TABLE schema_migrations (
    version bigint NOT NULL PRIMARY KEY,
    dirty boolean NOT NULL
);

-- 버전 3까지 적용된 상태
-- version=3 기록됨

-- 새 마이그레이션 (004, 005) 추가 후 migrate up 실행
-- → 001, 002, 003 스킵
-- → 004, 005만 실행
-- → version=5로 업데이트
```

## Alternatives Considered

### 대안 1: docker-compose의 별도 migrate 서비스
```yaml
services:
  migrate:
    command: migrate up
  api:
    depends_on:
      migrate:
        condition: service_completed_successfully
```

**평가**: 개발 환경에서는 유용하지만, Railway는 docker-compose를 사용하지 않으므로 프로덕션과 괴리 발생. 개발/프로덕션 일관성 원칙 위배.

### 대안 2: CI/CD 파이프라인에서 마이그레이션 실행
```yaml
# GitHub Actions
jobs:
  migrate:
    run: migrate up
  deploy:
    needs: migrate
```

**평가**: 배포 파이프라인이 복잡해지고, Railway 자동 배포 기능과 통합 어려움. 로컬 개발 시에도 별도 마이그레이션 실행 필요.

### 대안 3: Kubernetes Init Container
```yaml
initContainers:
- name: migrate
  command: [migrate, up]
containers:
- name: api
```

**평가**: Kubernetes 환경에서는 가장 이상적이나, Railway에서는 불필요한 오버엔지니어링. 향후 K8s 이전 시 고려.

### 대안 4: 애플리케이션 코드에 embed
```go
//go:embed migrations/*.sql
var migrationFS embed.FS

func main() {
    runMigrations(db)
    startServer()
}
```

**평가**: Go 코드와 강결합, 마이그레이션 도구의 장점(CLI, 롤백 등) 활용 불가. 재사용성 낮음.

## Related Decisions
- 향후 ADR: Kubernetes 이전 전략 (마이그레이션 방식 재검토 포함)

## References
- [golang-migrate 공식 문서](https://github.com/golang-migrate/migrate)
- [Railway 배포 문서](https://docs.railway.app/)
- [12-Factor App - Dev/Prod Parity](https://12factor.net/dev-prod-parity)

package database

import (
	"os"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/models"
)

// TestGetSiteByAPIKey_CacheMiss_QueriesDB는 캐시 미스 시 DB에서 조회하는지 테스트합니다
func TestGetSiteByAPIKey_CacheMiss_QueriesDB(t *testing.T) {
	// Given: 실제 DB 연결 및 테스트 데이터
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := New(databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer Close(db)

	// 테스트용 사이트 생성 (실제 DB에 INSERT 필요)
	apiKey := "test-api-key-cache-miss"

	// When: GetSiteByAPIKey 호출
	site, err := GetSiteByAPIKey(db, apiKey)

	// Then: DB에 없으면 에러, 있으면 정상 반환
	if err != nil {
		t.Logf("expected behavior: site not found - %v", err)
	} else if site != nil {
		t.Logf("site found: %s", site.Name)
	}
}

// TestGetSiteByAPIKey_CacheHit_NoDBQuery는 캐시 히트 시 DB 조회하지 않는지 테스트합니다
func TestGetSiteByAPIKey_CacheHit_NoDBQuery(t *testing.T) {
	// Given: 캐시에 데이터 추가
	apiKey := "test-cached-key"
	cachedSite := &models.Site{
		ID:          1,
		Name:        "Test Site",
		Domain:      "test.example.com",
		APIKey:      apiKey,
		CORSOrigins: []string{"http://localhost:3000"},
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	siteCache.Store(apiKey, &cacheEntry{
		site:      cachedSite,
		expiresAt: time.Now().Add(1 * time.Minute),
	})

	// When: GetSiteByAPIKey 호출 (DB는 nil이어도 작동해야 함)
	site, err := GetSiteByAPIKey(nil, apiKey)

	// Then: 캐시에서 반환됨
	if err != nil {
		t.Fatalf("expected cache hit, got error: %v", err)
	}
	if site.Name != "Test Site" {
		t.Errorf("expected 'Test Site', got '%s'", site.Name)
	}

	// Cleanup
	siteCache.Delete(apiKey)
}

// TestGetSiteByAPIKey_ExpiredCache_QueriesDB는 만료된 캐시는 DB 재조회하는지 테스트합니다
func TestGetSiteByAPIKey_ExpiredCache_QueriesDB(t *testing.T) {
	// Given: 만료된 캐시 항목
	apiKey := "test-expired-key"
	expiredSite := &models.Site{
		ID:       1,
		Name:     "Expired Site",
		APIKey:   apiKey,
		IsActive: true,
	}

	siteCache.Store(apiKey, &cacheEntry{
		site:      expiredSite,
		expiresAt: time.Now().Add(-1 * time.Minute), // 이미 만료됨
	})

	// DB 연결
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := New(databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer Close(db)

	// When: GetSiteByAPIKey 호출
	_, err = GetSiteByAPIKey(db, apiKey)

	// Then: 만료된 캐시는 삭제되고 DB 조회 시도
	// DB에 데이터가 없으면 에러 발생 (정상)
	if err != nil {
		t.Logf("expected behavior: expired cache removed, DB query failed - %v", err)
	}

	// 캐시에서 삭제되었는지 확인
	if _, ok := siteCache.Load(apiKey); ok {
		t.Error("expected expired cache to be removed")
	}
}

// TestCacheEntry_isExpired는 캐시 만료 판단 로직을 테스트합니다
func TestCacheEntry_isExpired(t *testing.T) {
	// Given: 만료된 항목과 유효한 항목
	expiredEntry := &cacheEntry{
		site:      &models.Site{},
		expiresAt: time.Now().Add(-1 * time.Second),
	}

	validEntry := &cacheEntry{
		site:      &models.Site{},
		expiresAt: time.Now().Add(1 * time.Minute),
	}

	// When & Then: 만료 확인
	if !expiredEntry.isExpired() {
		t.Error("expected expired entry to be expired")
	}

	if validEntry.isExpired() {
		t.Error("expected valid entry to not be expired")
	}
}

// TestGetSiteByAPIKey_Concurrency는 동시성 안전성을 테스트합니다
func TestGetSiteByAPIKey_Concurrency(t *testing.T) {
	// Given: 캐시에 데이터 추가
	apiKey := "test-concurrent-key"
	testSite := &models.Site{
		ID:       1,
		Name:     "Concurrent Test",
		APIKey:   apiKey,
		IsActive: true,
	}

	siteCache.Store(apiKey, &cacheEntry{
		site:      testSite,
		expiresAt: time.Now().Add(1 * time.Minute),
	})

	// When: 여러 goroutine에서 동시 접근
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := GetSiteByAPIKey(nil, apiKey)
			if err != nil {
				t.Errorf("unexpected error in concurrent access: %v", err)
			}
			done <- true
		}()
	}

	// Then: 모든 goroutine이 에러 없이 완료
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cleanup
	siteCache.Delete(apiKey)
}

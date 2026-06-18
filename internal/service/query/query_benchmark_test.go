package query

import (
	"context"
	"testing"

	"blockexplore/internal/model"
)

// BenchmarkGetBlockList_CacheMiss 模拟缓存未命中（每次都查 DB mock）
func BenchmarkGetBlockList_CacheMiss(b *testing.B) {
	repo := &mockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		total:  1,
	}
	// nil cache = 永远未命中，直接走 repo
	svc := NewQueryService(repo, &mockTxRepo{}, nil)
	ctx := context.Background()
	_ = ctx

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetBlockList("eth", 1, 20)
	}
}

// BenchmarkGetBlockList_CacheHit 模拟缓存命中（不查 DB）
func BenchmarkGetBlockList_CacheHit(b *testing.B) {
	repo := &mockBlockRepo{}
	cache := newMockCache()
	// 预填缓存
	preloaded := &BlockListResponse{
		Chain:  "eth",
		Blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		Pagination: Pagination{Page: 1, PageSize: 20, Total: 1},
	}
	_ = cache.Set(context.Background(), "blocks:eth:1:20", preloaded, 0)

	svc := NewQueryService(repo, &mockTxRepo{}, cache)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetBlockList("eth", 1, 20)
	}
}

// BenchmarkGetBlockList_Parallel 并发场景（nil cache，避免 mock map 并发写）
func BenchmarkGetBlockList_Parallel(b *testing.B) {
	repo := &mockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		total:  1,
	}
	// nil cache：每次走 repo，测 service 层并发开销
	svc := NewQueryService(repo, &mockTxRepo{}, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = svc.GetBlockList("eth", 1, 20)
		}
	})
}

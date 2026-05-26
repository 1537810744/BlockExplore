// Package price 提供价格查询服务
// 从外部 API 获取代币价格，写入数据库和缓存
package price

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"blockexplore/internal/model"
	"blockexplore/internal/repository"
	"blockexplore/pkg/cache"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// PriceService 价格服务
type PriceService struct {
	priceRepo *repository.PriceRepo // 价格数据访问层
	cache     *cache.RedisClient    // Redis 缓存
	apiURL    string                // CoinGecko API 地址
	httpClient *http.Client         // HTTP 客户端
}

// NewPriceService 创建价格服务实例
func NewPriceService(priceRepo *repository.PriceRepo, redisClient *cache.RedisClient, apiURL string) *PriceService {
	return &PriceService{
		priceRepo:  priceRepo,
		cache:      redisClient,
		apiURL:     apiURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// PriceResponse 价格响应
type PriceResponse struct {
	Chain     string  `json:"chain"`      // 链标识
	Symbol    string  `json:"symbol"`     // 代币符号
	PriceUSD  float64 `json:"price_usd"`  // 美元价格
	Timestamp int64   `json:"timestamp"`  // 时间戳
}

// PriceHistoryResponse 价格历史响应
type PriceHistoryResponse struct {
	Chain   string                  `json:"chain"`   // 链标识
	Symbol  string                  `json:"symbol"`  // 代币符号
	Prices  []model.PriceHistory    `json:"prices"`  // 价格列表
}

// 各链对应的 CoinGecko 代币 ID
var chainCoinIDs = map[string]string{
	"eth": "ethereum",
	"btc": "bitcoin",
	"sol": "solana",
}

// 各链对应的代币符号
var chainSymbols = map[string]string{
	"eth": "ETH",
	"btc": "BTC",
	"sol": "SOL",
}

// GetCurrentPrice 获取指定链的当前价格
func (s *PriceService) GetCurrentPrice(chain string) (*PriceResponse, error) {
	// 先从缓存读取
	cacheKey := fmt.Sprintf("price:current:%s", chain)
	var priceResp PriceResponse
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), cacheKey, &priceResp); err == nil {
			return &priceResp, nil
		}
	}

	// 缓存未命中，从数据库读取最新价格
	price, err := s.priceRepo.GetLatestPrice(chain)
	if err != nil {
		// 数据库也没有，从外部 API 获取
		return s.fetchPriceFromAPI(chain)
	}

	// 将数据库价格转换为响应格式
	var priceUSD float64
	fmt.Sscanf(price.PriceUSD, "%f", &priceUSD)

	priceResp = PriceResponse{
		Chain:     chain,
		Symbol:    price.Symbol,
		PriceUSD:  priceUSD,
		Timestamp: price.Timestamp,
	}

	// 写入缓存（60 秒过期）
	if s.cache != nil {
		s.cache.Set(context.Background(), cacheKey, &priceResp, 60*time.Second)
	}

	return &priceResp, nil
}

// GetPriceHistory 获取价格历史
func (s *PriceService) GetPriceHistory(chain string, startTime, endTime int64, limit int) (*PriceHistoryResponse, error) {
	symbol := chainSymbols[chain]

	prices, err := s.priceRepo.GetPriceHistory(chain, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	return &PriceHistoryResponse{
		Chain:  chain,
		Symbol: symbol,
		Prices: prices,
	}, nil
}

// SyncPrices 从 CoinGecko API 同步价格
// 定时任务会调用此方法
func (s *PriceService) SyncPrices() error {
	for chain, coinID := range chainCoinIDs {
		price, err := s.fetchPrice(coinID)
		if err != nil {
			logger.Error("获取价格失败",
				zap.String("chain", chain),
				zap.Error(err),
			)
			continue
		}

		// 保存到数据库
		priceHistory := model.PriceHistory{
			Chain:     chain,
			Symbol:    chainSymbols[chain],
			PriceUSD:  fmt.Sprintf("%.8f", price),
			Timestamp: time.Now().Unix(),
		}

		if err := s.priceRepo.Create(&priceHistory); err != nil {
			logger.Error("保存价格失败",
				zap.String("chain", chain),
				zap.Error(err),
			)
			continue
		}

		// 更新缓存
		cacheKey := fmt.Sprintf("price:current:%s", chain)
		if s.cache != nil {
			s.cache.Set(context.Background(), cacheKey, &PriceResponse{
				Chain:     chain,
				Symbol:    chainSymbols[chain],
				PriceUSD:  price,
				Timestamp: priceHistory.Timestamp,
			}, 60*time.Second)
		}

		logger.Debug("价格已更新",
			zap.String("chain", chain),
			zap.Float64("price", price),
		)
	}

	return nil
}

// fetchPriceFromAPI 从外部 API 获取价格
func (s *PriceService) fetchPriceFromAPI(chain string) (*PriceResponse, error) {
	coinID := chainCoinIDs[chain]
	price, err := s.fetchPrice(coinID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	return &PriceResponse{
		Chain:     chain,
		Symbol:    chainSymbols[chain],
		PriceUSD:  price,
		Timestamp: now,
	}, nil
}

// fetchPrice 从 CoinGecko API 获取代币价格
func (s *PriceService) fetchPrice(coinID string) (float64, error) {
	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd", s.apiURL, coinID)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("请求价格 API 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取价格响应失败: %w", err)
	}

	// CoinGecko 返回格式: {"bitcoin": {"usd": 50000}}
	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析价格数据失败: %w", err)
	}

	coinData, ok := result[coinID]
	if !ok {
		return 0, fmt.Errorf("未找到代币 %s 的价格数据", coinID)
	}

	price, ok := coinData["usd"]
	if !ok {
		return 0, fmt.Errorf("未找到 USD 价格")
	}

	return price, nil
}

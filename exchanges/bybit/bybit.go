package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/types"
)

const (
	bybitName       = "bybit"
	bybitBaseURL    = "https://api.bybit.com"
	bybitSandboxURL = "https://api-testnet.bybit.com"
)

// Bybit Bybit交易所实现
type Bybit struct {
	*exlink.BaseExchange
	client    *common.HTTPClient
	apiKey    string
	secretKey string
}

// NewBybit 创建Bybit交易所实例
func NewBybit(apiKey, secretKey string, options map[string]interface{}) (exlink.Exchange, error) {
	baseURL := bybitBaseURL
	sandbox := false
	proxyURL := ""

	if v, ok := options["baseURL"].(string); ok {
		baseURL = v
	}
	if v, ok := options["sandbox"].(bool); ok {
		sandbox = v
	}
	if v, ok := options["proxy"].(string); ok {
		proxyURL = v
	}

	if sandbox {
		baseURL = bybitSandboxURL
	}

	exchange := &Bybit{
		BaseExchange: exlink.NewBaseExchange(bybitName),
		client:       common.NewHTTPClient(baseURL),
		apiKey:       apiKey,
		secretKey:    secretKey,
	}

	// 设置模拟盘模式
	exchange.SetSandbox(sandbox)

	// 设置代理
	if proxyURL != "" {
		exchange.SetProxy(proxyURL)
		if err := exchange.client.SetProxy(proxyURL); err != nil {
			return nil, fmt.Errorf("set proxy: %w", err)
		}
	}

	// 设置请求头
	if apiKey != "" {
		exchange.client.SetHeader("X-BAPI-API-KEY", apiKey)
	}

	// 设置其他选项
	for k, v := range options {
		if k != "baseURL" && k != "sandbox" && k != "proxy" {
			exchange.SetOption(k, v)
		}
	}

	return exchange, nil
}

// LoadMarkets 加载市场信息
func (b *Bybit) LoadMarkets(ctx context.Context, reload bool) error {
	markets := make([]*types.Market, 0)

	// 获取要加载的市场类型
	fetchMarketsTypes := []string{"spot"}
	if v, ok := b.GetOption("fetchMarkets").([]string); ok && len(v) > 0 {
		fetchMarketsTypes = v
	} else if v, ok := b.GetOption("fetchMarkets").(string); ok {
		fetchMarketsTypes = []string{v}
	}

	// 加载现货市场
	if contains(fetchMarketsTypes, "spot") {
		spotMarkets, err := b.loadSpotMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load spot markets: %w", err)
		}
		markets = append(markets, spotMarkets...)
	}

	// 加载永续合约市场（linear）
	if contains(fetchMarketsTypes, "swap") || contains(fetchMarketsTypes, "linear") {
		swapMarkets, err := b.loadSwapMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load swap markets: %w", err)
		}
		markets = append(markets, swapMarkets...)
	}

	b.SetMarkets(markets)
	return nil
}

// loadSpotMarkets 加载现货市场
func (b *Bybit) loadSpotMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := b.client.Get(ctx, "/v5/market/instruments-info", map[string]interface{}{
		"category": "spot",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch spot markets: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol      string `json:"symbol"`
				BaseCoin    string `json:"baseCoin"`
				QuoteCoin   string `json:"quoteCoin"`
				Status      string `json:"status"`
				LotSizeFilter struct {
					BasePrecision  string `json:"basePrecision"`
					QuotePrecision string `json:"quotePrecision"`
					MinOrderQty    string `json:"minOrderQty"`
					MaxOrderQty    string `json:"maxOrderQty"`
					MinOrderAmt    string `json:"minOrderAmt"`
					MaxOrderAmt    string `json:"maxOrderAmt"`
				} `json:"lotSizeFilter"`
				PriceFilter struct {
					TickSize string `json:"tickSize"`
				} `json:"priceFilter"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal spot markets: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	markets := make([]*types.Market, 0)
	for _, s := range result.Result.List {
		if s.Status != "Trading" {
			continue
		}

		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(s.BaseCoin, s.QuoteCoin)

		market := &types.Market{
			ID:     s.Symbol,        // Bybit 原始格式 (BTCUSDT)
			Symbol: normalizedSymbol, // 标准化格式 (BTC/USDT)
			Base:   s.BaseCoin,
			Quote:  s.QuoteCoin,
			Type:   types.MarketTypeSpot,
			Active: s.Status == "Trading",
		}

		// 解析精度
		basePrecision, _ := strconv.ParseFloat(s.LotSizeFilter.BasePrecision, 64)
		tickSize, _ := strconv.ParseFloat(s.PriceFilter.TickSize, 64)
		quotePrecision, _ := strconv.ParseFloat(s.LotSizeFilter.QuotePrecision, 64)

		// 计算精度位数
		market.Precision.Amount = getPrecisionDigits(basePrecision)
		if tickSize > 0 {
			market.Precision.Price = getPrecisionDigits(tickSize)
		} else if quotePrecision > 0 {
			market.Precision.Price = getPrecisionDigits(quotePrecision)
		}

		// 解析限制
		market.Limits.Amount.Min, _ = strconv.ParseFloat(s.LotSizeFilter.MinOrderQty, 64)
		market.Limits.Amount.Max, _ = strconv.ParseFloat(s.LotSizeFilter.MaxOrderQty, 64)
		market.Limits.Cost.Min, _ = strconv.ParseFloat(s.LotSizeFilter.MinOrderAmt, 64)
		market.Limits.Cost.Max, _ = strconv.ParseFloat(s.LotSizeFilter.MaxOrderAmt, 64)

		markets = append(markets, market)
	}

	return markets, nil
}

// loadSwapMarkets 加载永续合约市场（linear）
func (b *Bybit) loadSwapMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := b.client.Get(ctx, "/v5/market/instruments-info", map[string]interface{}{
		"category": "linear",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch swap markets: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol    string `json:"symbol"`
				BaseCoin  string `json:"baseCoin"`
				QuoteCoin string `json:"quoteCoin"`
				Status    string `json:"status"`
				LotSizeFilter struct {
					BasePrecision  string `json:"basePrecision"`
					QuotePrecision string `json:"quotePrecision"`
					MinOrderQty    string `json:"minOrderQty"`
					MaxOrderQty    string `json:"maxOrderQty"`
					MinOrderAmt    string `json:"minOrderAmt"`
					MaxOrderAmt    string `json:"maxOrderAmt"`
				} `json:"lotSizeFilter"`
				PriceFilter struct {
					TickSize string `json:"tickSize"`
				} `json:"priceFilter"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal swap markets: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	markets := make([]*types.Market, 0)
	for _, s := range result.Result.List {
		if s.Status != "Trading" {
			continue
		}

		// Bybit linear 合约的 settle 通常是 quoteCoin
		settle := s.QuoteCoin

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(s.BaseCoin, s.QuoteCoin, settle)

		market := &types.Market{
			ID:       s.Symbol,
			Symbol:   normalizedSymbol,
			Base:     s.BaseCoin,
			Quote:    s.QuoteCoin,
			Settle:   settle,
			Type:     types.MarketTypeSwap,
			Active:   s.Status == "Trading",
			Contract: true,
			Linear:   true, // U本位永续合约
			Inverse:  false,
		}

		// 解析精度
		basePrecision, _ := strconv.ParseFloat(s.LotSizeFilter.BasePrecision, 64)
		tickSize, _ := strconv.ParseFloat(s.PriceFilter.TickSize, 64)
		quotePrecision, _ := strconv.ParseFloat(s.LotSizeFilter.QuotePrecision, 64)

		market.Precision.Amount = getPrecisionDigits(basePrecision)
		if tickSize > 0 {
			market.Precision.Price = getPrecisionDigits(tickSize)
		} else if quotePrecision > 0 {
			market.Precision.Price = getPrecisionDigits(quotePrecision)
		}

		// 解析限制
		market.Limits.Amount.Min, _ = strconv.ParseFloat(s.LotSizeFilter.MinOrderQty, 64)
		market.Limits.Amount.Max, _ = strconv.ParseFloat(s.LotSizeFilter.MaxOrderQty, 64)
		market.Limits.Cost.Min, _ = strconv.ParseFloat(s.LotSizeFilter.MinOrderAmt, 64)
		market.Limits.Cost.Max, _ = strconv.ParseFloat(s.LotSizeFilter.MaxOrderAmt, 64)

		markets = append(markets, market)
	}

	return markets, nil
}

// getPrecisionDigits 计算精度位数
func getPrecisionDigits(value float64) int {
	if value == 0 {
		return 8
	}
	str := fmt.Sprintf("%.10f", value)
	str = strings.TrimRight(str, "0")
	parts := strings.Split(str, ".")
	if len(parts) == 2 {
		return len(parts[1])
	}
	return 0
}

// contains 检查字符串切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetMarketID 获取Bybit格式的 symbol ID
// 优先从已加载的市场中查找，如果未找到则使用后备转换函数
func (b *Bybit) GetMarketID(symbol string) (string, error) {
	// 优先从已加载的市场中查找
	market, ok := b.GetMarketsMap()[symbol]
	if ok {
		return market.ID, nil
	}

	// 如果市场未加载，使用后备转换函数
	return common.ToBybitSymbol(symbol)
}

// GetMarketByID 通过交易所ID获取市场信息
func (b *Bybit) GetMarketByID(id string) (*types.Market, error) {
	for _, market := range b.GetMarketsMap() {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, exlink.ErrMarketNotFound
}

// FetchTicker 获取行情
func (b *Bybit) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	// 确定 category
	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	resp, err := b.client.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"symbol":   bybitSymbol,
		"category": category,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol        string `json:"symbol"`
				Bid1Price     string `json:"bid1Price"`
				Ask1Price     string `json:"ask1Price"`
				LastPrice     string `json:"lastPrice"`
				PrevPrice24h  string `json:"prevPrice24h"`
				HighPrice24h  string `json:"highPrice24h"`
				LowPrice24h   string `json:"lowPrice24h"`
				Volume24h     string `json:"volume24h"`
				Turnover24h   string `json:"turnover24h"`
				Price24hPcnt  string `json:"price24hPcnt"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	if len(result.Result.List) == 0 {
		return nil, fmt.Errorf("ticker not found")
	}

	item := result.Result.List[0]
	ticker := &types.Ticker{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	ticker.Bid, _ = strconv.ParseFloat(item.Bid1Price, 64)
	ticker.Ask, _ = strconv.ParseFloat(item.Ask1Price, 64)
	ticker.Last, _ = strconv.ParseFloat(item.LastPrice, 64)
	ticker.Open, _ = strconv.ParseFloat(item.PrevPrice24h, 64)
	ticker.High, _ = strconv.ParseFloat(item.HighPrice24h, 64)
	ticker.Low, _ = strconv.ParseFloat(item.LowPrice24h, 64)
	ticker.Volume, _ = strconv.ParseFloat(item.Volume24h, 64)
	ticker.QuoteVolume, _ = strconv.ParseFloat(item.Turnover24h, 64)
	changePercent, _ := strconv.ParseFloat(item.Price24hPcnt, 64)
	ticker.ChangePercent = changePercent * 100
	if ticker.Open > 0 {
		ticker.Change = ticker.Last - ticker.Open
	}

	return ticker, nil
}

// FetchTickers 批量获取行情
func (b *Bybit) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// Bybit 需要分别获取 spot 和 linear 的 tickers
	tickers := make(map[string]*types.Ticker)

	// 获取现货 tickers
	resp, err := b.client.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"category": "spot",
	})
	if err == nil {
		var result struct {
			RetCode int    `json:"retCode"`
			RetMsg  string `json:"retMsg"`
			Result  struct {
				List []struct {
					Symbol        string `json:"symbol"`
					Bid1Price     string `json:"bid1Price"`
					Ask1Price     string `json:"ask1Price"`
					LastPrice     string `json:"lastPrice"`
					PrevPrice24h  string `json:"prevPrice24h"`
					HighPrice24h  string `json:"highPrice24h"`
					LowPrice24h   string `json:"lowPrice24h"`
					Volume24h     string `json:"volume24h"`
					Turnover24h   string `json:"turnover24h"`
					Price24hPcnt  string `json:"price24hPcnt"`
				} `json:"list"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp, &result); err == nil && result.RetCode == 0 {
			for _, item := range result.Result.List {
				market, err := b.GetMarketByID(item.Symbol)
				if err != nil {
					continue
				}
				normalizedSymbol := market.Symbol

				if len(symbols) > 0 {
					found := false
					for _, s := range symbols {
						if s == normalizedSymbol {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				ticker := &types.Ticker{
					Symbol:    normalizedSymbol,
					Timestamp: time.Now(),
				}
				ticker.Bid, _ = strconv.ParseFloat(item.Bid1Price, 64)
				ticker.Ask, _ = strconv.ParseFloat(item.Ask1Price, 64)
				ticker.Last, _ = strconv.ParseFloat(item.LastPrice, 64)
				ticker.Open, _ = strconv.ParseFloat(item.PrevPrice24h, 64)
				ticker.High, _ = strconv.ParseFloat(item.HighPrice24h, 64)
				ticker.Low, _ = strconv.ParseFloat(item.LowPrice24h, 64)
				ticker.Volume, _ = strconv.ParseFloat(item.Volume24h, 64)
				ticker.QuoteVolume, _ = strconv.ParseFloat(item.Turnover24h, 64)
				changePercent, _ := strconv.ParseFloat(item.Price24hPcnt, 64)
				ticker.ChangePercent = changePercent * 100
				if ticker.Open > 0 {
					ticker.Change = ticker.Last - ticker.Open
				}
				tickers[normalizedSymbol] = ticker
			}
		}
	}

	// 获取合约 tickers
	resp, err = b.client.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"category": "linear",
	})
	if err == nil {
		var result struct {
			RetCode int    `json:"retCode"`
			RetMsg  string `json:"retMsg"`
			Result  struct {
				List []struct {
					Symbol        string `json:"symbol"`
					Bid1Price     string `json:"bid1Price"`
					Ask1Price     string `json:"ask1Price"`
					LastPrice     string `json:"lastPrice"`
					PrevPrice24h  string `json:"prevPrice24h"`
					HighPrice24h  string `json:"highPrice24h"`
					LowPrice24h   string `json:"lowPrice24h"`
					Volume24h     string `json:"volume24h"`
					Turnover24h   string `json:"turnover24h"`
					Price24hPcnt  string `json:"price24hPcnt"`
				} `json:"list"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp, &result); err == nil && result.RetCode == 0 {
			for _, item := range result.Result.List {
				market, err := b.GetMarketByID(item.Symbol)
				if err != nil {
					continue
				}
				normalizedSymbol := market.Symbol

				if len(symbols) > 0 {
					found := false
					for _, s := range symbols {
						if s == normalizedSymbol {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				ticker := &types.Ticker{
					Symbol:    normalizedSymbol,
					Timestamp: time.Now(),
				}
				ticker.Bid, _ = strconv.ParseFloat(item.Bid1Price, 64)
				ticker.Ask, _ = strconv.ParseFloat(item.Ask1Price, 64)
				ticker.Last, _ = strconv.ParseFloat(item.LastPrice, 64)
				ticker.Open, _ = strconv.ParseFloat(item.PrevPrice24h, 64)
				ticker.High, _ = strconv.ParseFloat(item.HighPrice24h, 64)
				ticker.Low, _ = strconv.ParseFloat(item.LowPrice24h, 64)
				ticker.Volume, _ = strconv.ParseFloat(item.Volume24h, 64)
				ticker.QuoteVolume, _ = strconv.ParseFloat(item.Turnover24h, 64)
				changePercent, _ := strconv.ParseFloat(item.Price24hPcnt, 64)
				ticker.ChangePercent = changePercent * 100
				if ticker.Open > 0 {
					ticker.Change = ticker.Last - ticker.Open
				}
				tickers[normalizedSymbol] = ticker
			}
		}
	}

	return tickers, nil
}

// FetchOHLCV 获取K线数据
func (b *Bybit) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.BybitTimeframe(timeframe)
	if normalizedTimeframe == "" {
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	// 确定 category
	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	params := map[string]interface{}{
		"symbol":   market.ID,
		"category": category,
		"interval": normalizedTimeframe,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["start"] = since.UnixMilli()
	}

	resp, err := b.client.Get(ctx, "/v5/market/kline", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string     `json:"category"`
			List     [][]string `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	ohlcvs := make(types.OHLCVs, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		if len(item) < 6 {
			continue
		}

		ohlcv := types.OHLCV{}
		ts, _ := strconv.ParseInt(item[0], 10, 64)
		ohlcv.Timestamp = time.UnixMilli(ts)
		ohlcv.Open, _ = strconv.ParseFloat(item[1], 64)
		ohlcv.High, _ = strconv.ParseFloat(item[2], 64)
		ohlcv.Low, _ = strconv.ParseFloat(item[3], 64)
		ohlcv.Close, _ = strconv.ParseFloat(item[4], 64)
		ohlcv.Volume, _ = strconv.ParseFloat(item[5], 64)

		ohlcvs = append(ohlcvs, ohlcv)
	}

	return ohlcvs, nil
}

// 以下方法需要实现但暂时返回错误，后续可以完善
func (b *Bybit) FetchBalance(ctx context.Context) (types.Balances, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price float64, params map[string]interface{}) (*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return fmt.Errorf("not implemented")
}

func (b *Bybit) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *Bybit) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return fmt.Errorf("not implemented")
}

func (b *Bybit) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return fmt.Errorf("not implemented")
}

func (b *Bybit) GetMarkets(ctx context.Context, marketType types.MarketType) ([]*types.Market, error) {
	markets := make([]*types.Market, 0)
	for _, market := range b.GetMarketsMap() {
		if marketType == "" || market.Type == marketType {
			markets = append(markets, market)
		}
	}
	return markets, nil
}


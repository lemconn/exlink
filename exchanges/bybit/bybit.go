package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lemconn/exlink/base"
	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/types"
)

const (
	bybitName       = "bybit"
	bybitBaseURL    = "https://api.bybit.com"
	bybitSandboxURL = "https://api-demo.bybit.com"
)

// Bybit Bybit交易所实现
type Bybit struct {
	*base.BaseExchange
	client    *common.HTTPClient
	apiKey    string
	secretKey string
}

// NewBybit 创建Bybit交易所实例
func NewBybit(apiKey, secretKey string, options map[string]interface{}) (base.Exchange, error) {
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
		BaseExchange: base.NewBaseExchange(bybitName),
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

	// 设置调试模式
	if v, ok := options["debug"].(bool); ok && v {
		exchange.client.SetDebug(true)
	}

	// 设置请求头
	if apiKey != "" {
		exchange.client.SetHeader("X-BAPI-API-KEY", apiKey)
	}

	// 设置其他选项
	for k, v := range options {
		if k != "baseURL" && k != "sandbox" && k != "proxy" && k != "debug" {
			exchange.SetOption(k, v)
		}
	}

	return exchange, nil
}

// LoadMarkets 加载市场信息
func (b *Bybit) LoadMarkets(ctx context.Context, reload bool) error {
	markets := make([]*types.Market, 0)

	// 获取要加载的市场类型
	fetchMarketsTypes := []types.MarketType{types.MarketTypeSpot}
	if v, ok := b.GetOption("fetchMarkets").([]types.MarketType); ok && len(v) > 0 {
		fetchMarketsTypes = v
	} else if v, ok := b.GetOption("fetchMarkets").([]string); ok && len(v) > 0 {
		// 向后兼容：支持字符串数组
		fetchMarketsTypes = make([]types.MarketType, len(v))
		for i, s := range v {
			fetchMarketsTypes[i] = types.MarketType(s)
		}
	} else if v, ok := b.GetOption("fetchMarkets").(string); ok {
		// 向后兼容：支持单个字符串
		fetchMarketsTypes = []types.MarketType{types.MarketType(v)}
	}

	// 加载现货市场
	if containsMarketType(fetchMarketsTypes, types.MarketTypeSpot) {
		spotMarkets, err := b.loadSpotMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load spot markets: %w", err)
		}
		markets = append(markets, spotMarkets...)
	}

	// 加载永续合约市场（linear）
	if containsMarketType(fetchMarketsTypes, types.MarketTypeSwap) || containsMarketType(fetchMarketsTypes, types.MarketTypeFuture) {
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
				Symbol        string `json:"symbol"`
				BaseCoin      string `json:"baseCoin"`
				QuoteCoin     string `json:"quoteCoin"`
				Status        string `json:"status"`
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
			ID:     s.Symbol,         // Bybit 原始格式 (BTCUSDT)
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
				Symbol        string `json:"symbol"`
				BaseCoin      string `json:"baseCoin"`
				QuoteCoin     string `json:"quoteCoin"`
				Status        string `json:"status"`
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

// containsMarketType 检查 MarketType 切片是否包含指定值
func containsMarketType(slice []types.MarketType, item types.MarketType) bool {
	for _, mt := range slice {
		if mt == item {
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
	return nil, base.ErrMarketNotFound
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
				Symbol       string `json:"symbol"`
				Bid1Price    string `json:"bid1Price"`
				Ask1Price    string `json:"ask1Price"`
				LastPrice    string `json:"lastPrice"`
				PrevPrice24h string `json:"prevPrice24h"`
				HighPrice24h string `json:"highPrice24h"`
				LowPrice24h  string `json:"lowPrice24h"`
				Volume24h    string `json:"volume24h"`
				Turnover24h  string `json:"turnover24h"`
				Price24hPcnt string `json:"price24hPcnt"`
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

	ticker.Bid = item.Bid1Price
	ticker.Ask = item.Ask1Price
	ticker.Last = item.LastPrice
	ticker.Open = item.PrevPrice24h
	ticker.High = item.HighPrice24h
	ticker.Low = item.LowPrice24h
	ticker.Volume = item.Volume24h
	ticker.QuoteVolume = item.Turnover24h

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
					Symbol       string `json:"symbol"`
					Bid1Price    string `json:"bid1Price"`
					Ask1Price    string `json:"ask1Price"`
					LastPrice    string `json:"lastPrice"`
					PrevPrice24h string `json:"prevPrice24h"`
					HighPrice24h string `json:"highPrice24h"`
					LowPrice24h  string `json:"lowPrice24h"`
					Volume24h    string `json:"volume24h"`
					Turnover24h  string `json:"turnover24h"`
					Price24hPcnt string `json:"price24hPcnt"`
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
				ticker.Bid = item.Bid1Price
				ticker.Ask = item.Ask1Price
				ticker.Last = item.LastPrice
				ticker.Open = item.PrevPrice24h
				ticker.High = item.HighPrice24h
				ticker.Low = item.LowPrice24h
				ticker.Volume = item.Volume24h
				ticker.QuoteVolume = item.Turnover24h
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
					Symbol       string `json:"symbol"`
					Bid1Price    string `json:"bid1Price"`
					Ask1Price    string `json:"ask1Price"`
					LastPrice    string `json:"lastPrice"`
					PrevPrice24h string `json:"prevPrice24h"`
					HighPrice24h string `json:"highPrice24h"`
					LowPrice24h  string `json:"lowPrice24h"`
					Volume24h    string `json:"volume24h"`
					Turnover24h  string `json:"turnover24h"`
					Price24hPcnt string `json:"price24hPcnt"`
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
				ticker.Bid = item.Bid1Price
				ticker.Ask = item.Ask1Price
				ticker.Last = item.LastPrice
				ticker.Open = item.PrevPrice24h
				ticker.High = item.HighPrice24h
				ticker.Low = item.LowPrice24h
				ticker.Volume = item.Volume24h
				ticker.QuoteVolume = item.Turnover24h
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

// signRequest Bybit 签名方法 (v5 API)
func (b *Bybit) signRequest(method, path string, params map[string]interface{}, body map[string]interface{}) (string, string) {
	timestamp := strconv.FormatInt(common.GetTimestamp(), 10)
	recvWindow := "5000" // 默认接收窗口

	// 构建查询字符串
	queryString := ""
	if len(params) > 0 {
		queryString = common.BuildQueryString(params)
	}

	// 构建请求体
	bodyStr := ""
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyStr = string(bodyBytes)
	}

	// Bybit v5 签名格式: timestamp + apiKey + recvWindow + (body or queryString)
	authBase := timestamp + b.apiKey + recvWindow
	var authFull string
	if method == "POST" {
		authFull = authBase + bodyStr
	} else {
		authFull = authBase + queryString
	}

	signature := common.SignHMAC256(authFull, b.secretKey)
	return signature, timestamp
}

// signAndRequest 签名并发送请求
func (b *Bybit) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	signature, timestamp := b.signRequest(method, path, params, body)
	recvWindow := "5000"

	// 设置请求头
	b.client.SetHeader("X-BAPI-API-KEY", b.apiKey)
	b.client.SetHeader("X-BAPI-TIMESTAMP", timestamp)
	b.client.SetHeader("X-BAPI-RECV-WINDOW", recvWindow)
	b.client.SetHeader("X-BAPI-SIGN", signature)
	b.client.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return b.client.Get(ctx, path, params)
	} else {
		return b.client.Post(ctx, path, body)
	}
}

// FetchBalance 获取余额
func (b *Bybit) FetchBalance(ctx context.Context) (types.Balances, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	// Bybit v5 统一账户余额
	resp, err := b.signAndRequest(ctx, "GET", "/v5/account/wallet-balance", map[string]interface{}{
		"accountType": "UNIFIED",
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Coin []struct {
					Coin                string `json:"coin"`
					Equity              string `json:"equity"`
					AvailableToWithdraw string `json:"availableToWithdraw"`
					WalletBalance       string `json:"walletBalance"`
				} `json:"coin"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	balances := make(types.Balances)
	if len(result.Result.List) > 0 {
		for _, coin := range result.Result.List[0].Coin {
			equity, _ := strconv.ParseFloat(coin.Equity, 64)
			available, _ := strconv.ParseFloat(coin.AvailableToWithdraw, 64)
			walletBalance, _ := strconv.ParseFloat(coin.WalletBalance, 64)

			balances[coin.Coin] = &types.Balance{
				Currency:  coin.Coin,
				Total:     equity,
				Free:      available,
				Used:      walletBalance - available,
				Available: available,
			}
		}
	}

	return balances, nil
}

// CreateOrder 创建订单
func (b *Bybit) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price string, params map[string]interface{}) (*types.Order, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	// 解析 amount 和 price 字符串为 float64 用于计算
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	var priceFloat float64
	if price != "" {
		priceFloat, err = strconv.ParseFloat(price, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
	}

	// 确定 category
	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	// Bybit API requires "Buy" or "Sell" (capitalized)
	sideStr := string(side)
	if len(sideStr) > 0 {
		sideStr = strings.ToUpper(sideStr[:1]) + strings.ToLower(sideStr[1:])
	}

	reqBody := map[string]interface{}{
		"category": category,
		"symbol":   bybitSymbol,
		"side":     sideStr,
	}

	// For spot market buy orders, Bybit requires marketUnit and qty should be the cost in quote currency
	if market.Type == types.MarketTypeSpot && orderType == types.OrderTypeMarket && side == types.OrderSideBuy {
		// Calculate cost: amount * price (use current ask price if price not provided)
		cost := amountFloat
		if priceFloat > 0 {
			cost = amountFloat * priceFloat
		} else {
			// Fetch current price to calculate cost
			ticker, err := b.FetchTicker(ctx, symbol)
			if err == nil && ticker.Ask != "" {
				if askPrice, parseErr := strconv.ParseFloat(ticker.Ask, 64); parseErr == nil && askPrice > 0 {
					cost = amountFloat * askPrice
				}
			}
		}
		reqBody["marketUnit"] = "quoteCoin"
		// Format cost with appropriate precision (use price precision for quote currency)
		precision := market.Precision.Price
		if precision <= 0 {
			precision = 8 // Default precision
		}
		reqBody["qty"] = strconv.FormatFloat(cost, 'f', precision, 64)
		reqBody["orderType"] = "Market"
	} else {
		// For other orders, qty is the amount in base currency
		// Format amount with appropriate precision
		precision := market.Precision.Amount
		if precision <= 0 {
			precision = 8 // Default precision
		}
		reqBody["qty"] = strconv.FormatFloat(amountFloat, 'f', precision, 64)
		if orderType == types.OrderTypeLimit {
			reqBody["orderType"] = "Limit"
			pricePrecision := market.Precision.Price
			if pricePrecision <= 0 {
				pricePrecision = 8 // Default precision
			}
			reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', pricePrecision, 64)
			reqBody["timeInForce"] = "GTC"
		} else {
			reqBody["orderType"] = "Market"
		}
	}

	// 生成客户端订单ID（如果未提供）
	// Bybit 使用 orderLinkId 参数，也支持 clientOrderId 作为别名
	if _, hasOrderLinkId := params["orderLinkId"]; !hasOrderLinkId {
		if clientOrderId, hasClientOrderId := params["clientOrderId"]; hasClientOrderId {
			// 如果用户提供了 clientOrderId，使用它
			reqBody["orderLinkId"] = clientOrderId
		} else {
			// 如果都没有提供，自动生成
			reqBody["orderLinkId"] = common.GenerateClientOrderID(b.Name(), side)
		}
	}

	// 合并额外参数（排除已处理的参数）
	for k, v := range params {
		if k != "clientOrderId" {
			reqBody[k] = v
		}
	}

	resp, err := b.signAndRequest(ctx, "POST", "/v5/order/create", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			OrderID     string `json:"orderId"`
			OrderLinkID string `json:"orderLinkId"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	order := &types.Order{
		ID:            result.Result.OrderID,
		ClientOrderID: result.Result.OrderLinkID,
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Amount:        amountFloat,
		Price:         priceFloat,
		Timestamp:     time.Now(),
		Status:        types.OrderStatusNew,
	}

	return order, nil
}

// CancelOrder 取消订单
func (b *Bybit) CancelOrder(ctx context.Context, orderID, symbol string) error {
	if b.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return err
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return fmt.Errorf("get market ID: %w", err)
	}

	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	reqBody := map[string]interface{}{
		"category": category,
		"symbol":   bybitSymbol,
		"orderId":  orderID,
	}

	_, err = b.signAndRequest(ctx, "POST", "/v5/order/cancel", nil, reqBody)
	return err
}

// FetchOrder 查询订单
func (b *Bybit) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   bybitSymbol,
		"orderId":  orderID,
	}

	// First try to fetch from open orders (realtime)
	resp, err := b.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
	if err == nil {
		var realtimeResult struct {
			RetCode int    `json:"retCode"`
			RetMsg  string `json:"retMsg"`
			Result  struct {
				List []struct {
					OrderID     string `json:"orderId"`
					OrderLinkID string `json:"orderLinkId"`
					OrderStatus string `json:"orderStatus"`
					Side        string `json:"side"`
					OrderType   string `json:"orderType"`
					Price       string `json:"price"`
					Qty         string `json:"qty"`
					CumExecQty  string `json:"cumExecQty"`
					CreatedTime string `json:"createdTime"`
				} `json:"list"`
			} `json:"result"`
		}

		if err := json.Unmarshal(resp, &realtimeResult); err == nil && realtimeResult.RetCode == 0 {
			for _, item := range realtimeResult.Result.List {
				if item.OrderID == orderID {
					return b.parseOrder(item, symbol), nil
				}
			}
		}
	}

	// If not found in open orders, try history
	resp, err = b.signAndRequest(ctx, "GET", "/v5/order/history", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OrderID     string `json:"orderId"`
				OrderLinkID string `json:"orderLinkId"`
				OrderStatus string `json:"orderStatus"`
				Side        string `json:"side"`
				OrderType   string `json:"orderType"`
				Price       string `json:"price"`
				Qty         string `json:"qty"`
				CumExecQty  string `json:"cumExecQty"`
				CreatedTime string `json:"createdTime"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	if len(result.Result.List) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	// Find the order by ID
	for _, item := range result.Result.List {
		if item.OrderID == orderID {
			return b.parseOrder(item, symbol), nil
		}
	}

	return nil, fmt.Errorf("order not found")
}

// parseOrder 解析订单数据
func (b *Bybit) parseOrder(item struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	OrderStatus string `json:"orderStatus"`
	Side        string `json:"side"`
	OrderType   string `json:"orderType"`
	Price       string `json:"price"`
	Qty         string `json:"qty"`
	CumExecQty  string `json:"cumExecQty"`
	CreatedTime string `json:"createdTime"`
}, symbol string) *types.Order {
	order := &types.Order{
		ID:            item.OrderID,
		ClientOrderID: item.OrderLinkID,
		Symbol:        symbol,
		Timestamp:     time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(item.Price, 64)
	order.Amount, _ = strconv.ParseFloat(item.Qty, 64)
	order.Filled, _ = strconv.ParseFloat(item.CumExecQty, 64)
	order.Remaining = order.Amount - order.Filled

	if strings.ToUpper(item.Side) == "BUY" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToUpper(item.OrderType) == "MARKET" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	switch item.OrderStatus {
	case "New":
		order.Status = types.OrderStatusNew
	case "PartiallyFilled":
		order.Status = types.OrderStatusPartiallyFilled
	case "Filled":
		order.Status = types.OrderStatusFilled
	case "Cancelled":
		order.Status = types.OrderStatusCanceled
	default:
		order.Status = types.OrderStatusNew
	}

	return order

}

// FetchOrders 查询订单列表
func (b *Bybit) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Bybit v5 没有直接的 fetchOrders，需要通过其他端点组合
	return nil, fmt.Errorf("not implemented: use FetchOpenOrders or implement custom logic")
}

// FetchOpenOrders 查询未成交订单
func (b *Bybit) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   bybitSymbol,
	}

	resp, err := b.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OrderID     string `json:"orderId"`
				OrderLinkID string `json:"orderLinkId"`
				OrderStatus string `json:"orderStatus"`
				Side        string `json:"side"`
				OrderType   string `json:"orderType"`
				Price       string `json:"price"`
				Qty         string `json:"qty"`
				CumExecQty  string `json:"cumExecQty"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	orders := make([]*types.Order, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		order := &types.Order{
			ID:            item.OrderID,
			ClientOrderID: item.OrderLinkID,
			Symbol:        symbol,
			Timestamp:     time.Now(),
		}

		order.Price, _ = strconv.ParseFloat(item.Price, 64)
		order.Amount, _ = strconv.ParseFloat(item.Qty, 64)
		order.Filled, _ = strconv.ParseFloat(item.CumExecQty, 64)
		order.Remaining = order.Amount - order.Filled

		if strings.ToUpper(item.Side) == "BUY" {
			order.Side = types.OrderSideBuy
		} else {
			order.Side = types.OrderSideSell
		}

		if strings.ToUpper(item.OrderType) == "MARKET" {
			order.Type = types.OrderTypeMarket
		} else {
			order.Type = types.OrderTypeLimit
		}

		switch item.OrderStatus {
		case "New":
			order.Status = types.OrderStatusNew
		case "PartiallyFilled":
			order.Status = types.OrderStatusPartiallyFilled
		default:
			order.Status = types.OrderStatusOpen
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// FetchTrades 获取交易记录
func (b *Bybit) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   bybitSymbol,
		"limit":    limit,
	}

	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := b.client.Get(ctx, "/v5/market/recent-trade", params)
	if err != nil {
		return nil, fmt.Errorf("fetch trades: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				ExecTime string `json:"execTime"`
				Symbol   string `json:"symbol"`
				Price    string `json:"price"`
				Size     string `json:"size"`
				Side     string `json:"side"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	trades := make([]*types.Trade, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		trade := &types.Trade{
			Symbol:    symbol,
			Timestamp: time.Now(),
		}

		trade.Price, _ = strconv.ParseFloat(item.Price, 64)
		trade.Amount, _ = strconv.ParseFloat(item.Size, 64)

		if strings.ToUpper(item.Side) == "BUY" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchMyTrades 获取我的交易记录
func (b *Bybit) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	category := "spot"
	if market.Contract && market.Linear {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   bybitSymbol,
		"limit":    limit,
	}

	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := b.signAndRequest(ctx, "GET", "/v5/execution/list", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				ExecTime string `json:"execTime"`
				Symbol   string `json:"symbol"`
				Price    string `json:"execPrice"`
				Size     string `json:"execQty"`
				Side     string `json:"side"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	trades := make([]*types.Trade, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		trade := &types.Trade{
			Symbol:    symbol,
			Timestamp: time.Now(),
		}

		trade.Price, _ = strconv.ParseFloat(item.Price, 64)
		trade.Amount, _ = strconv.ParseFloat(item.Size, 64)

		if strings.ToUpper(item.Side) == "BUY" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchPositions 获取持仓
func (b *Bybit) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	params := map[string]interface{}{
		"category": "linear",
	}

	if len(symbols) > 0 {
		// Bybit 需要 symbol 参数
		bybitSymbol, err := b.GetMarketID(symbols[0])
		if err == nil {
			params["symbol"] = bybitSymbol
		}
	}

	resp, err := b.signAndRequest(ctx, "GET", "/v5/position/list", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Symbol        string `json:"symbol"`
				Side          string `json:"side"`
				Size          string `json:"size"`
				EntryPrice    string `json:"avgPrice"`
				MarkPrice     string `json:"markPrice"`
				UnrealisedPnl string `json:"unrealisedPnl"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	positions := make([]*types.Position, 0)
	for _, item := range result.Result.List {
		size, _ := strconv.ParseFloat(item.Size, 64)
		if size == 0 {
			continue
		}

		market, err := b.GetMarketByID(item.Symbol)
		if err != nil {
			continue
		}

		if len(symbols) > 0 {
			found := false
			for _, s := range symbols {
				if s == market.Symbol {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		position := &types.Position{
			Symbol:    market.Symbol,
			Amount:    size,
			Timestamp: time.Now(),
		}

		position.EntryPrice, _ = strconv.ParseFloat(item.EntryPrice, 64)
		position.MarkPrice, _ = strconv.ParseFloat(item.MarkPrice, 64)
		position.UnrealizedPnl, _ = strconv.ParseFloat(item.UnrealisedPnl, 64)

		if strings.ToUpper(item.Side) == "BUY" {
			position.Side = types.PositionSideLong
		} else {
			position.Side = types.PositionSideShort
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// SetLeverage 设置杠杆
func (b *Bybit) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if b.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("leverage only supported for contracts")
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return fmt.Errorf("get market ID: %w", err)
	}

	reqBody := map[string]interface{}{
		"category":     "linear",
		"symbol":       bybitSymbol,
		"buyLeverage":  strconv.Itoa(leverage),
		"sellLeverage": strconv.Itoa(leverage),
	}

	_, err = b.signAndRequest(ctx, "POST", "/v5/position/set-leverage", nil, reqBody)
	return err
}

// SetMarginMode 设置保证金模式
func (b *Bybit) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	if b.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("margin mode only supported for contracts")
	}

	// 验证模式
	if mode != "isolated" && mode != "cross" {
		return fmt.Errorf("invalid margin mode: %s, must be 'isolated' or 'cross'", mode)
	}

	bybitSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return fmt.Errorf("get market ID: %w", err)
	}

	reqBody := map[string]interface{}{
		"category":  "linear",
		"symbol":    bybitSymbol,
		"tradeMode": strings.ToUpper(mode),
	}

	_, err = b.signAndRequest(ctx, "POST", "/v5/position/switch-mode", nil, reqBody)
	return err
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

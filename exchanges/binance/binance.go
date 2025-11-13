package binance

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
	binanceName           = "binance"
	binanceBaseURL        = "https://api.binance.com"
	binanceSandboxURL     = "https://testnet.binance.vision"
	binanceFapiBaseURL    = "https://fapi.binance.com"
	binanceFapiSandboxURL = "https://testnet.binancefuture.com"
)

// Binance Binance交易所实现
type Binance struct {
	*base.BaseExchange
	client     *common.HTTPClient
	fapiClient *common.HTTPClient // 永续合约API客户端
	apiKey     string
	secretKey  string
}

// NewBinance 创建Binance交易所实例
func NewBinance(apiKey, secretKey string, options map[string]interface{}) (base.Exchange, error) {
	baseURL := binanceBaseURL
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
		baseURL = binanceSandboxURL
	}

	fapiBaseURL := binanceFapiBaseURL
	if sandbox {
		fapiBaseURL = binanceFapiSandboxURL
	}

	exchange := &Binance{
		BaseExchange: base.NewBaseExchange(binanceName),
		client:       common.NewHTTPClient(baseURL),
		fapiClient:   common.NewHTTPClient(fapiBaseURL),
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
		if err := exchange.fapiClient.SetProxy(proxyURL); err != nil {
			return nil, fmt.Errorf("set fapi proxy: %w", err)
		}
	}

	// 设置调试模式
	if v, ok := options["debug"].(bool); ok && v {
		exchange.client.SetDebug(true)
		exchange.fapiClient.SetDebug(true)
	}

	// 设置请求头
	if apiKey != "" {
		exchange.client.SetHeader("X-MBX-APIKEY", apiKey)
		exchange.fapiClient.SetHeader("X-MBX-APIKEY", apiKey)
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
func (b *Binance) LoadMarkets(ctx context.Context, reload bool) error {
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

	// 加载永续合约市场（U本位）
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
func (b *Binance) loadSpotMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := b.client.Get(ctx, "/api/v3/exchangeInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch exchange info: %w", err)
	}

	var info struct {
		Symbols []struct {
			Symbol     string `json:"symbol"`
			BaseAsset  string `json:"baseAsset"`
			QuoteAsset string `json:"quoteAsset"`
			Status     string `json:"status"`
			Filters    []struct {
				FilterType  string `json:"filterType"`
				MinQty      string `json:"minQty,omitempty"`
				MaxQty      string `json:"maxQty,omitempty"`
				StepSize    string `json:"stepSize,omitempty"`
				MinPrice    string `json:"minPrice,omitempty"`
				MaxPrice    string `json:"maxPrice,omitempty"`
				TickSize    string `json:"tickSize,omitempty"`
				MinNotional string `json:"minNotional,omitempty"`
			} `json:"filters"`
			BaseAssetPrecision int `json:"baseAssetPrecision"`
			QuotePrecision     int `json:"quotePrecision"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("unmarshal exchange info: %w", err)
	}

	markets := make([]*types.Market, 0)
	for _, s := range info.Symbols {
		if s.Status != "TRADING" {
			continue
		}

		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(s.BaseAsset, s.QuoteAsset)

		market := &types.Market{
			ID:     s.Symbol,         // Binance 原始格式 (BTCUSDT)
			Symbol: normalizedSymbol, // 标准化格式 (BTC/USDT)
			Base:   s.BaseAsset,
			Quote:  s.QuoteAsset,
			Type:   types.MarketTypeSpot,
			Active: s.Status == "TRADING",
		}

		// 解析精度和限制
		market.Precision.Amount = s.BaseAssetPrecision
		market.Precision.Price = s.QuotePrecision

		for _, filter := range s.Filters {
			switch filter.FilterType {
			case "LOT_SIZE":
				if filter.MinQty != "" {
					market.Limits.Amount.Min, _ = strconv.ParseFloat(filter.MinQty, 64)
				}
				if filter.MaxQty != "" {
					market.Limits.Amount.Max, _ = strconv.ParseFloat(filter.MaxQty, 64)
				}
				if filter.StepSize != "" {
					// 计算精度
					parts := strings.Split(filter.StepSize, ".")
					if len(parts) > 1 {
						market.Precision.Amount = len(strings.TrimRight(parts[1], "0"))
					}
				}
			case "PRICE_FILTER":
				if filter.MinPrice != "" {
					market.Limits.Price.Min, _ = strconv.ParseFloat(filter.MinPrice, 64)
				}
				if filter.MaxPrice != "" {
					market.Limits.Price.Max, _ = strconv.ParseFloat(filter.MaxPrice, 64)
				}
				if filter.TickSize != "" {
					parts := strings.Split(filter.TickSize, ".")
					if len(parts) > 1 {
						market.Precision.Price = len(strings.TrimRight(parts[1], "0"))
					}
				}
			case "MIN_NOTIONAL":
				if filter.MinNotional != "" {
					market.Limits.Cost.Min, _ = strconv.ParseFloat(filter.MinNotional, 64)
				}
			}
		}

		markets = append(markets, market)
	}

	return markets, nil
}

// loadSwapMarkets 加载永续合约市场（U本位）
func (b *Binance) loadSwapMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := b.fapiClient.Get(ctx, "/fapi/v1/exchangeInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch fapi exchange info: %w", err)
	}

	var info struct {
		Symbols []struct {
			Symbol            string `json:"symbol"`
			Pair              string `json:"pair"`
			ContractType      string `json:"contractType"`
			BaseAsset         string `json:"baseAsset"`
			QuoteAsset        string `json:"quoteAsset"`
			MarginAsset       string `json:"marginAsset"`
			Status            string `json:"status"`
			PricePrecision    int    `json:"pricePrecision"`
			QuantityPrecision int    `json:"quantityPrecision"`
			Filters           []struct {
				FilterType  string `json:"filterType"`
				MinQty      string `json:"minQty,omitempty"`
				MaxQty      string `json:"maxQty,omitempty"`
				StepSize    string `json:"stepSize,omitempty"`
				MinPrice    string `json:"minPrice,omitempty"`
				MaxPrice    string `json:"maxPrice,omitempty"`
				TickSize    string `json:"tickSize,omitempty"`
				MinNotional string `json:"minNotional,omitempty"`
			} `json:"filters"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("unmarshal fapi exchange info: %w", err)
	}

	markets := make([]*types.Market, 0)
	for _, s := range info.Symbols {
		// 只处理永续合约
		if s.ContractType != "PERPETUAL" {
			continue
		}
		if s.Status != "TRADING" {
			continue
		}

		settle := s.MarginAsset
		if settle == "" {
			settle = s.QuoteAsset
		}

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(s.BaseAsset, s.QuoteAsset, settle)

		market := &types.Market{
			ID:       s.Symbol,
			Symbol:   normalizedSymbol,
			Base:     s.BaseAsset,
			Quote:    s.QuoteAsset,
			Settle:   settle,
			Type:     types.MarketTypeSwap,
			Active:   s.Status == "TRADING",
			Contract: true,
			Linear:   true, // U本位永续合约
			Inverse:  false,
		}

		// 解析精度
		market.Precision.Amount = s.QuantityPrecision
		market.Precision.Price = s.PricePrecision

		// 解析限制
		for _, filter := range s.Filters {
			switch filter.FilterType {
			case "LOT_SIZE":
				if filter.MinQty != "" {
					market.Limits.Amount.Min, _ = strconv.ParseFloat(filter.MinQty, 64)
				}
				if filter.MaxQty != "" {
					market.Limits.Amount.Max, _ = strconv.ParseFloat(filter.MaxQty, 64)
				}
				if filter.StepSize != "" {
					parts := strings.Split(filter.StepSize, ".")
					if len(parts) > 1 {
						market.Precision.Amount = len(strings.TrimRight(parts[1], "0"))
					}
				}
			case "PRICE_FILTER":
				if filter.MinPrice != "" {
					market.Limits.Price.Min, _ = strconv.ParseFloat(filter.MinPrice, 64)
				}
				if filter.MaxPrice != "" {
					market.Limits.Price.Max, _ = strconv.ParseFloat(filter.MaxPrice, 64)
				}
				if filter.TickSize != "" {
					parts := strings.Split(filter.TickSize, ".")
					if len(parts) > 1 {
						market.Precision.Price = len(strings.TrimRight(parts[1], "0"))
					}
				}
			case "MIN_NOTIONAL":
				if filter.MinNotional != "" {
					market.Limits.Cost.Min, _ = strconv.ParseFloat(filter.MinNotional, 64)
				}
			}
		}

		markets = append(markets, market)
	}

	return markets, nil
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

// containsMarketType 检查 MarketType 切片是否包含指定值
func containsMarketType(slice []types.MarketType, item types.MarketType) bool {
	for _, mt := range slice {
		if mt == item {
			return true
		}
	}
	return false
}

// FetchTicker 获取行情
func (b *Binance) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// 获取市场信息以判断使用哪个API
	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	var resp []byte
	var apiErr error
	if market.Contract && market.Linear {
		// 永续合约（U本位）- 使用fapi
		resp, apiErr = b.fapiClient.Get(ctx, "/fapi/v1/ticker/24hr", map[string]interface{}{
			"symbol": binanceSymbol,
		})
	} else {
		// 现货
		resp, apiErr = b.client.Get(ctx, "/api/v3/ticker/24hr", map[string]interface{}{
			"symbol": binanceSymbol,
		})
	}

	if apiErr != nil {
		return nil, fmt.Errorf("fetch ticker: %w", apiErr)
	}

	var data struct {
		Symbol             string `json:"symbol"`
		BidPrice           string `json:"bidPrice"`
		AskPrice           string `json:"askPrice"`
		LastPrice          string `json:"lastPrice"`
		OpenPrice          string `json:"openPrice"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
		PriceChange        string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	// 转换回标准化格式 - 使用输入的symbol（已经是标准化格式）
	ticker := &types.Ticker{
		Symbol:    symbol, // 使用输入的标准化格式
		Timestamp: time.Now(),
	}

	ticker.Bid, _ = strconv.ParseFloat(data.BidPrice, 64)
	ticker.Ask, _ = strconv.ParseFloat(data.AskPrice, 64)
	ticker.Last, _ = strconv.ParseFloat(data.LastPrice, 64)
	ticker.Open, _ = strconv.ParseFloat(data.OpenPrice, 64)
	ticker.High, _ = strconv.ParseFloat(data.HighPrice, 64)
	ticker.Low, _ = strconv.ParseFloat(data.LowPrice, 64)
	ticker.Volume, _ = strconv.ParseFloat(data.Volume, 64)
	ticker.QuoteVolume, _ = strconv.ParseFloat(data.QuoteVolume, 64)
	ticker.Change, _ = strconv.ParseFloat(data.PriceChange, 64)
	ticker.ChangePercent, _ = strconv.ParseFloat(data.PriceChangePercent, 64)

	return ticker, nil
}

// FetchTickers 批量获取行情
func (b *Binance) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	resp, err := b.client.Get(ctx, "/api/v3/ticker/24hr", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data []struct {
		Symbol             string `json:"symbol"`
		BidPrice           string `json:"bidPrice"`
		AskPrice           string `json:"askPrice"`
		LastPrice          string `json:"lastPrice"`
		OpenPrice          string `json:"openPrice"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
		PriceChange        string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	tickers := make(map[string]*types.Ticker)
	for _, item := range data {
		if len(symbols) > 0 {
			found := false
			for _, s := range symbols {
				if s == item.Symbol {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		ticker := &types.Ticker{
			Symbol:    item.Symbol,
			Timestamp: time.Now(),
		}

		ticker.Bid, _ = strconv.ParseFloat(item.BidPrice, 64)
		ticker.Ask, _ = strconv.ParseFloat(item.AskPrice, 64)
		ticker.Last, _ = strconv.ParseFloat(item.LastPrice, 64)
		ticker.Open, _ = strconv.ParseFloat(item.OpenPrice, 64)
		ticker.High, _ = strconv.ParseFloat(item.HighPrice, 64)
		ticker.Low, _ = strconv.ParseFloat(item.LowPrice, 64)
		ticker.Volume, _ = strconv.ParseFloat(item.Volume, 64)
		ticker.QuoteVolume, _ = strconv.ParseFloat(item.QuoteVolume, 64)
		ticker.Change, _ = strconv.ParseFloat(item.PriceChange, 64)
		ticker.ChangePercent, _ = strconv.ParseFloat(item.PriceChangePercent, 64)

		tickers[item.Symbol] = ticker
	}

	return tickers, nil
}

// FetchOHLCV 获取K线数据
func (b *Binance) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// 获取市场信息以判断使用哪个API
	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.BinanceTimeframe(timeframe)

	params := map[string]interface{}{
		"interval": normalizedTimeframe,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	// 获取交易所格式的 symbol ID（优先使用 market.ID）
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		// 如果 market.ID 为空，使用后备转换函数
		var err error
		binanceSymbol, err = common.ToBinanceSymbol(symbol)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}
	params["symbol"] = binanceSymbol

	var resp []byte
	var apiErr error
	if market.Contract && market.Linear {
		// 永续合约（U本位）- 使用fapi
		resp, apiErr = b.fapiClient.Get(ctx, "/fapi/v1/klines", params)
	} else {
		// 现货
		resp, apiErr = b.client.Get(ctx, "/api/v3/klines", params)
	}

	if apiErr != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", apiErr)
	}

	var data [][]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	ohlcvs := make(types.OHLCVs, 0, len(data))
	for _, item := range data {
		if len(item) < 6 {
			continue
		}

		ohlcv := types.OHLCV{}
		if ts, ok := item[0].(float64); ok {
			ohlcv.Timestamp = time.UnixMilli(int64(ts))
		}
		if open, ok := item[1].(string); ok {
			ohlcv.Open, _ = strconv.ParseFloat(open, 64)
		}
		if high, ok := item[2].(string); ok {
			ohlcv.High, _ = strconv.ParseFloat(high, 64)
		}
		if low, ok := item[3].(string); ok {
			ohlcv.Low, _ = strconv.ParseFloat(low, 64)
		}
		if close, ok := item[4].(string); ok {
			ohlcv.Close, _ = strconv.ParseFloat(close, 64)
		}
		if volume, ok := item[5].(string); ok {
			ohlcv.Volume, _ = strconv.ParseFloat(volume, 64)
		}

		ohlcvs = append(ohlcvs, ohlcv)
	}

	return ohlcvs, nil
}

// FetchBalance 获取余额
func (b *Binance) FetchBalance(ctx context.Context) (types.Balances, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}

	queryString := common.BuildQueryString(params)
	signature := common.SignHMAC256(queryString, b.secretKey)
	params["signature"] = signature

	resp, err := b.client.Get(ctx, "/api/v3/account", params)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var data struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	balances := make(types.Balances)
	for _, bal := range data.Balances {
		free, _ := strconv.ParseFloat(bal.Free, 64)
		locked, _ := strconv.ParseFloat(bal.Locked, 64)
		total := free + locked

		balances[bal.Asset] = &types.Balance{
			Currency:  bal.Asset,
			Free:      free,
			Used:      locked,
			Total:     total,
			Available: free,
		}
	}

	return balances, nil
}

// CreateOrder 创建订单
func (b *Binance) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price float64, params map[string]interface{}) (*types.Order, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	// 获取市场信息以判断使用哪个API
	market, err := b.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	var binanceSymbol string
	var resp []byte
	var apiErr error

	timestamp := common.GetTimestamp()
	reqParams := map[string]interface{}{
		"side":      strings.ToUpper(string(side)),
		"type":      strings.ToUpper(string(orderType)),
		"timestamp": timestamp,
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol, err = b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}
	reqParams["symbol"] = binanceSymbol
	reqParams["quantity"] = strconv.FormatFloat(amount, 'f', -1, 64)

	if orderType == types.OrderTypeLimit {
		reqParams["price"] = strconv.FormatFloat(price, 'f', -1, 64)
		reqParams["timeInForce"] = "GTC"
	}

	// 合并额外参数
	for k, v := range params {
		reqParams[k] = v
	}

	queryString := common.BuildQueryString(reqParams)
	signature := common.SignHMAC256(queryString, b.secretKey)
	reqParams["signature"] = signature

	if market.Contract && market.Linear {
		// 永续合约订单
		resp, apiErr = b.fapiClient.Post(ctx, "/fapi/v1/order", reqParams)
	} else {
		// 现货订单
		resp, apiErr = b.client.Post(ctx, "/api/v3/order", reqParams)
	}

	if apiErr != nil {
		return nil, fmt.Errorf("create order: %w", apiErr)
	}

	var data struct {
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Price         string `json:"price"`
		Quantity      string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		Time          int64  `json:"transactTime"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	order := &types.Order{
		ID:            strconv.FormatInt(data.OrderID, 10),
		ClientOrderID: data.ClientOrderID,
		Symbol:        symbol, // 使用标准化格式
		Type:          orderType,
		Side:          side,
		Amount:        amount,
		Timestamp:     time.UnixMilli(data.Time),
	}

	order.Price, _ = strconv.ParseFloat(data.Price, 64)
	order.Filled, _ = strconv.ParseFloat(data.ExecutedQty, 64)
	order.Remaining = amount - order.Filled

	// 转换状态
	switch data.Status {
	case "NEW":
		order.Status = types.OrderStatusNew
	case "PARTIALLY_FILLED":
		order.Status = types.OrderStatusPartiallyFilled
	case "FILLED":
		order.Status = types.OrderStatusFilled
	case "CANCELED":
		order.Status = types.OrderStatusCanceled
	case "EXPIRED":
		order.Status = types.OrderStatusExpired
	case "REJECTED":
		order.Status = types.OrderStatusRejected
	}

	return order, nil
}

// CancelOrder 取消订单
func (b *Binance) CancelOrder(ctx context.Context, orderID, symbol string) error {
	if b.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return fmt.Errorf("get market ID: %w", err)
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"symbol":    binanceSymbol,
		"orderId":   orderID,
		"timestamp": timestamp,
	}

	queryString := common.BuildQueryString(params)
	signature := common.SignHMAC256(queryString, b.secretKey)
	params["signature"] = signature

	_, err = b.client.Post(ctx, "/api/v3/order", params)
	return err
}

// FetchOrder 查询订单
func (b *Binance) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"symbol":    binanceSymbol,
		"orderId":   orderID,
		"timestamp": timestamp,
	}

	queryString := common.BuildQueryString(params)
	signature := common.SignHMAC256(queryString, b.secretKey)
	params["signature"] = signature

	resp, err := b.client.Get(ctx, "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var data struct {
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Price         string `json:"price"`
		Quantity      string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		Time          int64  `json:"time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	order := &types.Order{
		ID:            strconv.FormatInt(data.OrderID, 10),
		ClientOrderID: data.ClientOrderID,
		Symbol:        symbol, // 使用标准化格式
		Timestamp:     time.UnixMilli(data.Time),
	}

	order.Price, _ = strconv.ParseFloat(data.Price, 64)
	order.Amount, _ = strconv.ParseFloat(data.Quantity, 64)
	order.Filled, _ = strconv.ParseFloat(data.ExecutedQty, 64)
	order.Remaining = order.Amount - order.Filled

	if strings.ToUpper(data.Side) == "BUY" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToUpper(data.Type) == "MARKET" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	switch data.Status {
	case "NEW":
		order.Status = types.OrderStatusNew
	case "PARTIALLY_FILLED":
		order.Status = types.OrderStatusPartiallyFilled
	case "FILLED":
		order.Status = types.OrderStatusFilled
	case "CANCELED":
		order.Status = types.OrderStatusCanceled
	case "EXPIRED":
		order.Status = types.OrderStatusExpired
	case "REJECTED":
		order.Status = types.OrderStatusRejected
	}

	return order, nil
}

// FetchOrders 查询订单列表
func (b *Binance) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// 实现类似FetchOrder的逻辑，但获取多个订单
	return nil, fmt.Errorf("not implemented")
}

// FetchOpenOrders 查询未成交订单
func (b *Binance) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}
	if symbol != "" {
		// 获取交易所格式的 symbol ID
		binanceSymbol, err := b.GetMarketID(symbol)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
		params["symbol"] = binanceSymbol
	}

	queryString := common.BuildQueryString(params)
	signature := common.SignHMAC256(queryString, b.secretKey)
	params["signature"] = signature

	resp, err := b.client.Get(ctx, "/api/v3/openOrders", params)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
	}

	var data []struct {
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Price         string `json:"price"`
		Quantity      string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		Time          int64  `json:"time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	orders := make([]*types.Order, 0, len(data))
	for _, item := range data {
		// 转换回标准化格式
		normalizedSymbol := symbol
		if symbol == "" {
			// 如果没有提供symbol，尝试从市场信息中查找
			normalizedSymbol = item.Symbol // 临时使用原格式
		}

		order := &types.Order{
			ID:            strconv.FormatInt(item.OrderID, 10),
			ClientOrderID: item.ClientOrderID,
			Symbol:        normalizedSymbol,
			Timestamp:     time.UnixMilli(item.Time),
		}

		order.Price, _ = strconv.ParseFloat(item.Price, 64)
		order.Amount, _ = strconv.ParseFloat(item.Quantity, 64)
		order.Filled, _ = strconv.ParseFloat(item.ExecutedQty, 64)
		order.Remaining = order.Amount - order.Filled

		if strings.ToUpper(item.Side) == "BUY" {
			order.Side = types.OrderSideBuy
		} else {
			order.Side = types.OrderSideSell
		}

		if strings.ToUpper(item.Type) == "MARKET" {
			order.Type = types.OrderTypeMarket
		} else {
			order.Type = types.OrderTypeLimit
		}

		switch item.Status {
		case "NEW":
			order.Status = types.OrderStatusNew
		case "PARTIALLY_FILLED":
			order.Status = types.OrderStatusPartiallyFilled
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// FetchTrades 获取交易记录
func (b *Binance) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取交易所格式的 symbol ID
	binanceSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	params := map[string]interface{}{
		"symbol": binanceSymbol,
		"limit":  limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := b.client.Get(ctx, "/api/v3/trades", params)
	if err != nil {
		return nil, fmt.Errorf("fetch trades: %w", err)
	}

	var data []struct {
		ID      int64  `json:"id"`
		Price   string `json:"price"`
		Qty     string `json:"qty"`
		Time    int64  `json:"time"`
		IsBuyer bool   `json:"isBuyerMaker"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	trades := make([]*types.Trade, 0, len(data))
	for _, item := range data {
		price, _ := strconv.ParseFloat(item.Price, 64)
		qty, _ := strconv.ParseFloat(item.Qty, 64)

		trade := &types.Trade{
			ID:        strconv.FormatInt(item.ID, 10),
			Symbol:    symbol, // 使用标准化格式
			Price:     price,
			Amount:    qty,
			Cost:      price * qty,
			Timestamp: time.UnixMilli(item.Time),
		}

		if !item.IsBuyer {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchMyTrades 获取我的交易记录
func (b *Binance) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol, err := b.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"symbol":    binanceSymbol,
		"limit":     limit,
		"timestamp": timestamp,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	queryString := common.BuildQueryString(params)
	signature := common.SignHMAC256(queryString, b.secretKey)
	params["signature"] = signature

	resp, err := b.client.Get(ctx, "/api/v3/myTrades", params)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var data []struct {
		ID              int64  `json:"id"`
		OrderID         int64  `json:"orderId"`
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Time            int64  `json:"time"`
		IsBuyer         bool   `json:"isBuyer"`
		Commission      string `json:"commission"`
		CommissionAsset string `json:"commissionAsset"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	trades := make([]*types.Trade, 0, len(data))
	for _, item := range data {
		price, _ := strconv.ParseFloat(item.Price, 64)
		qty, _ := strconv.ParseFloat(item.Qty, 64)
		commission, _ := strconv.ParseFloat(item.Commission, 64)

		trade := &types.Trade{
			ID:        strconv.FormatInt(item.ID, 10),
			OrderID:   strconv.FormatInt(item.OrderID, 10),
			Symbol:    symbol, // 使用标准化格式
			Price:     price,
			Amount:    qty,
			Cost:      price * qty,
			Timestamp: time.UnixMilli(item.Time),
		}

		if item.IsBuyer {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		if commission > 0 {
			trade.Fee = &types.Fee{
				Currency: item.CommissionAsset,
				Cost:     commission,
			}
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchPositions 获取持仓（合约）
func (b *Binance) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	if b.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}

	queryString := common.BuildQueryString(params)
	signature := common.SignHMAC256(queryString, b.secretKey)
	params["signature"] = signature

	resp, err := b.fapiClient.Get(ctx, "/fapi/v2/positionRisk", params)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var data []struct {
		Symbol           string `json:"symbol"`
		PositionSide     string `json:"positionSide"`
		PositionAmt      string `json:"positionAmt"`
		EntryPrice       string `json:"entryPrice"`
		MarkPrice        string `json:"markPrice"`
		UnRealizedProfit string `json:"unRealizedProfit"`
		LiquidationPrice string `json:"liquidationPrice"`
		Leverage         string `json:"leverage"`
		MarginType       string `json:"marginType"`
		IsolatedMargin   string `json:"isolatedMargin"`
		UpdateTime       int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	positions := make([]*types.Position, 0)
	for _, item := range data {
		positionAmt, _ := strconv.ParseFloat(item.PositionAmt, 64)
		if positionAmt == 0 {
			continue // 跳过空仓
		}

		// 获取市场信息
		market, err := b.GetMarketByID(item.Symbol)
		if err != nil {
			continue
		}

		// 如果指定了symbols，只返回匹配的
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

		entryPrice, _ := strconv.ParseFloat(item.EntryPrice, 64)
		markPrice, _ := strconv.ParseFloat(item.MarkPrice, 64)
		unrealizedPnl, _ := strconv.ParseFloat(item.UnRealizedProfit, 64)
		liquidationPrice, _ := strconv.ParseFloat(item.LiquidationPrice, 64)
		leverage, _ := strconv.ParseFloat(item.Leverage, 64)
		margin, _ := strconv.ParseFloat(item.IsolatedMargin, 64)

		var side types.PositionSide
		if positionAmt > 0 {
			side = types.PositionSideLong
		} else {
			side = types.PositionSideShort
			positionAmt = -positionAmt
		}

		position := &types.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           positionAmt,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			LiquidationPrice: liquidationPrice,
			UnrealizedPnl:    unrealizedPnl,
			Leverage:         leverage,
			Margin:           margin,
			Timestamp:        time.UnixMilli(item.UpdateTime),
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// GetMarketByID 通过交易所ID获取市场信息
func (b *Binance) GetMarketByID(id string) (*types.Market, error) {
	for _, market := range b.GetMarketsMap() {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, base.ErrMarketNotFound
}

// GetMarketID 获取Binance格式的 symbol ID
// 优先从已加载的市场中查找，如果未找到则使用后备转换函数
func (b *Binance) GetMarketID(symbol string) (string, error) {
	// 优先从已加载的市场中查找
	market, ok := b.GetMarketsMap()[symbol]
	if ok {
		return market.ID, nil
	}

	// 如果市场未加载，使用后备转换函数
	return common.ToBinanceSymbol(symbol)
}

// SetLeverage 设置杠杆
func (b *Binance) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if b.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract || !market.Linear {
		return fmt.Errorf("leverage only supported for linear contracts")
	}

	timestamp := common.GetTimestamp()
	reqParams := map[string]interface{}{
		"symbol":    market.ID,
		"leverage":  leverage,
		"timestamp": timestamp,
	}

	queryString := common.BuildQueryString(reqParams)
	signature := common.SignHMAC256(queryString, b.secretKey)
	reqParams["signature"] = signature

	_, err = b.fapiClient.Post(ctx, "/fapi/v1/leverage", reqParams)
	return err
}

// SetMarginMode 设置保证金模式
func (b *Binance) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	if b.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := b.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract || !market.Linear {
		return fmt.Errorf("margin mode only supported for linear contracts")
	}

	// 验证模式
	if mode != "isolated" && mode != "cross" {
		return fmt.Errorf("invalid margin mode: %s, must be 'isolated' or 'cross'", mode)
	}

	timestamp := common.GetTimestamp()
	reqParams := map[string]interface{}{
		"symbol":     market.ID,
		"marginType": strings.ToUpper(mode),
		"timestamp":  timestamp,
	}

	queryString := common.BuildQueryString(reqParams)
	signature := common.SignHMAC256(queryString, b.secretKey)
	reqParams["signature"] = signature

	_, err = b.fapiClient.Post(ctx, "/fapi/v1/marginType", reqParams)
	return err
}

// GetMarkets 获取市场列表
func (b *Binance) GetMarkets(ctx context.Context, marketType types.MarketType) ([]*types.Market, error) {
	if err := b.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	markets := make([]*types.Market, 0)
	for _, market := range b.BaseExchange.GetMarketsMap() {
		if marketType == "" || market.Type == marketType {
			markets = append(markets, market)
		}
	}
	return markets, nil
}

package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lemconn/exlink/base"
	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/types"
)

const (
	gateName       = "gate"
	gateBaseURL    = "https://api.gateio.ws"
	gateSandboxURL = "https://api-testnet.gateapi.io"
)

// Gate Gate交易所实现
type Gate struct {
	*base.BaseExchange
	client    *common.HTTPClient
	apiKey    string
	secretKey string
}

// NewGate 创建Gate交易所实例
func NewGate(apiKey, secretKey string, options map[string]interface{}) (base.Exchange, error) {
	baseURL := gateBaseURL
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
		baseURL = gateSandboxURL
	}

	exchange := &Gate{
		BaseExchange: base.NewBaseExchange(gateName),
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
		exchange.client.SetHeader("X-Gate-Channel-Id", "api")
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
func (g *Gate) LoadMarkets(ctx context.Context, reload bool) error {
	markets := make([]*types.Market, 0)

	// 获取要加载的市场类型
	fetchMarketsTypes := []types.MarketType{types.MarketTypeSpot}
	if v, ok := g.GetOption("fetchMarkets").([]types.MarketType); ok && len(v) > 0 {
		fetchMarketsTypes = v
	} else if v, ok := g.GetOption("fetchMarkets").([]string); ok && len(v) > 0 {
		// 向后兼容：支持字符串数组
		fetchMarketsTypes = make([]types.MarketType, len(v))
		for i, s := range v {
			fetchMarketsTypes[i] = types.MarketType(s)
		}
	} else if v, ok := g.GetOption("fetchMarkets").(string); ok {
		// 向后兼容：支持单个字符串
		fetchMarketsTypes = []types.MarketType{types.MarketType(v)}
	}

	// 加载现货市场
	if containsMarketType(fetchMarketsTypes, types.MarketTypeSpot) {
		spotMarkets, err := g.loadSpotMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load spot markets: %w", err)
		}
		markets = append(markets, spotMarkets...)
	}

	// 加载永续合约市场（swap）
	if containsMarketType(fetchMarketsTypes, types.MarketTypeSwap) || containsMarketType(fetchMarketsTypes, types.MarketTypeFuture) {
		swapMarkets, err := g.loadSwapMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load swap markets: %w", err)
		}
		markets = append(markets, swapMarkets...)
	}

	g.SetMarkets(markets)
	return nil
}

// loadSpotMarkets 加载现货市场
func (g *Gate) loadSpotMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := g.client.Get(ctx, "/api/v4/spot/currency_pairs", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch spot markets: %w", err)
	}

	var data []struct {
		ID              string `json:"id"`
		Base            string `json:"base"`
		Quote           string `json:"quote"`
		Fee             string `json:"fee"`
		MinBaseAmount   string `json:"min_base_amount"`
		MinQuoteAmount  string `json:"min_quote_amount"`
		MaxQuoteAmount  string `json:"max_quote_amount"`
		AmountPrecision int    `json:"amount_precision"`
		Precision       int    `json:"precision"`
		TradeStatus     string `json:"trade_status"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal spot markets: %w", err)
	}

	markets := make([]*types.Market, 0)
	for _, s := range data {
		if s.TradeStatus != "tradable" {
			continue
		}

		// Gate 使用下划线分隔，如 BTC_USDT
		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(s.Base, s.Quote)

		market := &types.Market{
			ID:     s.ID,             // Gate 原始格式 (BTC_USDT)
			Symbol: normalizedSymbol, // 标准化格式 (BTC/USDT)
			Base:   s.Base,
			Quote:  s.Quote,
			Type:   types.MarketTypeSpot,
			Active: s.TradeStatus == "tradable",
		}

		// 解析精度
		market.Precision.Amount = s.AmountPrecision
		market.Precision.Price = s.Precision

		// 解析限制
		market.Limits.Amount.Min, _ = strconv.ParseFloat(s.MinBaseAmount, 64)
		market.Limits.Cost.Min, _ = strconv.ParseFloat(s.MinQuoteAmount, 64)
		market.Limits.Cost.Max, _ = strconv.ParseFloat(s.MaxQuoteAmount, 64)

		markets = append(markets, market)
	}

	return markets, nil
}

// loadSwapMarkets 加载永续合约市场
func (g *Gate) loadSwapMarkets(ctx context.Context) ([]*types.Market, error) {
	// Gate 永续合约使用 USDT 作为结算货币
	settle := "usdt"
	resp, err := g.client.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/contracts", settle), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch swap markets: %w", err)
	}

	var data []struct {
		Name             string `json:"name"`
		Type             string `json:"type"`
		QuantoMultiplier string `json:"quanto_multiplier"`
		OrderPriceRound  string `json:"order_price_round"`
		OrderSizeMin     int    `json:"order_size_min"`
		OrderSizeMax     int    `json:"order_size_max"`
		InDelisting      bool   `json:"in_delisting"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal swap markets: %w", err)
	}

	markets := make([]*types.Market, 0)
	for _, s := range data {
		if s.InDelisting {
			continue
		}

		// Gate 合约名称格式为 BTC_USDT
		parts := strings.Split(s.Name, "_")
		if len(parts) != 2 {
			continue
		}
		base := parts[0]
		quote := parts[1]

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(base, quote, strings.ToUpper(settle))

		market := &types.Market{
			ID:       s.Name,
			Symbol:   normalizedSymbol,
			Base:     base,
			Quote:    quote,
			Settle:   strings.ToUpper(settle),
			Type:     types.MarketTypeSwap,
			Active:   !s.InDelisting,
			Contract: true,
			Linear:   true, // U本位永续合约
			Inverse:  false,
		}

		// 解析精度
		orderPriceRound, _ := strconv.ParseFloat(s.OrderPriceRound, 64)
		market.Precision.Price = getPrecisionDigits(orderPriceRound)
		market.Precision.Amount = 0 // Gate 合约使用整数数量

		// 解析限制
		market.Limits.Amount.Min = float64(s.OrderSizeMin)
		market.Limits.Amount.Max = float64(s.OrderSizeMax)

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

// getString 从 map 中获取字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
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

// GetMarketID 获取Gate格式的 symbol ID
// 优先从已加载的市场中查找，如果未找到则使用后备转换函数
func (g *Gate) GetMarketID(symbol string) (string, error) {
	// 优先从已加载的市场中查找
	market, ok := g.GetMarketsMap()[symbol]
	if ok {
		return market.ID, nil
	}

	// 如果市场未加载，使用后备转换函数
	return common.ToGateSymbol(symbol)
}

// GetMarketByID 通过交易所ID获取市场信息
func (g *Gate) GetMarketByID(id string) (*types.Market, error) {
	for _, market := range g.GetMarketsMap() {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, base.ErrMarketNotFound
}

// FetchTicker 获取行情
func (g *Gate) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol, err := g.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	var resp []byte
	if market.Contract {
		// 合约市场
		settle := strings.ToLower(market.Settle)
		resp, err = g.client.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/tickers", settle), map[string]interface{}{
			"contract": gateSymbol,
		})
	} else {
		// 现货市场
		resp, err = g.client.Get(ctx, "/api/v4/spot/tickers", map[string]interface{}{
			"currency_pair": gateSymbol,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	if market.Contract {
		var data []struct {
			Contract              string `json:"contract"`
			Last                  string `json:"last"`
			ChangePercentage      string `json:"change_percentage"`
			TotalSize             string `json:"total_size"`
			Volume24h             string `json:"volume_24h"`
			Volume24hBase         string `json:"volume_24h_base"`
			Volume24hQuote        string `json:"volume_24h_quote"`
			Volume24hSettle       string `json:"volume_24h_settle"`
			MarkPrice             string `json:"mark_price"`
			FundingRate           string `json:"funding_rate"`
			FundingRateIndicative string `json:"funding_rate_indicative"`
			IndexPrice            string `json:"index_price"`
			QuantoBaseRate        string `json:"quanto_base_rate"`
		}
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, fmt.Errorf("unmarshal ticker: %w", err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("ticker not found")
		}

		item := data[0]
		ticker := &types.Ticker{
			Symbol:    symbol,
			Timestamp: time.Now(),
		}
		ticker.Last = item.Last
		ticker.Volume = item.Volume24hBase
		ticker.QuoteVolume = item.Volume24hQuote
		return ticker, nil
	} else {
		var data []struct {
			CurrencyPair     string `json:"currency_pair"`
			Last             string `json:"last"`
			LowestAsk        string `json:"lowest_ask"`
			HighestBid       string `json:"highest_bid"`
			ChangePercentage string `json:"change_percentage"`
			BaseVolume       string `json:"base_volume"`
			QuoteVolume      string `json:"quote_volume"`
			High24h          string `json:"high_24h"`
			Low24h           string `json:"low_24h"`
		}
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, fmt.Errorf("unmarshal ticker: %w", err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("ticker not found")
		}

		item := data[0]
		ticker := &types.Ticker{
			Symbol:    symbol,
			Timestamp: time.Now(),
		}
		ticker.Bid = item.HighestBid
		ticker.Ask = item.LowestAsk
		ticker.Last = item.Last
		ticker.High = item.High24h
		ticker.Low = item.Low24h
		ticker.Volume = item.BaseVolume
		ticker.QuoteVolume = item.QuoteVolume
		return ticker, nil
	}
}

// FetchTickers 批量获取行情
func (g *Gate) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	tickers := make(map[string]*types.Ticker)

	// 获取现货 tickers
	resp, err := g.client.Get(ctx, "/api/v4/spot/tickers", nil)
	if err == nil {
		var data []struct {
			CurrencyPair     string `json:"currency_pair"`
			Last             string `json:"last"`
			LowestAsk        string `json:"lowest_ask"`
			HighestBid       string `json:"highest_bid"`
			ChangePercentage string `json:"change_percentage"`
			BaseVolume       string `json:"base_volume"`
			QuoteVolume      string `json:"quote_volume"`
			High24h          string `json:"high_24h"`
			Low24h           string `json:"low_24h"`
		}
		if err := json.Unmarshal(resp, &data); err == nil {
			for _, item := range data {
				market, err := g.GetMarketByID(item.CurrencyPair)
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
				ticker.Bid = item.HighestBid
				ticker.Ask = item.LowestAsk
				ticker.Last = item.Last
				ticker.High = item.High24h
				ticker.Low = item.Low24h
				ticker.Volume = item.BaseVolume
				ticker.QuoteVolume = item.QuoteVolume
				tickers[normalizedSymbol] = ticker
			}
		}
	}

	// 获取合约 tickers
	resp, err = g.client.Get(ctx, "/api/v4/futures/usdt/tickers", nil)
	if err == nil {
		var data []struct {
			Contract         string `json:"contract"`
			Last             string `json:"last"`
			ChangePercentage string `json:"change_percentage"`
			Volume24hBase    string `json:"volume_24h_base"`
			Volume24hQuote   string `json:"volume_24h_quote"`
		}
		if err := json.Unmarshal(resp, &data); err == nil {
			for _, item := range data {
				market, err := g.GetMarketByID(item.Contract)
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
				ticker.Last = item.Last
				ticker.Volume = item.Volume24hBase
				ticker.QuoteVolume = item.Volume24hQuote
				tickers[normalizedSymbol] = ticker
			}
		}
	}

	return tickers, nil
}

// FetchOHLCV 获取K线数据
func (g *Gate) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.GateTimeframe(timeframe)
	if normalizedTimeframe == "" {
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	// 获取交易所格式的 symbol ID
	gateSymbol, err := g.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	var resp []byte
	if market.Contract {
		// 合约市场
		settle := strings.ToLower(market.Settle)
		params := map[string]interface{}{
			"contract": gateSymbol,
			"interval": normalizedTimeframe,
			"limit":    limit,
		}
		if !since.IsZero() {
			params["from"] = since.Unix()
		}
		resp, err = g.client.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/candlesticks", settle), params)
	} else {
		// 现货市场
		params := map[string]interface{}{
			"currency_pair": gateSymbol,
			"interval":      normalizedTimeframe,
			"limit":         limit,
		}
		if !since.IsZero() {
			params["from"] = since.Unix()
		}
		resp, err = g.client.Get(ctx, "/api/v4/spot/candlesticks", params)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	ohlcvs := make(types.OHLCVs, 0)

	if market.Contract {
		// 合约市场返回对象格式
		var data []map[string]interface{}
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
		}

		parseFloat := func(v interface{}) float64 {
			if str, ok := v.(string); ok {
				f, _ := strconv.ParseFloat(str, 64)
				return f
			}
			if f, ok := v.(float64); ok {
				return f
			}
			f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
			return f
		}

		for _, item := range data {
			ohlcv := types.OHLCV{}
			// 解析时间戳
			if t, ok := item["t"].(float64); ok {
				ohlcv.Timestamp = time.Unix(int64(t), 0)
			} else if t, ok := item["t"].(int64); ok {
				ohlcv.Timestamp = time.Unix(t, 0)
			}
			ohlcv.Open = parseFloat(item["o"])
			ohlcv.High = parseFloat(item["h"])
			ohlcv.Low = parseFloat(item["l"])
			ohlcv.Close = parseFloat(item["c"])
			ohlcv.Volume = parseFloat(item["v"])
			ohlcvs = append(ohlcvs, ohlcv)
		}
	} else {
		// 现货市场返回数组格式
		// Gate OHLCV 格式: [timestamp, quote_volume, close, high, low, open, base_volume]
		var data [][]interface{}
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
		}

		for _, item := range data {
			if len(item) < 7 {
				continue
			}

			ohlcv := types.OHLCV{}
			// Gate 返回的时间戳是字符串格式的 Unix 时间戳（秒）
			var ts int64
			switch v := item[0].(type) {
			case string:
				ts, _ = strconv.ParseInt(v, 10, 64)
			case float64:
				ts = int64(v)
			case int64:
				ts = v
			}
			ohlcv.Timestamp = time.Unix(ts, 0)

			// 解析价格和成交量（Gate 返回的是字符串）
			// item[1] = quote_volume (跳过)
			// item[2] = close
			// item[3] = high
			// item[4] = low
			// item[5] = open
			// item[6] = base_volume
			parseFloat := func(v interface{}) float64 {
				if str, ok := v.(string); ok {
					f, _ := strconv.ParseFloat(str, 64)
					return f
				}
				f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
				return f
			}

			ohlcv.Close = parseFloat(item[2])
			ohlcv.High = parseFloat(item[3])
			ohlcv.Low = parseFloat(item[4])
			ohlcv.Open = parseFloat(item[5])
			ohlcv.Volume = parseFloat(item[6]) // base_volume

			ohlcvs = append(ohlcvs, ohlcv)
		}
	}

	return ohlcvs, nil
}

// signRequest Gate 签名方法
func (g *Gate) signRequest(method, path string, queryString, body string, timestamp int64) string {
	bodyHash := common.HashSHA512(body)

	// 去掉路径中的 /api/v4 前缀（如果存在），因为签名格式中已经包含了
	signPath := path
	if strings.HasPrefix(path, "/api/v4") {
		signPath = strings.TrimPrefix(path, "/api/v4")
	}

	// Gate 签名格式: method\n/api/v4/path\nqueryString\nbodyHash\ntimestamp
	payload := fmt.Sprintf("%s\n/api/v4%s\n%s\n%s\n%d",
		strings.ToUpper(method), signPath, queryString, bodyHash, timestamp)

	return common.SignHMAC512(payload, g.secretKey)
}

// signAndRequest 签名并发送请求
func (g *Gate) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

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

	// 签名（使用同一个 timestamp 确保签名和请求头一致）
	timestamp := common.GetTimestampSeconds()
	signature := g.signRequest(method, path, queryString, bodyStr, timestamp)

	// 设置请求头
	g.client.SetHeader("KEY", g.apiKey)
	g.client.SetHeader("Timestamp", strconv.FormatInt(timestamp, 10))
	g.client.SetHeader("SIGN", signature)
	g.client.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return g.client.Get(ctx, path, params)
	} else {
		return g.client.Post(ctx, path, body)
	}
}

// FetchBalance 获取余额
func (g *Gate) FetchBalance(ctx context.Context) (types.Balances, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	// Gate 现货余额
	resp, err := g.signAndRequest(ctx, "GET", "/api/v4/spot/accounts", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var data []struct {
		Currency  string `json:"currency"`
		Available string `json:"available"`
		Locked    string `json:"locked"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	balances := make(types.Balances)
	for _, bal := range data {
		free, _ := strconv.ParseFloat(bal.Available, 64)
		locked, _ := strconv.ParseFloat(bal.Locked, 64)
		total := free + locked

		balances[bal.Currency] = &types.Balance{
			Currency:  bal.Currency,
			Free:      free,
			Used:      locked,
			Total:     total,
			Available: free,
		}
	}

	return balances, nil
}

// CreateOrder 创建订单
func (g *Gate) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price string, params map[string]interface{}) (*types.Order, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	gateSymbol, err := g.GetMarketID(symbol)
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

	var path string
	reqBody := map[string]interface{}{}

	if market.Contract {
		// 合约订单
		settle := strings.ToLower(market.Settle)
		path = fmt.Sprintf("/api/v4/futures/%s/orders", settle)
		reqBody["contract"] = gateSymbol
		// 非期权合约需要在请求体中包含 settle
		if !strings.Contains(strings.ToLower(market.ID), "option") {
			reqBody["settle"] = settle
		}

		// 合约订单使用 size 参数（整数），正数表示买入，负数表示卖出
		// 从 params 中获取 size，如果没有则根据 side 和 amount 计算
		var size int64
		if sizeVal, ok := params["size"]; ok {
			// 支持字符串格式的 size（可能是浮点数字符串，需要转换为整数）
			switch v := sizeVal.(type) {
			case string:
				// 先解析为浮点数，再转换为整数
				sizeFloat, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid size parameter: %w", err)
				}
				// 使用 math.Ceil 确保至少为 1（对于正数）或 -1（对于负数）
				if sizeFloat >= 0 {
					size = int64(math.Max(1, math.Ceil(sizeFloat)))
				} else {
					size = int64(math.Min(-1, math.Floor(sizeFloat)))
				}
			case int64:
				size = v
			case int:
				size = int64(v)
			case float64:
				// 使用 math.Ceil 确保至少为 1（对于正数）或 -1（对于负数）
				if v >= 0 {
					size = int64(math.Max(1, math.Ceil(v)))
				} else {
					size = int64(math.Min(-1, math.Floor(v)))
				}
			default:
				return nil, fmt.Errorf("invalid size parameter type")
			}
		} else {
			// 根据 side 和 amount 计算 size（使用 math.Ceil 确保至少为 1）
			var amountInt int64
			if amountFloat >= 0 {
				amountInt = int64(math.Max(1, math.Ceil(amountFloat)))
			} else {
				amountInt = int64(math.Min(-1, math.Floor(amountFloat)))
			}
			if side == types.OrderSideBuy {
				size = amountInt // 正数表示买入
			} else {
				size = -amountInt // 负数表示卖出
			}
		}
		// Gate 合约订单的 size 可以是正数（买入）或负数（卖出）
		// 如果 size 是负数且没有设置 reduce_only，可能需要根据实际情况处理
		reqBody["size"] = size

		// 处理 reduce_only 参数
		if reduceOnly, ok := params["reduce_only"].(bool); ok {
			reqBody["reduce_only"] = reduceOnly
		}

		// 设置价格
		if orderType == types.OrderTypeMarket {
			reqBody["price"] = "0" // 市价单使用 0
		} else {
			reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', -1, 64)
		}

		// 设置 time_in_force (tif)
		if tif, ok := params["tif"].(string); ok {
			reqBody["tif"] = tif
		} else if orderType == types.OrderTypeLimit {
			reqBody["tif"] = "gtc" // 限价单默认 gtc
		} else {
			reqBody["tif"] = "ioc" // 市价单默认 ioc
		}

		// 处理 reduce_only
		if reduceOnly, ok := params["reduce_only"].(bool); ok {
			reqBody["reduce_only"] = reduceOnly
		}
	} else {
		// 现货订单
		path = "/api/v4/spot/orders"
		reqBody["currency_pair"] = gateSymbol
		reqBody["side"] = strings.ToLower(string(side))

		if orderType == types.OrderTypeLimit {
			reqBody["type"] = "limit"
			reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', -1, 64)
			reqBody["amount"] = strconv.FormatFloat(amountFloat, 'f', -1, 64)
			reqBody["time_in_force"] = "gtc"
		} else {
			reqBody["type"] = "market"
			// 现货市价单不支持 gtc，使用 ioc 或 fok
			if tif, ok := params["time_in_force"].(string); ok {
				if tif == "ioc" || tif == "fok" {
					reqBody["time_in_force"] = tif
				} else {
					reqBody["time_in_force"] = "ioc" // 默认使用 ioc
				}
			} else {
				reqBody["time_in_force"] = "ioc" // 默认使用 ioc
			}

			// 现货市价买入订单需要使用 cost（报价货币数量）而不是 amount（基础货币数量）
			if side == types.OrderSideBuy {
				// 如果有 cost 参数，使用 cost；否则使用 amount * price 计算 cost
				if cost, ok := params["cost"].(float64); ok {
					reqBody["amount"] = strconv.FormatFloat(cost, 'f', -1, 64)
				} else if priceFloat > 0 {
					cost := amountFloat * priceFloat
					reqBody["amount"] = strconv.FormatFloat(cost, 'f', -1, 64)
				} else {
					// 如果没有价格，尝试获取当前价格
					ticker, err := g.FetchTicker(ctx, symbol)
					if err == nil && ticker.Last != "" {
						if lastPrice, parseErr := strconv.ParseFloat(ticker.Last, 64); parseErr == nil && lastPrice > 0 {
							cost := amountFloat * lastPrice
							reqBody["amount"] = strconv.FormatFloat(cost, 'f', -1, 64)
						}
					} else {
						reqBody["amount"] = strconv.FormatFloat(amountFloat, 'f', -1, 64)
					}
				}
			} else {
				// 现货市价卖出订单使用 amount（基础货币数量）
				reqBody["amount"] = strconv.FormatFloat(amountFloat, 'f', -1, 64)
			}
		}
	}

	// 生成客户端订单ID（如果未提供）
	// Gate 使用 text 参数，也支持 clientOrderId 作为别名
	if _, hasText := params["text"]; !hasText {
		if clientOrderId, hasClientOrderId := params["clientOrderId"]; hasClientOrderId {
			// 如果用户提供了 clientOrderId，使用它
			reqBody["text"] = clientOrderId
		} else {
			// 如果都没有提供，自动生成
			reqBody["text"] = common.GenerateClientOrderID(g.Name())
		}
	}

	// 合并额外参数（排除已处理的参数）
	excludedKeys := map[string]bool{
		"size": true, "tif": true, "reduce_only": true, "time_in_force": true,
		"clientOrderId": true, // 已处理，避免重复
	}
	for k, v := range params {
		if !excludedKeys[k] {
			reqBody[k] = v
		}
	}

	resp, err := g.signAndRequest(ctx, "POST", path, nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 合约订单和现货订单的响应格式不同，使用通用结构解析
	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	order := &types.Order{
		ID:        getString(data, "id"),
		Symbol:    symbol,
		Amount:    amountFloat,
		Price:     priceFloat,
		Timestamp: time.Now(),
	}

	// 解析价格
	if priceStr := getString(data, "price"); priceStr != "" {
		order.Price, _ = strconv.ParseFloat(priceStr, 64)
	}

	// 解析数量：合约订单使用 size，现货订单使用 amount
	if market.Contract {
		// 合约订单：size 是整数，正数表示买入，负数表示卖出
		if sizeVal, ok := data["size"]; ok {
			var size int64
			switch v := sizeVal.(type) {
			case float64:
				size = int64(v)
			case int64:
				size = v
			case int:
				size = int64(v)
			case string:
				size, _ = strconv.ParseInt(v, 10, 64)
			}
			if size < 0 {
				order.Amount = float64(-size)
				order.Side = types.OrderSideSell
			} else {
				order.Amount = float64(size)
				order.Side = types.OrderSideBuy
			}
		}
		// 解析剩余数量
		if leftVal, ok := data["left"]; ok {
			var left int64
			switch v := leftVal.(type) {
			case float64:
				left = int64(v)
			case int64:
				left = v
			case int:
				left = int64(v)
			case string:
				left, _ = strconv.ParseInt(v, 10, 64)
			}
			if left < 0 {
				order.Remaining = float64(-left)
			} else {
				order.Remaining = float64(left)
			}
			order.Filled = order.Amount - order.Remaining
		}
	} else {
		// 现货订单：使用 amount
		if amountStr := getString(data, "amount"); amountStr != "" {
			order.Amount, _ = strconv.ParseFloat(amountStr, 64)
		}
		// 解析剩余数量
		if leftStr := getString(data, "left"); leftStr != "" {
			left, _ := strconv.ParseFloat(leftStr, 64)
			order.Remaining = left
			order.Filled = order.Amount - left
		}
	}

	// 解析 type
	if typeStr := getString(data, "type"); typeStr != "" {
		if strings.ToLower(typeStr) == "market" {
			order.Type = types.OrderTypeMarket
		} else {
			order.Type = types.OrderTypeLimit
		}
	}

	// 转换状态
	if statusStr := getString(data, "status"); statusStr != "" {
		switch statusStr {
		case "open":
			order.Status = types.OrderStatusOpen
		case "closed":
			order.Status = types.OrderStatusFilled
		case "cancelled":
			order.Status = types.OrderStatusCanceled
		default:
			order.Status = types.OrderStatusNew
		}
	}

	return order, nil
}

// CancelOrder 取消订单
func (g *Gate) CancelOrder(ctx context.Context, orderID, symbol string) error {
	if g.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := g.GetMarket(symbol)
	if err != nil {
		return err
	}

	var path string
	if market.Contract {
		settle := strings.ToLower(market.Settle)
		path = fmt.Sprintf("/api/v4/futures/%s/orders/%s", settle, orderID)
	} else {
		path = fmt.Sprintf("/api/v4/spot/orders/%s", orderID)
	}

	_, err = g.signAndRequest(ctx, "DELETE", path, nil, nil)
	return err
}

// FetchOrder 查询订单
func (g *Gate) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	var path string
	if market.Contract {
		settle := strings.ToLower(market.Settle)
		path = fmt.Sprintf("/api/v4/futures/%s/orders/%s", settle, orderID)
	} else {
		path = fmt.Sprintf("/api/v4/spot/orders/%s", orderID)
	}

	resp, err := g.signAndRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var data struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		CurrencyPair string `json:"currency_pair"`
		Contract     string `json:"contract"`
		Type         string `json:"type"`
		Side         string `json:"side"`
		Amount       string `json:"amount"`
		Price        string `json:"price"`
		Left         string `json:"left"`
		FillPrice    string `json:"fill_price"`
		FilledTotal  string `json:"filled_total"`
		CreateTime   string `json:"create_time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	order := &types.Order{
		ID:        data.ID,
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(data.Price, 64)
	order.Amount, _ = strconv.ParseFloat(data.Amount, 64)
	if data.Left != "" {
		left, _ := strconv.ParseFloat(data.Left, 64)
		order.Remaining = left
		order.Filled = order.Amount - left
	}

	if strings.ToLower(data.Side) == "buy" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToLower(data.Type) == "market" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	switch data.Status {
	case "open":
		order.Status = types.OrderStatusOpen
	case "closed":
		order.Status = types.OrderStatusFilled
	case "cancelled":
		order.Status = types.OrderStatusCanceled
	default:
		order.Status = types.OrderStatusNew
	}

	return order, nil
}

// FetchOrders 查询订单列表
func (g *Gate) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Gate 没有直接的 fetchOrders API，需要通过 fetchOpenOrders 和 fetchClosedOrders 组合
	return nil, fmt.Errorf("not implemented: use FetchOpenOrders or implement custom logic")
}

// FetchOpenOrders 查询未成交订单
func (g *Gate) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	gateSymbol, err := g.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	var path string
	params := map[string]interface{}{}
	if market.Contract {
		settle := strings.ToLower(market.Settle)
		path = fmt.Sprintf("/api/v4/futures/%s/orders", settle)
		params["contract"] = gateSymbol
	} else {
		path = "/api/v4/spot/open_orders"
		params["currency_pair"] = gateSymbol
	}

	resp, err := g.signAndRequest(ctx, "GET", path, params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
	}

	var data []struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		CurrencyPair string `json:"currency_pair"`
		Contract     string `json:"contract"`
		Type         string `json:"type"`
		Side         string `json:"side"`
		Amount       string `json:"amount"`
		Price        string `json:"price"`
		Left         string `json:"left"`
		CreateTime   string `json:"create_time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	orders := make([]*types.Order, 0, len(data))
	for _, item := range data {
		order := &types.Order{
			ID:        item.ID,
			Symbol:    symbol,
			Timestamp: time.Now(),
		}

		order.Price, _ = strconv.ParseFloat(item.Price, 64)
		order.Amount, _ = strconv.ParseFloat(item.Amount, 64)
		if item.Left != "" {
			left, _ := strconv.ParseFloat(item.Left, 64)
			order.Remaining = left
			order.Filled = order.Amount - left
		}

		if strings.ToLower(item.Side) == "buy" {
			order.Side = types.OrderSideBuy
		} else {
			order.Side = types.OrderSideSell
		}

		if strings.ToLower(item.Type) == "market" {
			order.Type = types.OrderTypeMarket
		} else {
			order.Type = types.OrderTypeLimit
		}

		switch item.Status {
		case "open":
			order.Status = types.OrderStatusOpen
		default:
			order.Status = types.OrderStatusNew
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// FetchTrades 获取交易记录
func (g *Gate) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	gateSymbol, err := g.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	var path string
	params := map[string]interface{}{}
	if market.Contract {
		settle := strings.ToLower(market.Settle)
		path = fmt.Sprintf("/api/v4/futures/%s/trades", settle)
		params["contract"] = gateSymbol
	} else {
		path = "/api/v4/spot/trades"
		params["currency_pair"] = gateSymbol
	}

	if limit > 0 {
		params["limit"] = limit
	}

	resp, err := g.client.Get(ctx, path, params)
	if err != nil {
		return nil, fmt.Errorf("fetch trades: %w", err)
	}

	var data []struct {
		ID        string `json:"id"`
		Price     string `json:"price"`
		Amount    string `json:"amount"`
		Side      string `json:"side"`
		Timestamp string `json:"create_time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	trades := make([]*types.Trade, 0, len(data))
	for _, item := range data {
		trade := &types.Trade{
			ID:        item.ID,
			Symbol:    symbol,
			Timestamp: time.Now(),
		}

		trade.Price, _ = strconv.ParseFloat(item.Price, 64)
		trade.Amount, _ = strconv.ParseFloat(item.Amount, 64)

		if strings.ToLower(item.Side) == "buy" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchMyTrades 获取我的交易记录
func (g *Gate) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	market, err := g.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	gateSymbol, err := g.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	var path string
	params := map[string]interface{}{}
	if market.Contract {
		settle := strings.ToLower(market.Settle)
		path = fmt.Sprintf("/api/v4/futures/%s/my_trades", settle)
		params["contract"] = gateSymbol
	} else {
		path = "/api/v4/spot/my_trades"
		params["currency_pair"] = gateSymbol
	}

	if limit > 0 {
		params["limit"] = limit
	}

	resp, err := g.signAndRequest(ctx, "GET", path, params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var data []struct {
		ID        string `json:"id"`
		Price     string `json:"price"`
		Amount    string `json:"amount"`
		Side      string `json:"side"`
		Timestamp string `json:"create_time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	trades := make([]*types.Trade, 0, len(data))
	for _, item := range data {
		trade := &types.Trade{
			ID:        item.ID,
			Symbol:    symbol,
			Timestamp: time.Now(),
		}

		trade.Price, _ = strconv.ParseFloat(item.Price, 64)
		trade.Amount, _ = strconv.ParseFloat(item.Amount, 64)

		if strings.ToLower(item.Side) == "buy" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchPositions 获取持仓
func (g *Gate) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	if g.secretKey == "" {
		return nil, base.ErrAuthenticationRequired
	}

	// Gate 合约持仓
	resp, err := g.signAndRequest(ctx, "GET", "/api/v4/futures/usdt/positions", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var data []struct {
		Contract      string `json:"contract"`
		Size          int64  `json:"size"`
		Value         string `json:"value"`
		EntryPrice    string `json:"entry_price"`
		MarkPrice     string `json:"mark_price"`
		UnrealisedPnl string `json:"unrealised_pnl"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	positions := make([]*types.Position, 0)
	for _, item := range data {
		if len(symbols) > 0 {
			market, err := g.GetMarketByID(item.Contract)
			if err != nil {
				continue
			}
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

		market, err := g.GetMarketByID(item.Contract)
		if err != nil {
			continue
		}

		position := &types.Position{
			Symbol:    market.Symbol,
			Amount:    float64(item.Size),
			Timestamp: time.Now(),
		}

		position.EntryPrice, _ = strconv.ParseFloat(item.EntryPrice, 64)
		position.MarkPrice, _ = strconv.ParseFloat(item.MarkPrice, 64)
		position.UnrealizedPnl, _ = strconv.ParseFloat(item.UnrealisedPnl, 64)

		if item.Size > 0 {
			position.Side = types.PositionSideLong
		} else if item.Size < 0 {
			position.Side = types.PositionSideShort
			position.Amount = -position.Amount
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// SetLeverage 设置杠杆
func (g *Gate) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if g.secretKey == "" {
		return base.ErrAuthenticationRequired
	}

	market, err := g.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("leverage only supported for contracts")
	}

	settle := strings.ToLower(market.Settle)
	gateSymbol, err := g.GetMarketID(symbol)
	if err != nil {
		return fmt.Errorf("get market ID: %w", err)
	}

	reqBody := map[string]interface{}{
		"contract": gateSymbol,
		"leverage": strconv.Itoa(leverage),
	}

	path := fmt.Sprintf("/api/v4/futures/%s/positions/%s/leverage", settle, gateSymbol)
	_, err = g.signAndRequest(ctx, "POST", path, nil, reqBody)
	return err
}

// SetMarginMode 设置保证金模式
func (g *Gate) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	// Gate 不支持通过 API 设置保证金模式，需要在网页端设置
	return fmt.Errorf("not supported: Gate does not support setting margin mode via API")
}

func (g *Gate) GetMarkets(ctx context.Context, marketType types.MarketType) ([]*types.Market, error) {
	markets := make([]*types.Market, 0)
	for _, market := range g.GetMarketsMap() {
		if marketType == "" || market.Type == marketType {
			markets = append(markets, market)
		}
	}
	return markets, nil
}

package gate

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
	gateName       = "gate"
	gateBaseURL    = "https://api.gateio.ws"
	gateSandboxURL = "https://api-testnet.gateapi.io"
)

// Gate Gate交易所实现
type Gate struct {
	*exlink.BaseExchange
	client    *common.HTTPClient
	apiKey    string
	secretKey string
}

// NewGate 创建Gate交易所实例
func NewGate(apiKey, secretKey string, options map[string]interface{}) (exlink.Exchange, error) {
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
		BaseExchange: exlink.NewBaseExchange(gateName),
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
		exchange.client.SetHeader("X-Gate-Channel-Id", "api")
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
func (g *Gate) LoadMarkets(ctx context.Context, reload bool) error {
	markets := make([]*types.Market, 0)

	// 获取要加载的市场类型
	fetchMarketsTypes := []string{"spot"}
	if v, ok := g.GetOption("fetchMarkets").([]string); ok && len(v) > 0 {
		fetchMarketsTypes = v
	} else if v, ok := g.GetOption("fetchMarkets").(string); ok {
		fetchMarketsTypes = []string{v}
	}

	// 加载现货市场
	if contains(fetchMarketsTypes, "spot") {
		spotMarkets, err := g.loadSpotMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load spot markets: %w", err)
		}
		markets = append(markets, spotMarkets...)
	}

	// 加载永续合约市场（swap）
	if contains(fetchMarketsTypes, "swap") {
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
			ID:     s.ID,            // Gate 原始格式 (BTC_USDT)
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
		Name            string `json:"name"`
		Type            string `json:"type"`
		QuantoMultiplier string `json:"quanto_multiplier"`
		OrderPriceRound string `json:"order_price_round"`
		OrderSizeMin    int    `json:"order_size_min"`
		OrderSizeMax    int    `json:"order_size_max"`
		InDelisting     bool   `json:"in_delisting"`
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

// contains 检查字符串切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
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
	return nil, exlink.ErrMarketNotFound
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
			Contract        string `json:"contract"`
			Last            string `json:"last"`
			ChangePercentage string `json:"change_percentage"`
			TotalSize       string `json:"total_size"`
			Volume24h       string `json:"volume_24h"`
			Volume24hBase   string `json:"volume_24h_base"`
			Volume24hQuote  string `json:"volume_24h_quote"`
			Volume24hSettle string `json:"volume_24h_settle"`
			MarkPrice       string `json:"mark_price"`
			FundingRate     string `json:"funding_rate"`
			FundingRateIndicative string `json:"funding_rate_indicative"`
			IndexPrice      string `json:"index_price"`
			QuantoBaseRate  string `json:"quanto_base_rate"`
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
		ticker.Last, _ = strconv.ParseFloat(item.Last, 64)
		ticker.Volume, _ = strconv.ParseFloat(item.Volume24hBase, 64)
		ticker.QuoteVolume, _ = strconv.ParseFloat(item.Volume24hQuote, 64)
		changePercent, _ := strconv.ParseFloat(item.ChangePercentage, 64)
		ticker.ChangePercent = changePercent
		return ticker, nil
	} else {
		var data []struct {
			CurrencyPair    string `json:"currency_pair"`
			Last            string `json:"last"`
			LowestAsk       string `json:"lowest_ask"`
			HighestBid      string `json:"highest_bid"`
			ChangePercentage string `json:"change_percentage"`
			BaseVolume      string `json:"base_volume"`
			QuoteVolume     string `json:"quote_volume"`
			High24h         string `json:"high_24h"`
			Low24h          string `json:"low_24h"`
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
		ticker.Bid, _ = strconv.ParseFloat(item.HighestBid, 64)
		ticker.Ask, _ = strconv.ParseFloat(item.LowestAsk, 64)
		ticker.Last, _ = strconv.ParseFloat(item.Last, 64)
		ticker.High, _ = strconv.ParseFloat(item.High24h, 64)
		ticker.Low, _ = strconv.ParseFloat(item.Low24h, 64)
		ticker.Volume, _ = strconv.ParseFloat(item.BaseVolume, 64)
		ticker.QuoteVolume, _ = strconv.ParseFloat(item.QuoteVolume, 64)
		changePercent, _ := strconv.ParseFloat(item.ChangePercentage, 64)
		ticker.ChangePercent = changePercent
		if ticker.Last > 0 && ticker.ChangePercent != 0 {
			ticker.Change = ticker.Last * ticker.ChangePercent / 100
		}
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
			CurrencyPair    string `json:"currency_pair"`
			Last            string `json:"last"`
			LowestAsk       string `json:"lowest_ask"`
			HighestBid      string `json:"highest_bid"`
			ChangePercentage string `json:"change_percentage"`
			BaseVolume      string `json:"base_volume"`
			QuoteVolume     string `json:"quote_volume"`
			High24h         string `json:"high_24h"`
			Low24h          string `json:"low_24h"`
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
				ticker.Bid, _ = strconv.ParseFloat(item.HighestBid, 64)
				ticker.Ask, _ = strconv.ParseFloat(item.LowestAsk, 64)
				ticker.Last, _ = strconv.ParseFloat(item.Last, 64)
				ticker.High, _ = strconv.ParseFloat(item.High24h, 64)
				ticker.Low, _ = strconv.ParseFloat(item.Low24h, 64)
				ticker.Volume, _ = strconv.ParseFloat(item.BaseVolume, 64)
				ticker.QuoteVolume, _ = strconv.ParseFloat(item.QuoteVolume, 64)
				changePercent, _ := strconv.ParseFloat(item.ChangePercentage, 64)
				ticker.ChangePercent = changePercent
				if ticker.Last > 0 && ticker.ChangePercent != 0 {
					ticker.Change = ticker.Last * ticker.ChangePercent / 100
				}
				tickers[normalizedSymbol] = ticker
			}
		}
	}

	// 获取合约 tickers
	resp, err = g.client.Get(ctx, "/api/v4/futures/usdt/tickers", nil)
	if err == nil {
		var data []struct {
			Contract        string `json:"contract"`
			Last            string `json:"last"`
			ChangePercentage string `json:"change_percentage"`
			Volume24hBase   string `json:"volume_24h_base"`
			Volume24hQuote  string `json:"volume_24h_quote"`
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
				ticker.Last, _ = strconv.ParseFloat(item.Last, 64)
				ticker.Volume, _ = strconv.ParseFloat(item.Volume24hBase, 64)
				ticker.QuoteVolume, _ = strconv.ParseFloat(item.Volume24hQuote, 64)
				changePercent, _ := strconv.ParseFloat(item.ChangePercentage, 64)
				ticker.ChangePercent = changePercent
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

// 以下方法需要实现但暂时返回错误，后续可以完善
func (g *Gate) FetchBalance(ctx context.Context) (types.Balances, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price float64, params map[string]interface{}) (*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return fmt.Errorf("not implemented")
}

func (g *Gate) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Gate) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return fmt.Errorf("not implemented")
}

func (g *Gate) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return fmt.Errorf("not implemented")
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


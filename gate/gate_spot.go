package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/types"
)

// GateSpot Gate 现货实现
type GateSpot struct {
	gate   *Gate
	market *gateSpotMarket
	order  *gateSpotOrder
}

// NewGateSpot 创建 Gate 现货实例
func NewGateSpot(g *Gate) *GateSpot {
	return &GateSpot{
		gate:   g,
		market: &gateSpotMarket{gate: g},
		order:  &gateSpotOrder{gate: g},
	}
}

// ========== SpotExchange 接口实现 ==========

func (s *GateSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

func (s *GateSpot) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *GateSpot) GetMarket(symbol string) (*model.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *GateSpot) GetMarkets() ([]*model.Market, error) {
	return s.market.GetMarkets()
}

func (s *GateSpot) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *GateSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

func (s *GateSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (s *GateSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *GateSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *GateSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *GateSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

func (s *GateSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

func (s *GateSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

func (s *GateSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

func (s *GateSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

var _ exchange.SpotExchange = (*GateSpot)(nil)

// ========== 内部实现 ==========

type gateSpotMarket struct {
	gate *Gate
}

func (m *gateSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.gate.mu.RLock()
	if !reload && len(m.gate.spotMarkets) > 0 {
		m.gate.mu.RUnlock()
		return nil
	}
	m.gate.mu.RUnlock()

	// 获取现货市场信息
	resp, err := m.gate.client.HTTPClient.Get(ctx, "/api/v4/spot/currency_pairs", nil)
	if err != nil {
		return fmt.Errorf("fetch spot markets: %w", err)
	}

	var data gateSpotMarketsResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return fmt.Errorf("unmarshal spot markets: %w", err)
	}

	markets := make([]*model.Market, 0)
	for _, s := range data {
		if s.TradeStatus != "tradable" {
			continue
		}

		// Gate 使用下划线分隔，如 BTC_USDT
		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(s.Base, s.Quote)

		market := &model.Market{
			ID:     s.ID,             // Gate 原始格式 (BTC_USDT)
			Symbol: normalizedSymbol, // 标准化格式 (BTC/USDT)
			Base:   s.Base,
			Quote:  s.Quote,
			Type:   model.MarketTypeSpot,
			Active: s.TradeStatus == "tradable",
		}

		// 解析精度
		market.Precision.Amount = s.AmountPrecision
		market.Precision.Price = s.Precision

		// 解析限制
		if !s.MinBaseAmount.IsZero() {
			market.Limits.Amount.Min = s.MinBaseAmount
		}
		if !s.MinQuoteAmount.IsZero() {
			market.Limits.Cost.Min = s.MinQuoteAmount
		}
		if !s.MaxQuoteAmount.IsZero() {
			market.Limits.Cost.Max = s.MaxQuoteAmount
		}

		markets = append(markets, market)
	}

	// 存储市场信息
	m.gate.mu.Lock()
	if m.gate.spotMarkets == nil {
		m.gate.spotMarkets = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.gate.spotMarkets[market.Symbol] = market
	}
	m.gate.mu.Unlock()

	return nil
}

func (m *gateSpotMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.gate.spotMarkets))
	for _, market := range m.gate.spotMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *gateSpotMarket) GetMarket(symbol string) (*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	market, ok := m.gate.spotMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

func (m *gateSpotMarket) GetMarkets() ([]*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.gate.spotMarkets))
	for _, market := range m.gate.spotMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *gateSpotMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	resp, err := m.gate.client.HTTPClient.Get(ctx, "/api/v4/spot/tickers", map[string]interface{}{
		"currency_pair": gateSymbol,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var data gateSpotTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("ticker not found")
	}

	item := data[0]
	ticker := &model.Ticker{
		Symbol:    symbol,
		Timestamp: types.ExTimestamp{Time: time.Now()}, // Gate 现货 API 没有返回时间戳
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

func (m *gateSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	resp, err := m.gate.client.HTTPClient.Get(ctx, "/api/v4/spot/tickers", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data gateSpotTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	// 如果需要过滤特定 symbols，先转换为 Gate 格式
	var gateSymbols map[string]string
	if len(symbols) > 0 {
		gateSymbols = make(map[string]string)
		for _, s := range symbols {
			market, err := m.GetMarket(s)
			if err == nil {
				gateSymbols[market.ID] = s
			} else {
				// 如果市场未加载，尝试转换
				gateSymbol, err := ToGateSymbol(s, false)
				if err == nil {
					gateSymbols[gateSymbol] = s
				}
			}
		}
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range data {
		// 如果指定了 symbols，进行过滤
		if len(symbols) > 0 {
			normalizedSymbol, ok := gateSymbols[item.CurrencyPair]
			if !ok {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: types.ExTimestamp{Time: time.Now()}, // Gate 现货 API 没有返回时间戳
			}
			ticker.Bid = item.HighestBid
			ticker.Ask = item.LowestAsk
			ticker.Last = item.Last
			ticker.High = item.High24h
			ticker.Low = item.Low24h
			ticker.Volume = item.BaseVolume
			ticker.QuoteVolume = item.QuoteVolume
			tickers[normalizedSymbol] = ticker
		} else {
			// 如果没有指定 symbols，尝试从市场信息中查找
			market, err := m.getMarketByID(item.CurrencyPair)
			if err != nil {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    market.Symbol,
				Timestamp: types.ExTimestamp{Time: time.Now()}, // Gate 现货 API 没有返回时间戳
			}
			ticker.Bid = item.HighestBid
			ticker.Ask = item.LowestAsk
			ticker.Last = item.Last
			ticker.High = item.High24h
			ticker.Low = item.Low24h
			ticker.Volume = item.BaseVolume
			ticker.QuoteVolume = item.QuoteVolume
			tickers[market.Symbol] = ticker
		}
	}

	return tickers, nil
}

// getMarketByID 通过交易所ID获取市场信息
func (m *gateSpotMarket) getMarketByID(id string) (*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	for _, market := range m.gate.spotMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (m *gateSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.GateTimeframe(timeframe)

	params := map[string]interface{}{
		"currency_pair": market.ID,
		"interval":      normalizedTimeframe,
		"limit":         limit,
	}
	if !since.IsZero() {
		params["from"] = since.Unix()
	}

	resp, err := m.gate.client.HTTPClient.Get(ctx, "/api/v4/spot/candlesticks", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
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

		// Gate 返回格式: [timestamp, volume, close, high, low, open]
		if ts, ok := item[0].(float64); ok {
			ohlcv.Timestamp = time.Unix(int64(ts), 0)
		} else if tsStr, ok := item[0].(string); ok {
			if ts, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
				ohlcv.Timestamp = time.Unix(ts, 0)
			}
		}

		if open, ok := item[5].(string); ok {
			ohlcv.Open, _ = strconv.ParseFloat(open, 64)
		} else if open, ok := item[5].(float64); ok {
			ohlcv.Open = open
		}

		if high, ok := item[3].(string); ok {
			ohlcv.High, _ = strconv.ParseFloat(high, 64)
		} else if high, ok := item[3].(float64); ok {
			ohlcv.High = high
		}

		if low, ok := item[4].(string); ok {
			ohlcv.Low, _ = strconv.ParseFloat(low, 64)
		} else if low, ok := item[4].(float64); ok {
			ohlcv.Low = low
		}

		if close, ok := item[2].(string); ok {
			ohlcv.Close, _ = strconv.ParseFloat(close, 64)
		} else if close, ok := item[2].(float64); ok {
			ohlcv.Close = close
		}

		if volume, ok := item[1].(string); ok {
			ohlcv.Volume, _ = strconv.ParseFloat(volume, 64)
		} else if volume, ok := item[1].(float64); ok {
			ohlcv.Volume = volume
		}

		ohlcvs = append(ohlcvs, ohlcv)
	}

	return ohlcvs, nil
}

type gateSpotOrder struct {
	gate *Gate
}

// signAndRequest 签名并发送请求（Gate API）
func (o *gateSpotOrder) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if o.gate.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 构建查询字符串
	queryString := ""
	if len(params) > 0 {
		queryString = common.BuildQueryString(params)
	}

	// 构建请求体
	bodyStr := ""
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyStr = string(bodyBytes)
	}

	// 签名（使用同一个 timestamp 确保签名和请求头一致）
	timestamp := common.GetTimestampSeconds()
	signature := o.gate.signer.SignRequest(method, path, queryString, bodyStr, timestamp)

	// 设置请求头
	o.gate.client.HTTPClient.SetHeader("KEY", o.gate.client.APIKey)
	o.gate.client.HTTPClient.SetHeader("Timestamp", strconv.FormatInt(timestamp, 10))
	o.gate.client.HTTPClient.SetHeader("SIGN", signature)
	o.gate.client.HTTPClient.SetHeader("Content-Type", "application/json")
	o.gate.client.HTTPClient.SetHeader("X-Gate-Channel-Id", "api")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return o.gate.client.HTTPClient.Get(ctx, path, params)
	} else {
		return o.gate.client.HTTPClient.Post(ctx, path, body)
	}
}

func (o *gateSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	// Gate 现货余额
	resp, err := o.signAndRequest(ctx, "GET", "/api/v4/spot/accounts", nil, nil)
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

func (o *gateSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// 解析选项
	options := types.ApplyOrderOptions(opts...)

	// 判断订单类型
	var orderType types.OrderType
	var priceStr string
	if options.Price != nil && *options.Price != "" {
		orderType = types.OrderTypeLimit
		priceStr = *options.Price
	} else {
		orderType = types.OrderTypeMarket
		priceStr = ""
	}

	market, err := o.gate.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	var priceFloat float64
	if priceStr != "" {
		priceFloat, err = strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
	}

	path := "/api/v4/spot/orders"
	reqBody := map[string]interface{}{
		"currency_pair": gateSymbol,
		"side":          strings.ToLower(string(side)),
	}

	if orderType == types.OrderTypeLimit {
		reqBody["type"] = "limit"
		reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', -1, 64)
		reqBody["amount"] = strconv.FormatFloat(amountFloat, 'f', -1, 64)

		// TimeInForce 设置
		if options.TimeInForce != nil {
			reqBody["time_in_force"] = strings.ToLower(string(*options.TimeInForce))
		} else {
			reqBody["time_in_force"] = "gtc"
		}
	} else {
		reqBody["type"] = "market"
		reqBody["time_in_force"] = "ioc" // 市价单固定 ioc

		// 现货市价买单: 需要通过Ticker换算成USDT数量
		if side == types.OrderSideBuy {
			ticker, err := o.gate.spot.market.FetchTicker(ctx, symbol)
			if err != nil {
				return nil, fmt.Errorf("fetch ticker for market buy: %w", err)
			}

			lastPrice, _ := strconv.ParseFloat(ticker.Last.String(), 64)
			if lastPrice == 0 {
				return nil, fmt.Errorf("invalid ticker price")
			}

			cost := amountFloat * lastPrice
			reqBody["amount"] = strconv.FormatFloat(cost, 'f', -1, 64)
		} else {
			// 现货市价卖单: 直接使用 amount
			reqBody["amount"] = strconv.FormatFloat(amountFloat, 'f', -1, 64)
		}
	}

	// 客户端订单ID
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqBody["text"] = *options.ClientOrderID
	} else {
		reqBody["text"] = common.GenerateClientOrderID(o.gate.Name(), side)
	}

	resp, err := o.signAndRequest(ctx, "POST", path, nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	order := &types.Order{
		ID:        getString(data, "id"),
		Symbol:    symbol,
		Type:      orderType,
		Side:      side,
		Amount:    amountFloat,
		Price:     priceFloat,
		Timestamp: time.Now(),
		Status:    types.OrderStatusNew,
	}

	// 解析状态
	if statusStr := getString(data, "status"); statusStr != "" {
		switch statusStr {
		case "open":
			order.Status = types.OrderStatusOpen
		case "closed":
			order.Status = types.OrderStatusFilled
		case "cancelled":
			order.Status = types.OrderStatusCanceled
		}
	}

	return order, nil
}

func (o *gateSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// 获取市场信息
	market, err := o.gate.spot.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, false)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"currency_pair": gateSymbol,
		"order_id":      orderID,
	}

	_, err = o.signAndRequest(ctx, "DELETE", "/api/v4/spot/orders/"+orderID, nil, reqBody)
	return err
}

// parseOrder 解析订单数据
func (o *gateSpotOrder) parseOrder(data map[string]interface{}, symbol string) *types.Order {
	order := &types.Order{
		ID:        getString(data, "id"),
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(getString(data, "price"), 64)
	order.Amount, _ = strconv.ParseFloat(getString(data, "amount"), 64)
	order.Filled, _ = strconv.ParseFloat(getString(data, "filled_total"), 64)
	order.Remaining = order.Amount - order.Filled

	if strings.ToLower(getString(data, "side")) == "buy" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToLower(getString(data, "type")) == "market" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	statusStr := getString(data, "status")
	switch statusStr {
	case "open":
		order.Status = types.OrderStatusOpen
	case "finished":
		order.Status = types.OrderStatusFilled
	case "cancelled":
		order.Status = types.OrderStatusCanceled
	default:
		order.Status = types.OrderStatusNew
	}

	return order
}

func (o *gateSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// 获取市场信息
	market, err := o.gate.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"currency_pair": gateSymbol,
		"order_id":      orderID,
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v4/spot/orders/"+orderID, params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	return o.parseOrder(data, symbol), nil
}

func (o *gateSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Gate 现货 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: Gate spot API does not support fetching order history directly")
}

func (o *gateSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	params := map[string]interface{}{}
	if symbol != "" {
		// 获取市场信息
		market, err := o.gate.spot.market.GetMarket(symbol)
		if err != nil {
			return nil, err
		}

		// 获取交易所格式的 symbol ID
		gateSymbol := market.ID
		if gateSymbol == "" {
			var err error
			gateSymbol, err = ToGateSymbol(symbol, false)
			if err != nil {
				return nil, fmt.Errorf("get market ID: %w", err)
			}
		}
		params["currency_pair"] = gateSymbol
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v4/spot/open_orders", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	orders := make([]*types.Order, 0, len(data))
	for _, item := range data {
		normalizedSymbol := symbol
		if symbol == "" {
			// 如果没有提供symbol，尝试从市场信息中查找
			currencyPair := getString(item, "currency_pair")
			market, err := o.gate.spot.market.getMarketByID(currencyPair)
			if err == nil {
				normalizedSymbol = market.Symbol
			} else {
				normalizedSymbol = currencyPair // 临时使用原格式
			}
		}
		orders = append(orders, o.parseOrder(item, normalizedSymbol))
	}

	return orders, nil
}

func (o *gateSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.gate.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"currency_pair": gateSymbol,
		"limit":         limit,
	}
	if !since.IsZero() {
		params["from"] = since.Unix()
	}

	resp, err := o.gate.client.HTTPClient.Get(ctx, "/api/v4/spot/trades", params)
	if err != nil {
		return nil, fmt.Errorf("fetch trades: %w", err)
	}

	var data []struct {
		ID         string `json:"id"`
		Price      string `json:"price"`
		Amount     string `json:"amount"`
		Side       string `json:"side"`
		CreateTime string `json:"create_time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	trades := make([]*types.Trade, 0, len(data))
	for _, item := range data {
		price, _ := strconv.ParseFloat(item.Price, 64)
		amount, _ := strconv.ParseFloat(item.Amount, 64)
		createTime, _ := strconv.ParseInt(item.CreateTime, 10, 64)

		trade := &types.Trade{
			ID:        item.ID,
			Symbol:    symbol,
			Price:     price,
			Amount:    amount,
			Cost:      price * amount,
			Timestamp: time.Unix(createTime, 0),
		}

		if strings.ToLower(item.Side) == "buy" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func (o *gateSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.gate.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"currency_pair": gateSymbol,
		"limit":         limit,
	}
	if !since.IsZero() {
		params["from"] = since.Unix()
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v4/spot/my_trades", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var data []struct {
		ID          string `json:"id"`
		OrderID     string `json:"order_id"`
		Price       string `json:"price"`
		Amount      string `json:"amount"`
		Side        string `json:"side"`
		Fee         string `json:"fee"`
		FeeCurrency string `json:"fee_currency"`
		CreateTime  string `json:"create_time"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	trades := make([]*types.Trade, 0, len(data))
	for _, item := range data {
		price, _ := strconv.ParseFloat(item.Price, 64)
		amount, _ := strconv.ParseFloat(item.Amount, 64)
		createTime, _ := strconv.ParseInt(item.CreateTime, 10, 64)
		fee, _ := strconv.ParseFloat(item.Fee, 64)

		trade := &types.Trade{
			ID:        item.ID,
			OrderID:   item.OrderID,
			Symbol:    symbol,
			Price:     price,
			Amount:    amount,
			Cost:      price * amount,
			Timestamp: time.Unix(createTime, 0),
		}

		if strings.ToLower(item.Side) == "buy" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		if fee > 0 && item.FeeCurrency != "" {
			trade.Fee = &types.Fee{
				Currency: item.FeeCurrency,
				Cost:     fee,
			}
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

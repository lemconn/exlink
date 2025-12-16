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
	"github.com/lemconn/exlink/option"
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

func (s *GateSpot) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	return s.market.FetchTickers(ctx)
}

func (s *GateSpot) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}
	limit := 100
	if argsOpts.Limit != nil {
		limit = *argsOpts.Limit
	}
	since := time.Time{}
	if argsOpts.Since != nil {
		since = *argsOpts.Since
	}
	return s.market.FetchOHLCVs(ctx, symbol, timeframe, since, limit)
}

func (s *GateSpot) FetchBalance(ctx context.Context) (model.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *GateSpot) CreateOrder(ctx context.Context, symbol string, side model.OrderSide, amount string, opts ...option.ArgsOption) (*model.Order, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}
	orderOpts := []model.OrderOption{}
	if argsOpts.Price != nil {
		orderOpts = append(orderOpts, model.WithPrice(*argsOpts.Price))
	}
	if argsOpts.Amount != nil {
		orderOpts = append(orderOpts, model.WithAmount(*argsOpts.Amount))
	}
	if argsOpts.ClientOrderID != nil {
		orderOpts = append(orderOpts, model.WithClientOrderID(*argsOpts.ClientOrderID))
	}
	if argsOpts.TimeInForce != nil {
		orderOpts = append(orderOpts, model.WithTimeInForce(model.OrderTimeInForce(*argsOpts.TimeInForce)))
	}
	return s.order.CreateOrder(ctx, symbol, side, amount, orderOpts...)
}

func (s *GateSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *GateSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

var _ exchange.SpotExchange = (*GateSpot)(nil)

// ========== 内部实现 ==========

type gateSpotMarket struct {
	gate *Gate
}

func (m *gateSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.gate.mu.RLock()
	if !reload && len(m.gate.spotMarketsBySymbol) > 0 {
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
	if m.gate.spotMarketsBySymbol == nil {
		m.gate.spotMarketsBySymbol = make(map[string]*model.Market)
		m.gate.spotMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.gate.spotMarketsBySymbol[market.Symbol] = market
		m.gate.spotMarketsByID[market.ID] = market
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

	markets := make([]*model.Market, 0, len(m.gate.spotMarketsBySymbol))
	for _, market := range m.gate.spotMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *gateSpotMarket) GetMarket(key string) (*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := m.gate.spotMarketsBySymbol[key]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := m.gate.spotMarketsByID[key]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", key)
}

func (m *gateSpotMarket) GetMarkets() ([]*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.gate.spotMarketsBySymbol))
	for _, market := range m.gate.spotMarketsBySymbol {
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

func (m *gateSpotMarket) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := m.gate.client.HTTPClient.Get(ctx, "/api/v4/spot/tickers", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data gateSpotTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range data {
		// 尝试从市场信息中查找标准化格式
		market, err := m.GetMarket(item.CurrencyPair)
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

	return tickers, nil
}

func (m *gateSpotMarket) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
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

	var data gateSpotKlineResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	ohlcvs := make(model.OHLCVs, 0, len(data))
	for _, item := range data {
		ohlcv := &model.OHLCV{
			Timestamp: item.OpenTime,
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			Volume:    item.BaseVolume,
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
	if method == "GET" {
		return o.gate.client.HTTPClient.Get(ctx, path, params)
	} else if method == "DELETE" {
		return o.gate.client.HTTPClient.Delete(ctx, path, params, body)
	} else {
		return o.gate.client.HTTPClient.Post(ctx, path, body)
	}
}

func (o *gateSpotOrder) FetchBalance(ctx context.Context) (model.Balances, error) {
	// Gate 现货余额
	resp, err := o.signAndRequest(ctx, "GET", "/api/v4/spot/accounts", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var data gateSpotBalanceResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	balances := make(model.Balances, 0)
	for _, bal := range data {
		total := bal.Available.Add(bal.Locked.Decimal)
		balance := &model.Balance{
			Currency:  bal.Currency,
			Available: bal.Available,
			Locked:    bal.Locked,
			Total:     types.ExDecimal{Decimal: total},
			UpdatedAt: types.ExTimestamp{Time: time.Now()},
		}
		balances = append(balances, balance)
	}

	return balances, nil
}

func (o *gateSpotOrder) CreateOrder(ctx context.Context, symbol string, side model.OrderSide, amount string, opts ...model.OrderOption) (*model.Order, error) {
	// 解析选项
	options := model.ApplyOrderOptions(opts...)

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
		if side == model.OrderSideBuy {
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
		reqBody["text"] = common.GenerateClientOrderID(o.gate.Name(), types.OrderSide(side))
	}

	resp, err := o.signAndRequest(ctx, "POST", "/api/v4/spot/orders", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result gateSpotCreateOrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	// 解析状态
	var status model.OrderStatus
	switch result.FinishAs {
	case "":
		status = model.OrderStatusOpen
	case "filled":
		status = model.OrderStatusClosed
	case "cancelled":
		status = model.OrderStatusCanceled
	case "expired":
		status = model.OrderStatusExpired
	default:
		status = model.OrderStatusNew
	}

	// 转换订单类型
	var modelOrderType model.OrderType
	if orderType == types.OrderTypeMarket {
		modelOrderType = model.OrderTypeMarket
	} else {
		modelOrderType = model.OrderTypeLimit
	}

	// 转换订单方向
	var modelOrderSide model.OrderSide
	if side == model.OrderSideBuy {
		modelOrderSide = model.OrderSideBuy
	} else {
		modelOrderSide = model.OrderSideSell
	}

	order := &model.Order{
		ID:            result.ID,
		ClientOrderID: result.Text,
		Symbol:        symbol,
		Type:          modelOrderType,
		Side:          modelOrderSide,
		Amount:        result.Amount,
		Price:         result.Price,
		Timestamp:     result.CreateTimeMs,
		Status:        status,
	}

	// 设置手续费
	if !result.Fee.IsZero() && result.FeeCurrency != "" {
		order.Fee = &model.Fee{
			Currency: result.FeeCurrency,
			Cost:     result.Fee,
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
	}

	_, err = o.signAndRequest(ctx, "DELETE", "/api/v4/spot/orders/"+orderID, reqBody, nil)
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

package binance

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
	"github.com/shopspring/decimal"
)

// BinanceSpot Binance 现货实现
type BinanceSpot struct {
	binance *Binance
	market  *binanceSpotMarket
	order   *binanceSpotOrder
}

// NewBinanceSpot 创建 Binance 现货实例
func NewBinanceSpot(b *Binance) *BinanceSpot {
	return &BinanceSpot{
		binance: b,
		market:  &binanceSpotMarket{binance: b},
		order:   &binanceSpotOrder{binance: b},
	}
}

// ========== SpotExchange 接口实现 ==========

// LoadMarkets 加载市场信息
func (s *BinanceSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

// FetchMarkets 获取市场列表
func (s *BinanceSpot) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return s.market.FetchMarkets(ctx)
}

// GetMarket 获取单个市场信息
func (s *BinanceSpot) GetMarket(symbol string) (*model.Market, error) {
	return s.market.GetMarket(symbol)
}

// GetMarkets 从内存中获取所有市场信息
func (s *BinanceSpot) GetMarkets() ([]*model.Market, error) {
	return s.market.GetMarkets()
}

// FetchTicker 获取行情（单个）
func (s *BinanceSpot) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

// FetchTickers 批量获取行情
func (s *BinanceSpot) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	return s.market.FetchTickers(ctx)
}

// FetchOHLCVs 获取K线数据
func (s *BinanceSpot) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取参数值（带默认值）
	limit := 100 // 默认值
	if argsOpts.Limit != nil {
		limit = *argsOpts.Limit
	}

	since := time.Time{} // 默认值
	if argsOpts.Since != nil {
		since = *argsOpts.Since
	}

	return s.market.FetchOHLCVs(ctx, symbol, timeframe, since, limit)
}

// FetchBalance 获取余额
func (s *BinanceSpot) FetchBalance(ctx context.Context) (model.Balances, error) {
	return s.order.FetchBalance(ctx)
}

// CreateOrder 创建订单
func (s *BinanceSpot) CreateOrder(ctx context.Context, symbol string, side option.SpotOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

// CancelOrder 取消订单
func (s *BinanceSpot) CancelOrder(ctx context.Context, symbol string, orderID string, opts ...option.ArgsOption) error {
	return s.order.CancelOrder(ctx, symbol, orderID, opts...)
}

// FetchOrder 查询订单
func (s *BinanceSpot) FetchOrder(ctx context.Context, symbol string, orderID string, opts ...option.ArgsOption) (*model.SpotOrder, error) {
	return s.order.FetchOrder(ctx, symbol, orderID, opts...)
}

// 确保 BinanceSpot 实现了 exchange.SpotExchange 接口
var _ exchange.SpotExchange = (*BinanceSpot)(nil)

// ========== 内部实现 ==========

// binanceSpotMarket 现货市场相关方法
type binanceSpotMarket struct {
	binance *Binance
}

// LoadMarkets 加载市场信息
func (m *binanceSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.binance.mu.RLock()
	if !reload && len(m.binance.spotMarketsBySymbol) > 0 {
		m.binance.mu.RUnlock()
		return nil
	}
	m.binance.mu.RUnlock()

	// 获取现货市场信息
	resp, err := m.binance.client.SpotClient.Get(ctx, "/api/v3/exchangeInfo", map[string]interface{}{
		"showPermissionSets": false,
	})
	if err != nil {
		return fmt.Errorf("fetch exchange info: %w", err)
	}

	var info binanceSpotMarketsResponse
	if err := json.Unmarshal(resp, &info); err != nil {
		return fmt.Errorf("unmarshal exchange info: %w", err)
	}

	markets := make([]*model.Market, 0)
	for _, s := range info.Symbols {
		if s.Status != "TRADING" {
			continue
		}

		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(s.BaseAsset, s.QuoteAsset)

		market := &model.Market{
			ID:     s.Symbol,         // Binance 原始格式 (BTCUSDT)
			Symbol: normalizedSymbol, // 标准化格式 (BTC/USDT)
			Base:   s.BaseAsset,
			Quote:  s.QuoteAsset,
			Type:   model.MarketTypeSpot,
			Active: s.Status == "TRADING",
		}

		// 解析精度和限制
		market.Precision.Amount = s.BaseAssetPrecision
		market.Precision.Price = s.QuotePrecision

		for _, filter := range s.Filters {
			switch filter.FilterType {
			case "LOT_SIZE":
				if !filter.MinQty.IsZero() {
					market.Limits.Amount.Min = filter.MinQty
				}
				if !filter.MaxQty.IsZero() {
					market.Limits.Amount.Max = filter.MaxQty
				}
				if !filter.StepSize.IsZero() {
					// 从 StepSize 计算数量精度
					stepSizeStr := filter.StepSize.String()
					parts := strings.Split(stepSizeStr, ".")
					if len(parts) > 1 {
						market.Precision.Amount = len(strings.TrimRight(parts[1], "0"))
					}
				}
			case "PRICE_FILTER":
				if !filter.MinPrice.IsZero() {
					market.Limits.Price.Min = filter.MinPrice
				}
				if !filter.MaxPrice.IsZero() {
					market.Limits.Price.Max = filter.MaxPrice
				}
				if !filter.TickSize.IsZero() {
					// 从 TickSize 计算价格精度
					tickSizeStr := filter.TickSize.String()
					parts := strings.Split(tickSizeStr, ".")
					if len(parts) > 1 {
						market.Precision.Price = len(strings.TrimRight(parts[1], "0"))
					}
				}
			case "MIN_NOTIONAL":
				if !filter.MinNotional.IsZero() {
					market.Limits.Cost.Min = filter.MinNotional
				}
			}
		}

		markets = append(markets, market)
	}

	// 存储市场信息
	m.binance.mu.Lock()
	if m.binance.spotMarketsBySymbol == nil {
		m.binance.spotMarketsBySymbol = make(map[string]*model.Market)
		m.binance.spotMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.binance.spotMarketsBySymbol[market.Symbol] = market
		m.binance.spotMarketsByID[market.ID] = market
	}
	m.binance.mu.Unlock()

	return nil
}

// FetchMarkets 获取市场列表
func (m *binanceSpotMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.binance.spotMarketsBySymbol))
	for _, market := range m.binance.spotMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

// GetMarket 获取单个市场信息
func (m *binanceSpotMarket) GetMarket(key string) (*model.Market, error) {
	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := m.binance.spotMarketsBySymbol[key]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := m.binance.spotMarketsByID[key]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", key)
}

// GetMarkets 从内存中获取所有市场信息
func (m *binanceSpotMarket) GetMarkets() ([]*model.Market, error) {
	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.binance.spotMarketsBySymbol))
	for _, market := range m.binance.spotMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

// FetchTicker 获取行情（单个）
func (m *binanceSpotMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		// 如果 market.ID 为空，使用转换函数
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// 使用现货 API
	resp, err := m.binance.client.SpotClient.Get(ctx, "/api/v3/ticker/24hr", map[string]interface{}{
		"symbol": binanceSymbol,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var data binanceSpotTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	// 转换回标准化格式 - 使用输入的symbol（已经是标准化格式）
	ticker := &model.Ticker{
		Symbol:    symbol, // 使用输入的标准化格式
		Timestamp: data.CloseTime,
	}

	ticker.Bid = data.BidPrice
	ticker.Ask = data.AskPrice
	ticker.Last = data.LastPrice
	ticker.Open = data.OpenPrice
	ticker.High = data.HighPrice
	ticker.Low = data.LowPrice
	ticker.Volume = data.Volume
	ticker.QuoteVolume = data.QuoteVolume

	return ticker, nil
}

// FetchTickers 批量获取行情
func (m *binanceSpotMarket) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := m.binance.client.SpotClient.Get(ctx, "/api/v3/ticker/24hr", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data []binanceSpotTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range data {
		// 尝试从市场信息中查找标准化格式
		market, err := m.GetMarket(item.Symbol)
		if err != nil {
			// 如果找不到市场信息，使用 Binance 原始格式
			ticker := &model.Ticker{
				Symbol:    item.Symbol,
				Timestamp: item.CloseTime,
			}
			ticker.Bid = item.BidPrice
			ticker.Ask = item.AskPrice
			ticker.Last = item.LastPrice
			ticker.Open = item.OpenPrice
			ticker.High = item.HighPrice
			ticker.Low = item.LowPrice
			ticker.Volume = item.Volume
			ticker.QuoteVolume = item.QuoteVolume
			tickers[item.Symbol] = ticker
		} else {
			ticker := &model.Ticker{
				Symbol:    market.Symbol,
				Timestamp: item.CloseTime,
			}
			ticker.Bid = item.BidPrice
			ticker.Ask = item.AskPrice
			ticker.Last = item.LastPrice
			ticker.Open = item.OpenPrice
			ticker.High = item.HighPrice
			ticker.Low = item.LowPrice
			ticker.Volume = item.Volume
			ticker.QuoteVolume = item.QuoteVolume
			tickers[market.Symbol] = ticker
		}
	}

	return tickers, nil
}

// FetchOHLCVs 获取K线数据
func (m *binanceSpotMarket) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
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
		// 如果 market.ID 为空，使用转换函数
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}
	params["symbol"] = binanceSymbol

	// 使用现货 API
	resp, err := m.binance.client.SpotClient.Get(ctx, "/api/v3/klines", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var data binanceSpotKlineResponse
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
			Volume:    item.Volume,
		}
		ohlcvs = append(ohlcvs, ohlcv)
	}

	return ohlcvs, nil
}

// binanceSpotOrder 现货订单相关方法
type binanceSpotOrder struct {
	binance *Binance
}

// FetchBalance 获取余额
func (o *binanceSpotOrder) FetchBalance(ctx context.Context) (model.Balances, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	resp, err := o.binance.client.SpotClient.Get(ctx, "/api/v3/account", params)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var data binanceSpotBalanceResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	balances := make(model.Balances, 0)
	for _, bal := range data.Balances {
		total := bal.Free.Add(bal.Locked.Decimal)
		balance := &model.Balance{
			Currency:  bal.Asset,
			Available: bal.Free,
			Locked:    bal.Locked,
			Total:     types.ExDecimal{Decimal: total},
			UpdatedAt: data.UpdateTime,
		}
		balances = append(balances, balance)
	}

	return balances, nil
}

// CreateOrder 创建订单
func (o *binanceSpotOrder) CreateOrder(ctx context.Context, symbol string, side option.SpotOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 解析订单选项
	options := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 获取市场信息
	market, err := o.binance.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// 判断订单类型：如果 options.Price 设置了且不为空，则为限价单，否则为市价单
	var orderType model.OrderType
	var priceStr string
	if options.Price != nil && *options.Price != "" {
		orderType = model.OrderTypeLimit
		priceStr = *options.Price
	} else {
		orderType = model.OrderTypeMarket
		priceStr = ""
	}

	// 解析 amount 字符串为 decimal
	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if amountDecimal.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	// 解析 price 字符串为 decimal（如果存在）
	var priceDecimal decimal.Decimal
	if priceStr != "" {
		priceDecimal, err = decimal.NewFromString(priceStr)
		if err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
		if priceDecimal.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("limit order requires price > 0")
		}
	}

	// 构建基础请求参数
	reqTimestamp := common.GetTimestamp()
	reqParams := map[string]interface{}{
		"symbol":    binanceSymbol,
		"side":      side,
		"type":      orderType.Upper(),
		"timestamp": reqTimestamp,
	}

	// 格式化数量（现货订单使用市场精度）
	amountPrecision := market.Precision.Amount
	if amountPrecision == 0 {
		amountPrecision = 8 // 默认精度
	}
	reqParams["quantity"] = amountDecimal.StringFixed(int32(amountPrecision))

	// 处理限价单的价格和 timeInForce
	if orderType == model.OrderTypeLimit {
		pricePrecision := market.Precision.Price
		if pricePrecision == 0 {
			pricePrecision = 8 // 默认精度
		}
		reqParams["price"] = priceDecimal.StringFixed(int32(pricePrecision))

		// 处理 timeInForce：如果设置了则使用，否则使用默认值 GTC
		if options.TimeInForce != nil {
			reqParams["timeInForce"] = options.TimeInForce.Upper()
		} else {
			reqParams["timeInForce"] = model.OrderTimeInForceGTC.Upper()
		}
	}

	// 生成客户端订单ID（如果未提供）
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqParams["newClientOrderId"] = *options.ClientOrderID
	} else {
		reqParams["newClientOrderId"] = common.GenerateClientOrderID(o.binance.Name(), side.ToSide())
	}

	// 构建签名
	queryString := BuildQueryString(reqParams)
	signature := o.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	// 发送请求（现货订单使用 SpotClient）
	resp, err := o.binance.client.SpotClient.Request(ctx, "POST", "/api/v3/order", reqParams, nil)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 解析响应（现货订单响应）
	var respData binanceSpotCreateOrderResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal spot order response: %w", err)
	}

	// 构建订单对象
	order := &model.NewOrder{
		OrderId:       strconv.FormatInt(respData.OrderID, 10),
		ClientOrderID: respData.ClientOrderID,
		Symbol:        symbol,
		Timestamp:     respData.Time,
	}

	return order, nil
}

// CancelOrder 取消订单
func (o *binanceSpotOrder) CancelOrder(ctx context.Context, symbol string, orderID string, opts ...option.ArgsOption) error {
	if o.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取市场信息
	market, err := o.binance.spot.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, false)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"symbol":    binanceSymbol,
		"orderId":   orderID,
		"timestamp": timestamp,
	}
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["origClientOrderId"] = *argsOpts.ClientOrderID
	}

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	_, err = o.binance.client.SpotClient.Delete(ctx, "/api/v3/order", params, nil)
	return err
}

// FetchOrder 查询订单
func (o *binanceSpotOrder) FetchOrder(ctx context.Context, symbol string, orderID string, opts ...option.ArgsOption) (*model.SpotOrder, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取市场信息
	market, err := o.binance.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"symbol":    binanceSymbol,
		"orderId":   orderID,
		"timestamp": timestamp,
	}
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["origClientOrderId"] = *argsOpts.ClientOrderID
	}

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	// 使用现货 API
	resp, err := o.binance.client.SpotClient.Get(ctx, "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	// 解析响应
	var data binanceSpotFetchOrderResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	// 计算剩余数量
	remaining := data.OrigQty.Sub(data.ExecutedQty.Decimal)

	// 转换状态
	var status model.OrderStatus
	switch data.Status {
	case "NEW":
		status = model.OrderStatusNew
	case "PARTIALLY_FILLED":
		status = model.OrderStatusOpen
	case "FILLED":
		status = model.OrderStatusFilled
	case "CANCELED", "CANCELLED":
		status = model.OrderStatusCanceled
	case "EXPIRED":
		status = model.OrderStatusExpired
	case "REJECTED":
		status = model.OrderStatusRejected
	default:
		status = model.OrderStatusNew
	}

	// 转换订单类型
	var orderType model.OrderType
	if strings.ToUpper(data.Type) == "MARKET" {
		orderType = model.OrderTypeMarket
	} else {
		orderType = model.OrderTypeLimit
	}

	// 转换订单方向
	var side model.OrderSide
	if strings.ToUpper(data.Side) == "BUY" {
		side = model.OrderSideBuy
	} else {
		side = model.OrderSideSell
	}

	// 构建订单对象
	order := &model.SpotOrder{
		ID:            strconv.FormatInt(data.OrderID, 10),
		ClientOrderID: data.ClientOrderID,
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Amount:        data.OrigQty,
		Price:         data.Price,
		Filled:        data.ExecutedQty,
		Remaining:     types.ExDecimal{Decimal: remaining},
		Cost:          data.CummulativeQuoteQty,
		Average:       data.Price, // Binance 查询订单接口没有返回平均价格，使用订单价格
		Status:        status,
		TimeInForce:   data.TimeInForce,
		CreatedAt:     data.Time,
		UpdatedAt:     data.UpdateTime,
	}

	return order, nil
}

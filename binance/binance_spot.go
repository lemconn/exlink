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
func (s *BinanceSpot) CreateOrder(ctx context.Context, symbol string, side model.OrderSide, opts ...option.ArgsOption) (*model.Order, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 转换为 model.OrderOption
	orderOpts := []model.OrderOption{}
	if argsOpts.Price != nil {
		orderOpts = append(orderOpts, model.WithPrice(*argsOpts.Price))
	}
	if argsOpts.Amount != nil {
		orderOpts = append(orderOpts, model.WithAmount(*argsOpts.Amount))
	}
	if argsOpts.Size != nil {
		orderOpts = append(orderOpts, model.WithSize(*argsOpts.Size))
	}
	if argsOpts.ClientOrderID != nil {
		orderOpts = append(orderOpts, model.WithClientOrderID(*argsOpts.ClientOrderID))
	}
	if argsOpts.TimeInForce != nil {
		orderOpts = append(orderOpts, model.WithTimeInForce(model.OrderTimeInForce(*argsOpts.TimeInForce)))
	}

	return s.order.CreateOrder(ctx, symbol, side, orderOpts...)
}

// CancelOrder 取消订单
func (s *BinanceSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

// FetchOrder 查询订单
func (s *BinanceSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
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
func (o *binanceSpotOrder) CreateOrder(ctx context.Context, symbol string, side model.OrderSide, opts ...model.OrderOption) (*model.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 解析订单选项
	options := model.ApplyOrderOptions(opts...)

	if options == nil || *options.Amount == "" {
		return nil, fmt.Errorf("amount is required")
	}

	amount := *options.Amount

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
	var orderType types.OrderType
	var priceStr string
	if options.Price != nil && *options.Price != "" {
		orderType = types.OrderTypeLimit
		priceStr = *options.Price
	} else {
		orderType = types.OrderTypeMarket
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
		"side":      side.Upper(),
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
	if orderType == types.OrderTypeLimit {
		pricePrecision := market.Precision.Price
		if pricePrecision == 0 {
			pricePrecision = 8 // 默认精度
		}
		reqParams["price"] = priceDecimal.StringFixed(int32(pricePrecision))

		// 处理 timeInForce：如果设置了则使用，否则使用默认值 GTC
		if options.TimeInForce != nil {
			reqParams["timeInForce"] = options.TimeInForce.Upper()
		} else {
			reqParams["timeInForce"] = types.OrderTimeInForceGTC.Upper()
		}
	}

	// 生成客户端订单ID（如果未提供）
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqParams["newClientOrderId"] = *options.ClientOrderID
	} else {
		reqParams["newClientOrderId"] = common.GenerateClientOrderID(o.binance.Name(), types.OrderSide(side))
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

	// 计算平均价格和手续费
	var avgPrice decimal.Decimal
	var totalFee decimal.Decimal
	var feeCurrency string
	if len(respData.Fills) > 0 {
		var totalCost decimal.Decimal
		for _, fill := range respData.Fills {
			totalCost = totalCost.Add(fill.Price.Mul(fill.Qty.Decimal))
			totalFee = totalFee.Add(fill.Commission.Decimal)
			if feeCurrency == "" {
				feeCurrency = fill.CommissionAsset
			}
		}
		if !respData.ExecutedQty.IsZero() {
			avgPrice = totalCost.Div(respData.ExecutedQty.Decimal)
		}
	}

	// 计算剩余数量
	remaining := respData.OrigQty.Sub(respData.ExecutedQty.Decimal)

	// 转换状态
	var status model.OrderStatus
	switch respData.Status {
	case "NEW":
		status = model.OrderStatusNew
	case "PARTIALLY_FILLED":
		status = model.OrderStatusOpen // 部分成交视为开放状态
	case "FILLED":
		status = model.OrderStatusClosed
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

	// 构建订单对象
	order := &model.Order{
		ID:            strconv.FormatInt(respData.OrderID, 10),
		ClientOrderID: respData.ClientOrderID,
		Symbol:        symbol,
		Type:          modelOrderType,
		Side:          modelOrderSide,
		Amount:        respData.OrigQty,
		Price:         respData.Price,
		Filled:        respData.ExecutedQty,
		Remaining:     types.ExDecimal{Decimal: remaining},
		Cost:          respData.CummulativeQuoteQty,
		Average:       types.ExDecimal{Decimal: avgPrice},
		Status:        status,
		Timestamp:     respData.TransactTime,
	}

	// 设置手续费
	if !totalFee.IsZero() && feeCurrency != "" {
		order.Fee = &model.Fee{
			Currency: feeCurrency,
			Cost:     types.ExDecimal{Decimal: totalFee},
		}
	}

	return order, nil
}

// CancelOrder 取消订单
func (o *binanceSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	if o.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
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

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	_, err = o.binance.client.SpotClient.Post(ctx, "/api/v3/order", params)
	return err
}

// FetchOrder 查询订单
func (o *binanceSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
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

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	// 使用现货 API
	resp, err := o.binance.client.SpotClient.Get(ctx, "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	// 解析响应
	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	// 检查是否有错误码
	if code, ok := data["code"].(float64); ok && code != 0 {
		return nil, fmt.Errorf("fetch order error: %v", data["msg"])
	}

	// 提取订单ID
	var orderIDInt int64
	if id, ok := data["orderId"].(float64); ok {
		orderIDInt = int64(id)
	} else {
		return nil, fmt.Errorf("missing orderId in response")
	}

	// 提取时间戳（现货使用 time）
	var timestampInt int64
	if t, ok := data["time"].(float64); ok {
		timestampInt = int64(t)
	} else if t, ok := data["transactTime"].(float64); ok {
		timestampInt = int64(t)
	} else {
		return nil, fmt.Errorf("missing timestamp in response")
	}

	// 提取数量
	origQtyStr := getString(data, "origQty", "quantity")
	executedQtyStr := getString(data, "executedQty", "cumQty")
	priceStr := getString(data, "price", "avgPrice")

	// 解析数值
	origQty, _ := strconv.ParseFloat(origQtyStr, 64)
	executedQty, _ := strconv.ParseFloat(executedQtyStr, 64)
	orderPrice, _ := strconv.ParseFloat(priceStr, 64)

	// 构建订单对象
	order := &types.Order{
		ID:            strconv.FormatInt(orderIDInt, 10),
		ClientOrderID: getString(data, "clientOrderId", "newClientStrategyId"),
		Symbol:        symbol, // 使用标准化格式
		Amount:        origQty,
		Price:         orderPrice,
		Filled:        executedQty,
		Remaining:     origQty - executedQty,
		Timestamp:     time.UnixMilli(timestampInt),
	}

	// 转换方向
	sideStr := getString(data, "side")
	if strings.ToUpper(sideStr) == "BUY" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	// 转换类型
	typeStr := getString(data, "type")
	if strings.ToUpper(typeStr) == "MARKET" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	statusStr := getString(data, "status", "strategyStatus")
	switch statusStr {
	case "NEW":
		order.Status = types.OrderStatusNew
	case "PARTIALLY_FILLED":
		order.Status = types.OrderStatusPartiallyFilled
	case "FILLED":
		order.Status = types.OrderStatusFilled
	case "CANCELED", "CANCELLED":
		order.Status = types.OrderStatusCanceled
	case "EXPIRED":
		order.Status = types.OrderStatusExpired
	case "REJECTED":
		order.Status = types.OrderStatusRejected
	default:
		order.Status = types.OrderStatusNew
	}

	return order, nil
}

// getString 从 map 中获取字符串值，支持多个键名
func getString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key].(string); ok {
			return val
		}
	}
	return ""
}

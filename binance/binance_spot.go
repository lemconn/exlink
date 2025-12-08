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
func (s *BinanceSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return s.market.FetchMarkets(ctx)
}

// GetMarket 获取单个市场信息
func (s *BinanceSpot) GetMarket(symbol string) (*types.Market, error) {
	return s.market.GetMarket(symbol)
}

// FetchTicker 获取行情（单个）
func (s *BinanceSpot) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

// FetchTickers 批量获取行情
func (s *BinanceSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

// FetchOHLCV 获取K线数据
func (s *BinanceSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

// FetchBalance 获取余额
func (s *BinanceSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

// CreateOrder 创建订单
func (s *BinanceSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

// CancelOrder 取消订单
func (s *BinanceSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

// FetchOrder 查询订单
func (s *BinanceSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

// FetchOrders 查询订单列表
func (s *BinanceSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

// FetchOpenOrders 查询未成交订单
func (s *BinanceSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

// FetchTrades 获取交易记录（公共）
func (s *BinanceSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

// FetchMyTrades 获取我的交易记录
func (s *BinanceSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
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
	if !reload && len(m.binance.spotMarkets) > 0 {
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
				if !filter.MinQty.IsZero() {
					market.Limits.Amount.Min = filter.MinQty.InexactFloat64()
				}
				if !filter.MaxQty.IsZero() {
					market.Limits.Amount.Max = filter.MaxQty.InexactFloat64()
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
					market.Limits.Price.Min = filter.MinPrice.InexactFloat64()
				}
				if !filter.MaxPrice.IsZero() {
					market.Limits.Price.Max = filter.MaxPrice.InexactFloat64()
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
					market.Limits.Cost.Min = filter.MinNotional.InexactFloat64()
				}
			}
		}

		markets = append(markets, market)
	}

	// 存储市场信息
	m.binance.mu.Lock()
	if m.binance.spotMarkets == nil {
		m.binance.spotMarkets = make(map[string]*types.Market)
	}
	for _, market := range markets {
		m.binance.spotMarkets[market.Symbol] = market
	}
	m.binance.mu.Unlock()

	return nil
}

// FetchMarkets 获取市场列表
func (m *binanceSpotMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	markets := make([]*types.Market, 0, len(m.binance.spotMarkets))
	for _, market := range m.binance.spotMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

// GetMarket 获取单个市场信息
func (m *binanceSpotMarket) GetMarket(symbol string) (*types.Market, error) {
	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	market, ok := m.binance.spotMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

// FetchTicker 获取行情（单个）
func (m *binanceSpotMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
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
func (m *binanceSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	resp, err := m.binance.client.SpotClient.Get(ctx, "/api/v3/ticker/24hr", nil)
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

	// 如果需要过滤特定 symbols，先转换为 Binance 格式
	var binanceSymbols map[string]string
	if len(symbols) > 0 {
		binanceSymbols = make(map[string]string)
		for _, s := range symbols {
			market, err := m.GetMarket(s)
			if err == nil {
				binanceSymbols[market.ID] = s
			} else {
				// 如果市场未加载，尝试转换
				binanceSymbol, err := ToBinanceSymbol(s, false)
				if err == nil {
					binanceSymbols[binanceSymbol] = s
				}
			}
		}
	}

	tickers := make(map[string]*types.Ticker)
	for _, item := range data {
		// 如果指定了 symbols，进行过滤
		if len(symbols) > 0 {
			normalizedSymbol, ok := binanceSymbols[item.Symbol]
			if !ok {
				continue
			}
			ticker := &types.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: time.Now(),
			}
			ticker.Bid = item.BidPrice
			ticker.Ask = item.AskPrice
			ticker.Last = item.LastPrice
			ticker.Open = item.OpenPrice
			ticker.High = item.HighPrice
			ticker.Low = item.LowPrice
			ticker.Volume = item.Volume
			ticker.QuoteVolume = item.QuoteVolume
			tickers[normalizedSymbol] = ticker
		} else {
			// 如果没有指定 symbols，返回所有（使用 Binance 原始格式）
			ticker := &types.Ticker{
				Symbol:    item.Symbol,
				Timestamp: time.Now(),
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
		}
	}

	return tickers, nil
}

// FetchOHLCV 获取K线数据
func (m *binanceSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
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

// binanceSpotOrder 现货订单相关方法
type binanceSpotOrder struct {
	binance *Binance
}

// FetchBalance 获取余额
func (o *binanceSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
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
func (o *binanceSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 解析订单选项
	options := types.ApplyOrderOptions(opts...)

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
		reqParams["newClientOrderId"] = common.GenerateClientOrderID(o.binance.Name(), side)
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
	var respData types.BinanceSpotOrderResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal spot order response: %w", err)
	}

	// 解析数量
	origQty, _ := decimal.NewFromString(respData.OrigQty)
	executedQty, _ := decimal.NewFromString(respData.ExecutedQty)
	if origQty.IsZero() {
		origQty = amountDecimal
	}
	orderPrice, _ := decimal.NewFromString(respData.Price)
	cummulativeQuoteQty, _ := decimal.NewFromString(respData.CummulativeQuoteQty)

	// 计算平均价格和手续费
	var avgPrice decimal.Decimal
	var totalFee decimal.Decimal
	var feeCurrency string
	if len(respData.Fills) > 0 {
		var totalCost decimal.Decimal
		for _, fill := range respData.Fills {
			fillPrice, _ := decimal.NewFromString(fill.Price)
			fillQty, _ := decimal.NewFromString(fill.Qty)
			commission, _ := decimal.NewFromString(fill.Commission)
			totalCost = totalCost.Add(fillPrice.Mul(fillQty))
			totalFee = totalFee.Add(commission)
			if feeCurrency == "" {
				feeCurrency = fill.CommissionAsset
			}
		}
		if !executedQty.IsZero() {
			avgPrice = totalCost.Div(executedQty)
		}
	}

	// 构建订单对象
	order := &types.Order{
		ID:            strconv.FormatInt(respData.OrderID, 10),
		ClientOrderID: respData.ClientOrderID,
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Amount:        origQty.InexactFloat64(),
		Price:         orderPrice.InexactFloat64(),
		Filled:        executedQty.InexactFloat64(),
		Remaining:     origQty.Sub(executedQty).InexactFloat64(),
		Cost:          cummulativeQuoteQty.InexactFloat64(),
		Average:       avgPrice.InexactFloat64(),
		Timestamp:     time.UnixMilli(respData.TransactTime),
	}

	// 设置手续费
	if !totalFee.IsZero() && feeCurrency != "" {
		order.Fee = &types.Fee{
			Currency: feeCurrency,
			Cost:     totalFee.InexactFloat64(),
		}
	}

	// 转换状态
	switch respData.Status {
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

// FetchOrders 查询订单列表
func (o *binanceSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Binance 现货 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: Binance spot API does not support fetching order history directly")
}

// FetchOpenOrders 查询未成交订单
func (o *binanceSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}
	if symbol != "" {
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
		params["symbol"] = binanceSymbol
	}

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	resp, err := o.binance.client.SpotClient.Get(ctx, "/api/v3/openOrders", params)
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

// FetchTrades 获取交易记录（公共）
func (o *binanceSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
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

	params := map[string]interface{}{
		"symbol": binanceSymbol,
		"limit":  limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := o.binance.client.SpotClient.Get(ctx, "/api/v3/trades", params)
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
func (o *binanceSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
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
		"limit":     limit,
		"timestamp": timestamp,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	resp, err := o.binance.client.SpotClient.Get(ctx, "/api/v3/myTrades", params)
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

// getString 从 map 中获取字符串值，支持多个键名
func getString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key].(string); ok {
			return val
		}
	}
	return ""
}

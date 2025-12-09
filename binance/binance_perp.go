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
	"github.com/lemconn/exlink/types"
	"github.com/shopspring/decimal"
)

// BinancePerp Binance 永续合约实现
type BinancePerp struct {
	binance   *Binance
	market    *binancePerpMarket
	order     *binancePerpOrder
	hedgeMode bool
}

// NewBinancePerp 创建 Binance 永续合约实例
func NewBinancePerp(b *Binance) *BinancePerp {
	return &BinancePerp{
		binance:   b,
		market:    &binancePerpMarket{binance: b},
		order:     &binancePerpOrder{binance: b},
		hedgeMode: false,
	}
}

// ========== PerpExchange 接口实现 ==========

// LoadMarkets 加载市场信息
func (p *BinancePerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

// FetchMarkets 获取市场列表
func (p *BinancePerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return p.market.FetchMarkets(ctx)
}

// GetMarket 获取单个市场信息
func (p *BinancePerp) GetMarket(symbol string) (*model.Market, error) {
	return p.market.GetMarket(symbol)
}

// GetMarkets 从内存中获取所有市场信息
func (p *BinancePerp) GetMarkets() ([]*model.Market, error) {
	return p.market.GetMarkets()
}

// FetchTicker 获取行情（单个）
func (p *BinancePerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

// FetchTickers 批量获取行情
func (p *BinancePerp) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	return p.market.FetchTickers(ctx, symbols...)
}

// FetchOHLCV 获取K线数据
func (p *BinancePerp) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	return p.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

// FetchPositions 获取持仓
func (p *BinancePerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

// CreateOrder 创建订单
func (p *BinancePerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

// CancelOrder 取消订单
func (p *BinancePerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

// FetchOrder 查询订单
func (p *BinancePerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

// FetchOrders 查询订单列表
func (p *BinancePerp) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return p.order.FetchOrders(ctx, symbol, since, limit)
}

// FetchOpenOrders 查询未成交订单
func (p *BinancePerp) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return p.order.FetchOpenOrders(ctx, symbol)
}

// FetchTrades 获取交易记录（公共）
func (p *BinancePerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

// FetchMyTrades 获取我的交易记录
func (p *BinancePerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

// SetLeverage 设置杠杆
func (p *BinancePerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

// SetMarginMode 设置保证金模式
func (p *BinancePerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

// SetHedgeMode 设置双向持仓模式
func (p *BinancePerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

// IsHedgeMode 是否为双向持仓模式
func (p *BinancePerp) IsHedgeMode() bool {
	return p.hedgeMode
}

// 确保 BinancePerp 实现了 exchange.PerpExchange 接口
var _ exchange.PerpExchange = (*BinancePerp)(nil)

// ========== 内部实现 ==========

// binancePerpMarket 永续合约市场相关方法
type binancePerpMarket struct {
	binance *Binance
}

// LoadMarkets 加载市场信息
func (m *binancePerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.binance.mu.RLock()
	if !reload && len(m.binance.perpMarkets) > 0 {
		m.binance.mu.RUnlock()
		return nil
	}
	m.binance.mu.RUnlock()

	// 获取永续合约市场信息
	resp, err := m.binance.client.PerpClient.Get(ctx, "/fapi/v1/exchangeInfo", map[string]interface{}{
		"showPermissionSets": false,
	})
	if err != nil {
		return fmt.Errorf("fetch fapi exchange info: %w", err)
	}

	var info binancePerpMarketsResponse
	if err := json.Unmarshal(resp, &info); err != nil {
		return fmt.Errorf("unmarshal fapi exchange info: %w", err)
	}

	markets := make([]*model.Market, 0)
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

		market := &model.Market{
			ID:       s.Symbol,
			Symbol:   normalizedSymbol,
			Base:     s.BaseAsset,
			Quote:    s.QuoteAsset,
			Settle:   settle,
			Type:     model.MarketTypeSwap,
			Active:   s.Status == "TRADING",
			Contract: true,
			Linear:   true, // U本位永续合约
			Inverse:  false,
		}

		// 解析精度 - 合约订单优先使用 QuantityPrecision
		market.Precision.Amount = s.QuantityPrecision
		market.Precision.Price = s.PricePrecision

		// 解析限制
		for _, filter := range s.Filters {
			switch filter.FilterType {
			case "LOT_SIZE":
				if !filter.MinQty.IsZero() {
					market.Limits.Amount.Min = filter.MinQty
				}
				if !filter.MaxQty.IsZero() {
					market.Limits.Amount.Max = filter.MaxQty
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
	if m.binance.perpMarkets == nil {
		m.binance.perpMarkets = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.binance.perpMarkets[market.Symbol] = market
	}
	m.binance.mu.Unlock()

	return nil
}

// FetchMarkets 获取市场列表
func (m *binancePerpMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.binance.perpMarkets))
	for _, market := range m.binance.perpMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

// GetMarket 获取单个市场信息
func (m *binancePerpMarket) GetMarket(symbol string) (*model.Market, error) {
	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	market, ok := m.binance.perpMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

// GetMarkets 从内存中获取所有市场信息
func (m *binancePerpMarket) GetMarkets() ([]*model.Market, error) {
	m.binance.mu.RLock()
	defer m.binance.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.binance.perpMarkets))
	for _, market := range m.binance.perpMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

// FetchTicker 获取行情（单个）
func (m *binancePerpMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
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
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// 使用合约 API
	resp, err := m.binance.client.PerpClient.Get(ctx, "/fapi/v1/ticker/24hr", map[string]interface{}{
		"symbol": binanceSymbol,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var data binancePerpTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	// 转换回标准化格式 - 使用输入的symbol（已经是标准化格式）
	ticker := &model.Ticker{
		Symbol:    symbol, // 使用输入的标准化格式
		Timestamp: data.CloseTime,
	}

	// 注意：永续合约 API 可能不返回 bidPrice 和 askPrice，需要从其他接口获取
	// 这里先使用 lastPrice 作为默认值，或者留空
	ticker.Bid = data.LastPrice // 如果没有 bidPrice，使用 lastPrice 作为近似值
	ticker.Ask = data.LastPrice // 如果没有 askPrice，使用 lastPrice 作为近似值
	ticker.Last = data.LastPrice
	ticker.Open = data.OpenPrice
	ticker.High = data.HighPrice
	ticker.Low = data.LowPrice
	ticker.Volume = data.Volume
	ticker.QuoteVolume = data.QuoteVolume

	return ticker, nil
}

// FetchTickers 批量获取行情
func (m *binancePerpMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	resp, err := m.binance.client.PerpClient.Get(ctx, "/fapi/v1/ticker/24hr", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data []binancePerpTickerResponse

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
				binanceSymbol, err := ToBinanceSymbol(s, true)
				if err == nil {
					binanceSymbols[binanceSymbol] = s
				}
			}
		}
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range data {
		// 如果指定了 symbols，进行过滤
		if len(symbols) > 0 {
			normalizedSymbol, ok := binanceSymbols[item.Symbol]
			if !ok {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: item.CloseTime,
			}
			// 注意：永续合约 API 可能不返回 bidPrice 和 askPrice，使用 lastPrice 作为近似值
			ticker.Bid = item.LastPrice
			ticker.Ask = item.LastPrice
			ticker.Last = item.LastPrice
			ticker.Open = item.OpenPrice
			ticker.High = item.HighPrice
			ticker.Low = item.LowPrice
			ticker.Volume = item.Volume
			ticker.QuoteVolume = item.QuoteVolume
			tickers[normalizedSymbol] = ticker
		} else {
			// 如果没有指定 symbols，返回所有（使用 Binance 原始格式）
			ticker := &model.Ticker{
				Symbol:    item.Symbol,
				Timestamp: item.CloseTime,
			}
			// 注意：永续合约 API 可能不返回 bidPrice 和 askPrice，使用 lastPrice 作为近似值
			ticker.Bid = item.LastPrice
			ticker.Ask = item.LastPrice
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
func (m *binancePerpMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
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
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}
	params["symbol"] = binanceSymbol

	// 使用合约 API
	resp, err := m.binance.client.PerpClient.Get(ctx, "/fapi/v1/klines", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var data binancePerpKlineResponse
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

// binancePerpOrder 永续合约订单相关方法
type binancePerpOrder struct {
	binance         *Binance
	positionMode    *bool     // 持仓模式缓存: true=双向, false=单向
	positionModeExp time.Time // 持仓模式缓存过期时间
}

// FetchPositions 获取持仓
func (o *binancePerpOrder) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
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

	resp, err := o.binance.client.PerpClient.Get(ctx, "/fapi/v2/positionRisk", params)
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

		// 获取市场信息（通过 ID 查找）
		market, err := o.getMarketByID(item.Symbol)
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

// getMarketByID 通过交易所ID获取市场信息
func (o *binancePerpOrder) getMarketByID(id string) (*model.Market, error) {
	o.binance.mu.RLock()
	defer o.binance.mu.RUnlock()

	for _, market := range o.binance.perpMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

// CreateOrder 创建订单
func (o *binancePerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 解析订单选项
	options := types.ApplyOrderOptions(opts...)

	// 获取市场信息
	market, err := o.binance.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
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

	// 格式化数量（合约订单使用更保守的精度策略）
	// 合约订单：如果数量是整数，使用 0 精度；否则使用 1 位小数精度
	var amountPrecision int
	if amountDecimal.IsInteger() && amountDecimal.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		amountPrecision = 0
	} else {
		amountPrecision = 1
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

	// ========== 永续合约订单处理 ==========
	// 合约下单必须指定 PositionSide
	if options.PositionSide == nil {
		return nil, fmt.Errorf("contract order requires PositionSide (long/short)")
	}

	// 获取持仓模式
	isDualMode, err := o.getPositionMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("get position mode: %w", err)
	}

	if isDualMode {
		// 双向持仓模式
		// 开多/平多: positionSide=LONG
		// 开空/平空: positionSide=SHORT
		if *options.PositionSide == types.PositionSideLong {
			reqParams["positionSide"] = "LONG"
		} else {
			reqParams["positionSide"] = "SHORT"
		}
	} else {
		// 单向持仓模式
		reqParams["positionSide"] = "BOTH"

		// 判断是否为平仓操作
		// 平多：PositionSideLong + SideSell -> reduceOnly = true
		// 平空：PositionSideShort + SideBuy -> reduceOnly = true
		// 开多：PositionSideLong + SideBuy -> 不设置 reduceOnly
		// 开空：PositionSideShort + SideSell -> 不设置 reduceOnly
		if (*options.PositionSide == types.PositionSideLong && side == types.OrderSideSell) ||
			(*options.PositionSide == types.PositionSideShort && side == types.OrderSideBuy) {
			reqParams["reduceOnly"] = "true"
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

	// 发送请求（合约订单使用 PerpClient）
	resp, err := o.binance.client.PerpClient.Request(ctx, "POST", "/fapi/v1/order", reqParams, nil)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 解析响应（合约订单响应）
	var respData types.BinanceContractOrderResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal contract order response: %w", err)
	}

	// 解析数量
	origQty, _ := decimal.NewFromString(respData.OrigQty)
	executedQty, _ := decimal.NewFromString(respData.ExecutedQty)
	if origQty.IsZero() {
		origQty = amountDecimal
	}
	orderPrice, _ := decimal.NewFromString(respData.Price)
	avgPrice, _ := decimal.NewFromString(respData.AvgPrice)
	cumQuote, _ := decimal.NewFromString(respData.CumQuote)

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
		Cost:          cumQuote.InexactFloat64(),
		Average:       avgPrice.InexactFloat64(),
		Timestamp:     time.UnixMilli(respData.UpdateTime),
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
func (o *binancePerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	if o.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	// 获取市场信息
	market, err := o.binance.perp.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
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

	_, err = o.binance.client.PerpClient.Post(ctx, "/fapi/v1/order", params)
	return err
}

// FetchOrder 查询订单
func (o *binancePerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 获取市场信息
	market, err := o.binance.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
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

	// 使用合约 API
	resp, err := o.binance.client.PerpClient.Get(ctx, "/fapi/v1/order", params)
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

	// 提取时间戳（合约使用 updateTime）
	var timestampInt int64
	if t, ok := data["updateTime"].(float64); ok {
		timestampInt = int64(t)
	} else if t, ok := data["time"].(float64); ok {
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
func (o *binancePerpOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Binance 合约 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: Binance perp API does not support fetching order history directly")
}

// FetchOpenOrders 查询未成交订单
func (o *binancePerpOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}
	if symbol != "" {
		// 获取市场信息
		market, err := o.binance.perp.market.GetMarket(symbol)
		if err != nil {
			return nil, err
		}

		// 获取交易所格式的 symbol ID
		binanceSymbol := market.ID
		if binanceSymbol == "" {
			var err error
			binanceSymbol, err = ToBinanceSymbol(symbol, true)
			if err != nil {
				return nil, fmt.Errorf("get market ID: %w", err)
			}
		}
		params["symbol"] = binanceSymbol
	}

	queryString := BuildQueryString(params)
	signature := o.binance.signer.Sign(queryString)
	params["signature"] = signature

	resp, err := o.binance.client.PerpClient.Get(ctx, "/fapi/v1/openOrders", params)
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
		UpdateTime    int64  `json:"updateTime"`
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
			Timestamp:     time.UnixMilli(item.UpdateTime),
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
func (o *binancePerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.binance.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
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

	resp, err := o.binance.client.PerpClient.Get(ctx, "/fapi/v1/trades", params)
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
func (o *binancePerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	if o.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 获取市场信息
	market, err := o.binance.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
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

	resp, err := o.binance.client.PerpClient.Get(ctx, "/fapi/v1/userTrades", params)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var data []struct {
		ID              int64  `json:"id"`
		OrderID         int64  `json:"orderId"`
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Time            int64  `json:"time"`
		IsBuyer         bool   `json:"buyer"`
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

// SetLeverage 设置杠杆
func (o *binancePerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if o.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	market, err := o.binance.perp.market.GetMarket(symbol)
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

	queryString := BuildQueryString(reqParams)
	signature := o.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	_, err = o.binance.client.PerpClient.Post(ctx, "/fapi/v1/leverage", reqParams)
	return err
}

// SetMarginMode 设置保证金模式
func (o *binancePerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	if o.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	market, err := o.binance.perp.market.GetMarket(symbol)
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

	queryString := BuildQueryString(reqParams)
	signature := o.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	_, err = o.binance.client.PerpClient.Post(ctx, "/fapi/v1/marginType", reqParams)
	return err
}

// getPositionMode 获取持仓模式（带缓存）
// 返回: true=双向持仓(hedge mode), false=单向持仓(one-way mode)
func (o *binancePerpOrder) getPositionMode(ctx context.Context) (bool, error) {
	// 检查缓存是否有效（5分钟）
	if o.positionMode != nil && time.Now().Before(o.positionModeExp) {
		return *o.positionMode, nil
	}

	// 查询持仓模式
	reqTimestamp := common.GetTimestamp()
	reqParams := map[string]interface{}{
		"timestamp": reqTimestamp,
	}

	queryString := BuildQueryString(reqParams)
	signature := o.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	resp, err := o.binance.client.PerpClient.Request(ctx, "GET", "/fapi/v1/positionSide/dual", reqParams, nil)
	if err != nil {
		return false, fmt.Errorf("get position mode: %w", err)
	}

	var result struct {
		DualSidePosition bool `json:"dualSidePosition"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("unmarshal position mode: %w", err)
	}

	// 缓存结果
	o.positionMode = &result.DualSidePosition
	o.positionModeExp = time.Now().Add(5 * time.Minute)

	return result.DualSidePosition, nil
}

package bybit

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

// BybitPerp Bybit 永续合约实现
type BybitPerp struct {
	bybit     *Bybit
	market    *bybitPerpMarket
	order     *bybitPerpOrder
	hedgeMode bool
}

// NewBybitPerp 创建 Bybit 永续合约实例
func NewBybitPerp(b *Bybit) *BybitPerp {
	return &BybitPerp{
		bybit:     b,
		market:    &bybitPerpMarket{bybit: b},
		order:     &bybitPerpOrder{bybit: b},
		hedgeMode: false,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *BybitPerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

func (p *BybitPerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return p.market.FetchMarkets(ctx)
}

func (p *BybitPerp) GetMarket(symbol string) (*model.Market, error) {
	return p.market.GetMarket(symbol)
}

func (p *BybitPerp) GetMarkets() ([]*model.Market, error) {
	return p.market.GetMarkets()
}

func (p *BybitPerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

func (p *BybitPerp) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	return p.market.FetchTickers(ctx, symbols...)
}

func (p *BybitPerp) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	return p.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (p *BybitPerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *BybitPerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *BybitPerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *BybitPerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *BybitPerp) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return p.order.FetchOrders(ctx, symbol, since, limit)
}

func (p *BybitPerp) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return p.order.FetchOpenOrders(ctx, symbol)
}

func (p *BybitPerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

func (p *BybitPerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

func (p *BybitPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *BybitPerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

func (p *BybitPerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

func (p *BybitPerp) IsHedgeMode() bool {
	return p.hedgeMode
}

var _ exchange.PerpExchange = (*BybitPerp)(nil)

// ========== 内部实现 ==========

type bybitPerpMarket struct {
	bybit *Bybit
}

func (m *bybitPerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.bybit.mu.RLock()
	if !reload && len(m.bybit.perpMarkets) > 0 {
		m.bybit.mu.RUnlock()
		return nil
	}
	m.bybit.mu.RUnlock()

	// 获取永续合约市场信息
	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/instruments-info", map[string]interface{}{
		"category": "linear",
	})
	if err != nil {
		return fmt.Errorf("fetch swap markets: %w", err)
	}

	var result bybitPerpMarketsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("unmarshal swap markets: %w", err)
	}

	if result.RetCode != 0 {
		return fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	markets := make([]*model.Market, 0)
	for _, s := range result.Result.List {
		if s.Status != "Trading" {
			continue
		}

		// Bybit linear 合约的 settle 通常是 quoteCoin
		settle := s.QuoteCoin

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(s.BaseCoin, s.QuoteCoin, settle)

		market := &model.Market{
			ID:       s.Symbol,
			Symbol:   normalizedSymbol,
			Base:     s.BaseCoin,
			Quote:    s.QuoteCoin,
			Settle:   settle,
			Type:     model.MarketTypeSwap,
			Active:   s.Status == "Trading",
			Contract: true,
			Linear:   true, // U本位永续合约
			Inverse:  false,
		}

		// 解析精度
		basePrecision := s.LotSizeFilter.BasePrecision.InexactFloat64()
		tickSize := s.PriceFilter.TickSize.InexactFloat64()
		quotePrecision := s.LotSizeFilter.QuotePrecision.InexactFloat64()

		market.Precision.Amount = getPrecisionDigits(basePrecision)
		if tickSize > 0 {
			market.Precision.Price = getPrecisionDigits(tickSize)
		} else if quotePrecision > 0 {
			market.Precision.Price = getPrecisionDigits(quotePrecision)
		}

		// 解析限制
		market.Limits.Amount.Min = s.LotSizeFilter.MinOrderQty
		market.Limits.Amount.Max = s.LotSizeFilter.MaxOrderQty
		market.Limits.Cost.Min = s.LotSizeFilter.MinOrderAmt
		market.Limits.Cost.Max = s.LotSizeFilter.MaxOrderAmt

		markets = append(markets, market)
	}

	// 存储市场信息
	m.bybit.mu.Lock()
	if m.bybit.perpMarkets == nil {
		m.bybit.perpMarkets = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.bybit.perpMarkets[market.Symbol] = market
	}
	m.bybit.mu.Unlock()

	return nil
}

func (m *bybitPerpMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.bybit.perpMarkets))
	for _, market := range m.bybit.perpMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *bybitPerpMarket) GetMarket(symbol string) (*model.Market, error) {
	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	market, ok := m.bybit.perpMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

func (m *bybitPerpMarket) GetMarkets() ([]*model.Market, error) {
	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.bybit.perpMarkets))
	for _, market := range m.bybit.perpMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *bybitPerpMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"symbol":   bybitSymbol,
		"category": "linear",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var result bybitPerpTickerResponse

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
	ticker := &model.Ticker{
		Symbol:    symbol,
		Timestamp: result.Time,
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

func (m *bybitPerpMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"category": "linear",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var result bybitPerpTickerResponse

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	// 如果需要过滤特定 symbols，先转换为 Bybit 格式
	var bybitSymbols map[string]string
	if len(symbols) > 0 {
		bybitSymbols = make(map[string]string)
		for _, s := range symbols {
			market, err := m.GetMarket(s)
			if err == nil {
				bybitSymbols[market.ID] = s
			} else {
				// 如果市场未加载，尝试转换
				bybitSymbol, err := ToBybitSymbol(s, true)
				if err == nil {
					bybitSymbols[bybitSymbol] = s
				}
			}
		}
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range result.Result.List {
		// 如果指定了 symbols，进行过滤
		if len(symbols) > 0 {
			normalizedSymbol, ok := bybitSymbols[item.Symbol]
			if !ok {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: result.Time,
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
		} else {
			// 如果没有指定 symbols，尝试从市场信息中查找
			market, err := m.getMarketByID(item.Symbol)
			if err != nil {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    market.Symbol,
				Timestamp: result.Time,
			}
			ticker.Bid = item.Bid1Price
			ticker.Ask = item.Ask1Price
			ticker.Last = item.LastPrice
			ticker.Open = item.PrevPrice24h
			ticker.High = item.HighPrice24h
			ticker.Low = item.LowPrice24h
			ticker.Volume = item.Volume24h
			ticker.QuoteVolume = item.Turnover24h
			tickers[market.Symbol] = ticker
		}
	}

	return tickers, nil
}

// getMarketByID 通过交易所ID获取市场信息
func (m *bybitPerpMarket) getMarketByID(id string) (*model.Market, error) {
	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	for _, market := range m.bybit.perpMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (m *bybitPerpMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.BybitTimeframe(timeframe)

	params := map[string]interface{}{
		"symbol":   market.ID,
		"category": "linear",
		"interval": normalizedTimeframe,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["start"] = since.UnixMilli()
	}

	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/kline", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var result bybitPerpKlineResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	ohlcvs := make(model.OHLCVs, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		ohlcv := &model.OHLCV{
			Timestamp: item.StartTime,
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

type bybitPerpOrder struct {
	bybit           *Bybit
	positionMode    *bool     // 持仓模式缓存: true=双向, false=单向
	positionModeExp time.Time // 持仓模式缓存过期时间
}

// signAndRequest 签名并发送请求（Bybit v5 API）
func (o *bybitPerpOrder) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if o.bybit.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	signature, timestamp := o.bybit.signer.SignRequest(method, params, body)
	recvWindow := "5000"

	// 设置请求头
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-API-KEY", o.bybit.client.APIKey)
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-TIMESTAMP", timestamp)
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-RECV-WINDOW", recvWindow)
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-SIGN", signature)
	o.bybit.client.HTTPClient.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return o.bybit.client.HTTPClient.Get(ctx, path, params)
	} else {
		return o.bybit.client.HTTPClient.Post(ctx, path, body)
	}
}

func (o *bybitPerpOrder) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	params := map[string]interface{}{
		"category": "linear",
	}

	if len(symbols) > 0 {
		// Bybit 需要 symbol 参数
		market, err := o.bybit.perp.market.GetMarket(symbols[0])
		if err == nil {
			params["symbol"] = market.ID
		} else {
			// 如果市场未加载，尝试转换
			bybitSymbol, err := ToBybitSymbol(symbols[0], true)
			if err == nil {
				params["symbol"] = bybitSymbol
			}
		}
	}

	resp, err := o.signAndRequest(ctx, "GET", "/v5/position/list", params, nil)
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

		market, err := o.getMarketByID(item.Symbol)
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

// getMarketByID 通过交易所ID获取市场信息
func (o *bybitPerpOrder) getMarketByID(id string) (*model.Market, error) {
	o.bybit.mu.RLock()
	defer o.bybit.mu.RUnlock()

	for _, market := range o.bybit.perpMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (o *bybitPerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
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

	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// Bybit API requires "Buy" or "Sell" (capitalized)
	sideStr := string(side)
	if len(sideStr) > 0 {
		sideStr = strings.ToUpper(sideStr[:1]) + strings.ToLower(sideStr[1:])
	}

	reqBody := map[string]interface{}{
		"category": "linear",
		"symbol":   bybitSymbol,
		"side":     sideStr,
	}

	// 格式化数量
	precision := market.Precision.Amount
	if precision <= 0 {
		precision = 8
	}
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	reqBody["qty"] = strconv.FormatFloat(amountFloat, 'f', precision, 64)

	if orderType == types.OrderTypeLimit {
		reqBody["orderType"] = "Limit"
		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
		pricePrecision := market.Precision.Price
		if pricePrecision <= 0 {
			pricePrecision = 8
		}
		reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', pricePrecision, 64)

		// 处理 timeInForce
		if options.TimeInForce != nil {
			reqBody["timeInForce"] = strings.ToUpper(string(*options.TimeInForce))
		} else {
			reqBody["timeInForce"] = "GTC"
		}
	} else {
		reqBody["orderType"] = "Market"
	}

	// 合约订单处理持仓方向
	if options.PositionSide == nil {
		return nil, fmt.Errorf("contract order requires PositionSide (long/short)")
	}

	// 获取持仓模式
	isDualMode, err := o.getPositionMode(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("get position mode: %w", err)
	}

	if isDualMode {
		// 双向持仓模式
		// 开多/平多: positionIdx=1
		// 开空/平空: positionIdx=2
		if *options.PositionSide == types.PositionSideLong {
			reqBody["positionIdx"] = 1
		} else {
			reqBody["positionIdx"] = 2
		}
	} else {
		// 单向持仓模式
		reqBody["positionIdx"] = 0

		// 判断是否为平仓操作
		// 平多：PositionSideLong + SideSell -> reduceOnly = true
		// 平空：PositionSideShort + SideBuy -> reduceOnly = true
		if (*options.PositionSide == types.PositionSideLong && side == types.OrderSideSell) ||
			(*options.PositionSide == types.PositionSideShort && side == types.OrderSideBuy) {
			reqBody["reduceOnly"] = true
		} else {
			reqBody["reduceOnly"] = false
		}
	}

	// 客户端订单ID
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqBody["orderLinkId"] = *options.ClientOrderID
	} else {
		reqBody["orderLinkId"] = common.GenerateClientOrderID(o.bybit.Name(), side)
	}

	resp, err := o.signAndRequest(ctx, "POST", "/v5/order/create", nil, reqBody)
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

	amountFloat, _ = strconv.ParseFloat(amount, 64)
	var priceFloat float64
	if priceStr != "" {
		priceFloat, _ = strconv.ParseFloat(priceStr, 64)
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

// parseOrder 解析订单数据（合约版本）
func (o *bybitPerpOrder) parseOrder(item struct {
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

func (o *bybitPerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// 获取市场信息
	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"category": "linear",
		"symbol":   bybitSymbol,
		"orderId":  orderID,
	}

	_, err = o.signAndRequest(ctx, "POST", "/v5/order/cancel", nil, reqBody)
	return err
}

func (o *bybitPerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// 获取市场信息
	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   bybitSymbol,
		"orderId":  orderID,
	}

	// First try to fetch from open orders (realtime)
	resp, err := o.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
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
					return o.parseOrder(item, symbol), nil
				}
			}
		}
	}

	// If not found in open orders, try history
	resp, err = o.signAndRequest(ctx, "GET", "/v5/order/history", params, nil)
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
			return o.parseOrder(item, symbol), nil
		}
	}

	return nil, fmt.Errorf("order not found")
}

func (o *bybitPerpOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Bybit 合约 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: Bybit perp API does not support fetching order history directly")
}

func (o *bybitPerpOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	params := map[string]interface{}{
		"category": "linear",
	}
	if symbol != "" {
		// 获取市场信息
		market, err := o.bybit.perp.market.GetMarket(symbol)
		if err != nil {
			return nil, err
		}

		// 获取交易所格式的 symbol ID
		bybitSymbol := market.ID
		if bybitSymbol == "" {
			var err error
			bybitSymbol, err = ToBybitSymbol(symbol, true)
			if err != nil {
				return nil, fmt.Errorf("get market ID: %w", err)
			}
		}
		params["symbol"] = bybitSymbol
	}

	resp, err := o.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
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
				CreatedTime string `json:"createdTime"`
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
		normalizedSymbol := symbol
		if symbol == "" {
			// 如果没有提供symbol，尝试从市场信息中查找
			market, err := o.getMarketByID(item.OrderID)
			if err == nil {
				normalizedSymbol = market.Symbol
			} else {
				normalizedSymbol = item.OrderID // 临时使用原格式
			}
		}
		orders = append(orders, o.parseOrder(item, normalizedSymbol))
	}

	return orders, nil
}

func (o *bybitPerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   bybitSymbol,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := o.bybit.client.HTTPClient.Get(ctx, "/v5/market/recent-trade", params)
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
	for i, item := range result.Result.List {
		price, _ := strconv.ParseFloat(item.Price, 64)
		size, _ := strconv.ParseFloat(item.Size, 64)
		execTime, _ := strconv.ParseInt(item.ExecTime, 10, 64)

		trade := &types.Trade{
			ID:        strconv.FormatInt(execTime, 10) + "-" + strconv.Itoa(i),
			Symbol:    symbol,
			Price:     price,
			Amount:    size,
			Cost:      price * size,
			Timestamp: time.UnixMilli(execTime),
		}

		if strings.ToUpper(item.Side) == "BUY" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func (o *bybitPerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   bybitSymbol,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := o.signAndRequest(ctx, "GET", "/v5/execution/list", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				ExecID      string `json:"execId"`
				OrderID     string `json:"orderId"`
				ExecPrice   string `json:"execPrice"`
				ExecQty     string `json:"execQty"`
				ExecTime    string `json:"execTime"`
				Side        string `json:"side"`
				Fee         string `json:"execFee"`
				FeeCurrency string `json:"feeCurrencyId"`
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
		price, _ := strconv.ParseFloat(item.ExecPrice, 64)
		qty, _ := strconv.ParseFloat(item.ExecQty, 64)
		execTime, _ := strconv.ParseInt(item.ExecTime, 10, 64)
		fee, _ := strconv.ParseFloat(item.Fee, 64)

		trade := &types.Trade{
			ID:        item.ExecID,
			OrderID:   item.OrderID,
			Symbol:    symbol,
			Price:     price,
			Amount:    qty,
			Cost:      price * qty,
			Timestamp: time.UnixMilli(execTime),
		}

		if strings.ToUpper(item.Side) == "BUY" {
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

func (o *bybitPerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("leverage only supported for contracts")
	}

	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"category":     "linear",
		"symbol":       bybitSymbol,
		"buyLeverage":  strconv.Itoa(leverage),
		"sellLeverage": strconv.Itoa(leverage),
	}

	_, err = o.signAndRequest(ctx, "POST", "/v5/position/set-leverage", nil, reqBody)
	return err
}

func (o *bybitPerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	market, err := o.bybit.perp.market.GetMarket(symbol)
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

	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"category":  "linear",
		"symbol":    bybitSymbol,
		"tradeMode": strings.ToUpper(mode),
	}

	_, err = o.signAndRequest(ctx, "POST", "/v5/position/switch-mode", nil, reqBody)
	return err
}

// getPositionMode 获取持仓模式（带缓存）
// 返回: true=双向持仓, false=单向持仓
func (o *bybitPerpOrder) getPositionMode(ctx context.Context, symbol string) (bool, error) {
	// 检查缓存是否有效（5分钟）
	if o.positionMode != nil && time.Now().Before(o.positionModeExp) {
		return *o.positionMode, nil
	}

	// 获取交易所格式的 symbol ID
	market, err := o.bybit.perp.market.GetMarket(symbol)
	if err != nil {
		return false, fmt.Errorf("get market: %w", err)
	}

	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return false, fmt.Errorf("get market ID: %w", err)
		}
	}

	// 尝试切换到单向持仓模式（mode=0）
	reqBody := map[string]interface{}{
		"category": "linear",
		"symbol":   bybitSymbol,
		"mode":     0, // 0=单向, 3=双向
	}

	resp, err := o.signAndRequest(ctx, "POST", "/v5/position/switch-mode", nil, reqBody)
	if err != nil {
		return false, fmt.Errorf("switch position mode: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("unmarshal position mode: %w", err)
	}

	var isDualMode bool
	if result.RetCode == 110025 || strings.Contains(result.RetMsg, "Position mode is not modified") {
		// 当前已经是单向持仓模式
		isDualMode = false
	} else if result.RetCode == 0 || result.RetMsg == "OK" {
		// 切换成功，说明之前是双向持仓，需要切换回去
		isDualMode = true
		// 切换回双向持仓模式
		reqBody["mode"] = 3
		_, err := o.signAndRequest(ctx, "POST", "/v5/position/switch-mode", nil, reqBody)
		if err != nil {
			return false, fmt.Errorf("restore position mode: %w", err)
		}
	} else {
		return false, fmt.Errorf("unexpected response: code=%d, msg=%s", result.RetCode, result.RetMsg)
	}

	// 缓存结果
	o.positionMode = &isDualMode
	o.positionModeExp = time.Now().Add(5 * time.Minute)

	return isDualMode, nil
}

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
	"github.com/lemconn/exlink/option"
	"github.com/lemconn/exlink/types"
)

// BybitPerp Bybit 永续合约实现
type BybitPerp struct {
	bybit *Bybit
}

// NewBybitPerp 创建 Bybit 永续合约实例
func NewBybitPerp(b *Bybit) *BybitPerp {
	return &BybitPerp{
		bybit: b,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *BybitPerp) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	p.bybit.mu.RLock()
	if !reload && len(p.bybit.perpMarketsBySymbol) > 0 {
		p.bybit.mu.RUnlock()
		return nil
	}
	p.bybit.mu.RUnlock()

	// 获取永续合约市场信息
	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/instruments-info", map[string]interface{}{
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
	p.bybit.mu.Lock()
	if p.bybit.perpMarketsBySymbol == nil {
		p.bybit.perpMarketsBySymbol = make(map[string]*model.Market)
		p.bybit.perpMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		p.bybit.perpMarketsBySymbol[market.Symbol] = market
		p.bybit.perpMarketsByID[market.ID] = market
	}
	p.bybit.mu.Unlock()

	return nil
}

func (p *BybitPerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	p.bybit.mu.RLock()
	defer p.bybit.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.bybit.perpMarketsBySymbol))
	for _, market := range p.bybit.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *BybitPerp) GetMarket(symbol string) (*model.Market, error) {
	p.bybit.mu.RLock()
	defer p.bybit.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := p.bybit.perpMarketsBySymbol[symbol]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := p.bybit.perpMarketsByID[symbol]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", symbol)
}

func (p *BybitPerp) GetMarkets() ([]*model.Market, error) {
	p.bybit.mu.RLock()
	defer p.bybit.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.bybit.perpMarketsBySymbol))
	for _, market := range p.bybit.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *BybitPerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := p.GetMarket(symbol)
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

	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
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
	ticker.Timestamp = result.Time

	return ticker, nil
}

func (p *BybitPerp) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
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

	tickers := make(map[string]*model.Ticker)
	for _, item := range result.Result.List {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.Symbol)
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
		ticker.Timestamp = result.Time
		tickers[market.Symbol] = ticker
	}

	return tickers, nil
}

func (p *BybitPerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
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

	// 获取市场信息
	market, err := p.GetMarket(symbol)
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

	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/kline", params)
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

func (p *BybitPerp) FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}
	symbols := []string{}
	if argsOpts.Symbols != nil {
		symbols = argsOpts.Symbols
	}
	return p.fetchPositions(ctx, symbols...)
}

// ========== 内部辅助方法 ==========

// signAndRequest 签名并发送请求（Bybit v5 API）
func (p *BybitPerp) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if p.bybit.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	signature, timestamp := p.bybit.signer.SignRequest(method, params, body)
	recvWindow := "5000"

	// 设置请求头
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-API-KEY", p.bybit.client.APIKey)
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-TIMESTAMP", timestamp)
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-RECV-WINDOW", recvWindow)
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-SIGN", signature)
	p.bybit.client.HTTPClient.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return p.bybit.client.HTTPClient.Get(ctx, path, params)
	} else {
		return p.bybit.client.HTTPClient.Post(ctx, path, body)
	}
}

func (p *BybitPerp) fetchPositions(ctx context.Context, symbols ...string) (model.Positions, error) {
	params := map[string]interface{}{
		"category": "linear",
	}

	if len(symbols) > 0 {
		// Bybit 需要 symbol 参数
		market, err := p.GetMarket(symbols[0])
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

	resp, err := p.signAndRequest(ctx, "GET", "/v5/position/list", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var result bybitPerpPositionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	positions := make([]*model.Position, 0)
	for _, item := range result.Result.List {
		size, _ := item.Size.Float64()
		if size == 0 {
			continue
		}

		market, err := p.GetMarket(item.Symbol)
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

		var side string
		if strings.ToUpper(item.Side) == "BUY" {
			side = string(types.PositionSideLong)
		} else {
			side = string(types.PositionSideShort)
		}

		position := &model.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           item.Size,
			EntryPrice:       item.AvgPrice,
			MarkPrice:        item.MarkPrice,
			UnrealizedPnl:    item.UnrealisedPnl,
			LiquidationPrice: item.LiqPrice,
			RealizedPnl:      item.CumRealisedPnl,
			Leverage:         item.Leverage,
			Margin:           item.PositionIM,
			Percentage:       types.ExDecimal{},
			Timestamp:        item.UpdatedTime,
		}

		positions = append(positions, position)
	}

	return positions, nil
}

func (p *BybitPerp) CreateOrder(ctx context.Context, symbol string, side option.PerpOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	// 解析选项
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 判断订单类型：优先使用 argsOpts.OrderType，如果未设置则默认为 Market
	var orderType option.OrderType
	if argsOpts.OrderType != nil {
		orderType = *argsOpts.OrderType
	} else {
		// 默认使用 Market
		orderType = option.Market
	}

	// 如果订单类型为 Limit，必须设置价格
	var priceStr string
	if orderType == option.Limit {
		if argsOpts.Price == nil || *argsOpts.Price == "" {
			return nil, fmt.Errorf("limit order requires price")
		}
		priceStr = *argsOpts.Price
	} else if argsOpts.Price != nil && *argsOpts.Price != "" {
		// 市价单也可以设置价格（用于某些交易所的限价市价单）
		priceStr = *argsOpts.Price
	} else {
		priceStr = ""
	}

	market, err := p.GetMarket(symbol)
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
	sideStr := side.ToSide()
	if len(sideStr) > 0 {
		sideStr = strings.ToUpper(sideStr[:1]) + strings.ToLower(sideStr[1:])
	}

	// 构建请求结构体
	req := bybitPerpCreateOrderRequest{
		Category: "linear",
		Symbol:   bybitSymbol,
		Side:     sideStr,
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
	req.Qty = strconv.FormatFloat(amountFloat, 'f', precision, 64)

	if orderType == option.Limit {
		req.OrderType = "Limit"
		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
		pricePrecision := market.Precision.Price
		if pricePrecision <= 0 {
			pricePrecision = 8
		}
		req.Price = strconv.FormatFloat(priceFloat, 'f', pricePrecision, 64)

		// 处理 timeInForce
		if argsOpts.TimeInForce != nil {
			req.TimeInForce = argsOpts.TimeInForce.Upper()
		} else {
			req.TimeInForce = "GTC"
		}
	} else {
		req.OrderType = "Market"
	}

	// 从 PerpOrderSide 自动推断 PositionSide 和 reduceOnly
	positionSideStr := side.ToPositionSide()
	reduceOnly := side.ToReduceOnly()

	// 获取持仓模式：从 hedgeMode 选项获取，未设置时默认为 false（单向持仓模式）
	isDualMode := false
	if argsOpts.HedgeMode != nil {
		isDualMode = *argsOpts.HedgeMode
	}

	if isDualMode {
		// 双向持仓模式
		// 开多/平多: positionIdx=1
		// 开空/平空: positionIdx=2
		if positionSideStr == "LONG" {
			req.PositionIdx = 1
		} else {
			req.PositionIdx = 2
		}
	} else {
		// 单向持仓模式
		req.PositionIdx = 0

		// 如果是平仓操作，设置 reduceOnly
		req.ReduceOnly = reduceOnly
	}

	// 客户端订单ID
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		req.OrderLinkID = *argsOpts.ClientOrderID
	} else {
		// 将 PerpOrderSide 转换为 OrderSide 用于生成订单ID
		orderSide := types.OrderSide(strings.ToLower(side.ToSide()))
		req.OrderLinkID = common.GenerateClientOrderID(p.bybit.Name(), orderSide)
	}

	// 将结构体转换为 map
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var reqBody map[string]interface{}
	if err := json.Unmarshal(reqBytes, &reqBody); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	resp, err := p.signAndRequest(ctx, "POST", "/v5/order/create", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result bybitPerpCreateOrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	// 构建 NewOrder 对象
	perpOrder := &model.NewOrder{
		Symbol:        symbol,
		OrderId:       result.Result.OrderID,
		ClientOrderID: result.Result.OrderLinkID,
		Timestamp:     result.Time,
	}

	return perpOrder, nil
}

// parsePerpOrder 将 Bybit 响应转换为 model.PerpOrder
func (p *BybitPerp) parsePerpOrder(item bybitPerpFetchOrderItem, symbol string) *model.PerpOrder {
	// 确定 positionSide
	var positionSide string
	switch item.PositionIdx {
	case 1:
		positionSide = "LONG"
	case 2:
		positionSide = "SHORT"
	default:
		positionSide = "BOTH"
	}

	order := &model.PerpOrder{
		ID:               item.OrderID,
		ClientID:         item.OrderLinkID,
		Type:             item.OrderType,
		Side:             item.Side,
		PositionSide:     positionSide,
		Symbol:           symbol,
		Price:            item.Price,
		AvgPrice:         item.AvgPrice,
		Quantity:         item.Qty,
		ExecutedQuantity: item.CumExecQty,
		Status:           item.OrderStatus,
		TimeInForce:      item.TimeInForce,
		ReduceOnly:       item.ReduceOnly,
		CreateTime:       item.CreatedTime,
		UpdateTime:       item.UpdatedTime,
	}

	return order
}

func (p *BybitPerp) CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
	}

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		reqBody["orderId"] = orderId
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		reqBody["orderLinkId"] = *argsOpts.ClientOrderID
	} else {
		return fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	_, err = p.signAndRequest(ctx, "POST", "/v5/order/cancel", nil, reqBody)
	return err
}

func (p *BybitPerp) FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.PerpOrder, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
	}

	// 确定要匹配的 ID
	var targetOrderID string
	var targetOrderLinkID string

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		params["orderId"] = orderId
		targetOrderID = orderId
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["orderLinkId"] = *argsOpts.ClientOrderID
		targetOrderLinkID = *argsOpts.ClientOrderID
	} else {
		return nil, fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	// First try to fetch from open orders (realtime)
	resp, err := p.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
	if err == nil {
		var realtimeResult bybitPerpFetchOrderResponse

		if err := json.Unmarshal(resp, &realtimeResult); err == nil && realtimeResult.RetCode == 0 {
			for _, item := range realtimeResult.Result.List {
				if targetOrderID != "" && item.OrderID == targetOrderID {
					return p.parsePerpOrder(item, symbol), nil
				}
				if targetOrderLinkID != "" && item.OrderLinkID == targetOrderLinkID {
					return p.parsePerpOrder(item, symbol), nil
				}
			}
		}
	}

	// If not found in open orders, try history
	resp, err = p.signAndRequest(ctx, "GET", "/v5/order/history", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var result bybitPerpFetchOrderResponse

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
		if targetOrderID != "" && item.OrderID == targetOrderID {
			return p.parsePerpOrder(item, symbol), nil
		}
		if targetOrderLinkID != "" && item.OrderLinkID == targetOrderLinkID {
			return p.parsePerpOrder(item, symbol), nil
		}
	}

	return nil, fmt.Errorf("order not found")
}

func (p *BybitPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := p.GetMarket(symbol)
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

	_, err = p.signAndRequest(ctx, "POST", "/v5/position/set-leverage", nil, reqBody)
	return err
}

func (p *BybitPerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	market, err := p.GetMarket(symbol)
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

	_, err = p.signAndRequest(ctx, "POST", "/v5/position/switch-mode", nil, reqBody)
	return err
}

var _ exchange.PerpExchange = (*BybitPerp)(nil)

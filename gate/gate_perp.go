package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/types"
	"github.com/shopspring/decimal"
)

// GatePerp Gate 永续合约实现
type GatePerp struct {
	gate      *Gate
	market    *gatePerpMarket
	order     *gatePerpOrder
	hedgeMode bool
}

// NewGatePerp 创建 Gate 永续合约实例
func NewGatePerp(g *Gate) *GatePerp {
	return &GatePerp{
		gate:      g,
		market:    &gatePerpMarket{gate: g},
		order:     &gatePerpOrder{gate: g},
		hedgeMode: false,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *GatePerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

func (p *GatePerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return p.market.FetchMarkets(ctx)
}

func (p *GatePerp) GetMarket(symbol string) (*model.Market, error) {
	return p.market.GetMarket(symbol)
}

func (p *GatePerp) GetMarkets() ([]*model.Market, error) {
	return p.market.GetMarkets()
}

func (p *GatePerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

func (p *GatePerp) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	return p.market.FetchTickers(ctx, symbols...)
}

func (p *GatePerp) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	return p.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (p *GatePerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *GatePerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *GatePerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *GatePerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *GatePerp) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return p.order.FetchOrders(ctx, symbol, since, limit)
}

func (p *GatePerp) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return p.order.FetchOpenOrders(ctx, symbol)
}

func (p *GatePerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

func (p *GatePerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

func (p *GatePerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *GatePerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

func (p *GatePerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

func (p *GatePerp) IsHedgeMode() bool {
	return p.hedgeMode
}

var _ exchange.PerpExchange = (*GatePerp)(nil)

// ========== 内部实现 ==========

type gatePerpMarket struct {
	gate *Gate
}

func (m *gatePerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.gate.mu.RLock()
	if !reload && len(m.gate.perpMarkets) > 0 {
		m.gate.mu.RUnlock()
		return nil
	}
	m.gate.mu.RUnlock()

	// 获取永续合约市场信息
	// Gate 永续合约使用 USDT 作为结算货币
	settle := "usdt"
	resp, err := m.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/contracts", settle), nil)
	if err != nil {
		return fmt.Errorf("fetch swap markets: %w", err)
	}

	var data gatePerpMarketsResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return fmt.Errorf("unmarshal swap markets: %w", err)
	}

	markets := make([]*model.Market, 0)
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

		market := &model.Market{
			ID:            s.Name,
			Symbol:        normalizedSymbol,
			Base:          base,
			Quote:         quote,
			Settle:        strings.ToUpper(settle),
			Type:          model.MarketTypeSwap,
			Active:        !s.InDelisting,
			Contract:      true,
			ContractValue: s.QuantoMultiplier, // 合约面值（每张合约等于多少个币）
			Linear:        true,               // U本位永续合约
			Inverse:       false,
		}

		// 解析精度
		if !s.OrderPriceRound.IsZero() {
			orderPriceRound := s.OrderPriceRound.InexactFloat64()
			market.Precision.Price = getPrecisionDigits(orderPriceRound)
		}
		market.Precision.Amount = 0 // Gate 合约使用整数数量

		// 解析限制
		market.Limits.Amount.Min = types.ExDecimal{Decimal: decimal.NewFromInt(int64(s.OrderSizeMin))}
		market.Limits.Amount.Max = types.ExDecimal{Decimal: decimal.NewFromInt(int64(s.OrderSizeMax))}

		markets = append(markets, market)
	}

	// 存储市场信息
	m.gate.mu.Lock()
	if m.gate.perpMarkets == nil {
		m.gate.perpMarkets = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.gate.perpMarkets[market.Symbol] = market
	}
	m.gate.mu.Unlock()

	return nil
}

func (m *gatePerpMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.gate.perpMarkets))
	for _, market := range m.gate.perpMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *gatePerpMarket) GetMarket(symbol string) (*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	market, ok := m.gate.perpMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

func (m *gatePerpMarket) GetMarkets() ([]*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.gate.perpMarkets))
	for _, market := range m.gate.perpMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *gatePerpMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	settle := strings.ToLower(market.Settle)
	resp, err := m.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/tickers", settle), map[string]interface{}{
		"contract": gateSymbol,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var data gatePerpTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("ticker not found")
	}

	item := data[0]
	ticker := &model.Ticker{
		Symbol:    symbol,
		Timestamp: types.ExTimestamp{Time: time.Now()}, // Gate 永续合约 API 没有返回时间戳
	}

	ticker.Bid = item.HighestBid
	ticker.Ask = item.LowestAsk
	ticker.Last = item.Last
	ticker.High = item.High24h
	ticker.Low = item.Low24h
	ticker.Volume = item.Volume24hBase
	ticker.QuoteVolume = item.Volume24hQuote

	return ticker, nil
}

func (m *gatePerpMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	settle := "usdt" // Gate 永续合约默认使用 USDT 结算
	resp, err := m.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/tickers", settle), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data gatePerpTickerResponse

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
				gateSymbol, err := ToGateSymbol(s, true)
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
			normalizedSymbol, ok := gateSymbols[item.Contract]
			if !ok {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: types.ExTimestamp{Time: time.Now()}, // Gate 永续合约 API 没有返回时间戳
			}
			ticker.Bid = item.HighestBid
			ticker.Ask = item.LowestAsk
			ticker.Last = item.Last
			ticker.High = item.High24h
			ticker.Low = item.Low24h
			ticker.Volume = item.Volume24hBase
			ticker.QuoteVolume = item.Volume24hQuote
			tickers[normalizedSymbol] = ticker
		} else {
			// 如果没有指定 symbols，尝试从市场信息中查找
			market, err := m.getMarketByID(item.Contract)
			if err != nil {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    market.Symbol,
				Timestamp: types.ExTimestamp{Time: time.Now()}, // Gate 永续合约 API 没有返回时间戳
			}
			ticker.Bid = item.HighestBid
			ticker.Ask = item.LowestAsk
			ticker.Last = item.Last
			ticker.High = item.High24h
			ticker.Low = item.Low24h
			ticker.Volume = item.Volume24hBase
			ticker.QuoteVolume = item.Volume24hQuote
			tickers[market.Symbol] = ticker
		}
	}

	return tickers, nil
}

// getMarketByID 通过交易所ID获取市场信息
func (m *gatePerpMarket) getMarketByID(id string) (*model.Market, error) {
	m.gate.mu.RLock()
	defer m.gate.mu.RUnlock()

	for _, market := range m.gate.perpMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (m *gatePerpMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.GateTimeframe(timeframe)

	settle := strings.ToLower(market.Settle)
	params := map[string]interface{}{
		"contract": market.ID,
		"interval": normalizedTimeframe,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["from"] = since.Unix()
	}

	resp, err := m.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/candlesticks", settle), params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var data gatePerpKlineResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	ohlcvs := make(model.OHLCVs, 0, len(data))
	for _, item := range data {
		// 将 int64 Volume 转换为 types.ExDecimal
		volumeDecimal := types.ExDecimal{}
		if err := volumeDecimal.UnmarshalJSON([]byte(fmt.Sprintf(`"%d"`, item.Volume))); err != nil {
			return nil, fmt.Errorf("parse volume: %w", err)
		}
		ohlcv := &model.OHLCV{
			Timestamp: item.Time,
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			Volume:    volumeDecimal,
		}
		ohlcvs = append(ohlcvs, ohlcv)
	}

	return ohlcvs, nil
}

type gatePerpOrder struct {
	gate *Gate
}

// signAndRequest 签名并发送请求（Gate API）
func (o *gatePerpOrder) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
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

func (o *gatePerpOrder) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	// Gate 合约持仓
	resp, err := o.signAndRequest(ctx, "GET", "/api/v4/futures/usdt/positions", nil, nil)
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
		if item.Size == 0 {
			continue
		}

		market, err := o.getMarketByID(item.Contract)
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
			Amount:    float64(item.Size),
			Timestamp: time.Now(),
		}

		position.EntryPrice, _ = strconv.ParseFloat(item.EntryPrice, 64)
		position.MarkPrice, _ = strconv.ParseFloat(item.MarkPrice, 64)
		position.UnrealizedPnl, _ = strconv.ParseFloat(item.UnrealisedPnl, 64)

		if item.Size > 0 {
			position.Side = types.PositionSideLong
		} else {
			position.Side = types.PositionSideShort
			position.Amount = -position.Amount
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// getMarketByID 通过交易所ID获取市场信息
func (o *gatePerpOrder) getMarketByID(id string) (*model.Market, error) {
	o.gate.mu.RLock()
	defer o.gate.mu.RUnlock()

	for _, market := range o.gate.perpMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (o *gatePerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
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

	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, true)
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

	settle := strings.ToLower(market.Settle)
	path := fmt.Sprintf("/api/v4/futures/%s/orders", settle)
	reqBody := map[string]interface{}{
		"contract": gateSymbol,
	}

	// 合约下单必须指定 PositionSide
	if options.PositionSide == nil {
		return nil, fmt.Errorf("contract order requires PositionSide (long/short)")
	}

	// 计算 size（张数）: 张数 = 币的个数 / quanto_multiplier
	var size int64
	exContractValue, err := decimal.NewFromString(market.ContractValue)
	if err == nil && exContractValue.GreaterThan(decimal.Zero) {
		amountDecimal := decimal.NewFromFloat(amountFloat)
		contractSizeDecimal := amountDecimal.Div(exContractValue)
		contractSizeFloat, _ := contractSizeDecimal.Float64()
		size = int64(math.Ceil(contractSizeFloat))
		if size < 1 {
			size = 1
		}
	} else {
		size = int64(math.Ceil(amountFloat))
		if size < 1 {
			size = 1
		}
	}

	// 根据 side + PositionSide 确定 size 符号 和 reduce_only
	// 开多: PositionSideLong + SideBuy -> size正数, reduce_only=false
	// 平多: PositionSideLong + SideSell -> size负数, reduce_only=true
	// 开空: PositionSideShort + SideSell -> size负数, reduce_only=false
	// 平空: PositionSideShort + SideBuy -> size正数, reduce_only=true
	var reduceOnly bool
	if *options.PositionSide == types.PositionSideLong {
		if side == types.OrderSideBuy {
			// 开多
			reqBody["size"] = size
			reduceOnly = false
		} else {
			// 平多
			reqBody["size"] = -size
			reduceOnly = true
		}
	} else { // PositionSideShort
		if side == types.OrderSideSell {
			// 开空
			reqBody["size"] = -size
			reduceOnly = false
		} else {
			// 平空
			reqBody["size"] = size
			reduceOnly = true
		}
	}

	// 设置 reduce_only
	if reduceOnly {
		reqBody["reduce_only"] = true
	}

	// 价格设置
	if orderType == types.OrderTypeMarket {
		reqBody["price"] = "0"
	} else {
		reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', -1, 64)
	}

	// TimeInForce 设置
	if options.TimeInForce != nil {
		reqBody["tif"] = strings.ToLower(string(*options.TimeInForce))
	} else if orderType == types.OrderTypeMarket {
		reqBody["tif"] = "ioc" // 市价单固定 ioc
	} else {
		reqBody["tif"] = "gtc" // 限价单默认 gtc
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

// parseOrder 解析订单数据（合约版本）
func (o *gatePerpOrder) parseOrder(data map[string]interface{}, symbol string) *types.Order {
	order := &types.Order{
		ID:        getString(data, "id"),
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(getString(data, "price"), 64)
	order.Amount, _ = strconv.ParseFloat(getString(data, "size"), 64)
	if left := getString(data, "left"); left != "" {
		leftFloat, _ := strconv.ParseFloat(left, 64)
		order.Remaining = leftFloat
		order.Filled = order.Amount - leftFloat
	}

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
	case "closed":
		order.Status = types.OrderStatusFilled
	case "cancelled":
		order.Status = types.OrderStatusCanceled
	default:
		order.Status = types.OrderStatusNew
	}

	return order
}

func (o *gatePerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// 获取市场信息（用于获取 settle）
	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	settle := strings.ToLower(market.Settle)
	path := fmt.Sprintf("/api/v4/futures/%s/orders/%s", settle, orderID)

	_, err = o.signAndRequest(ctx, "DELETE", path, nil, nil)
	return err
}

func (o *gatePerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// 获取市场信息（用于获取 settle）
	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	settle := strings.ToLower(market.Settle)
	path := fmt.Sprintf("/api/v4/futures/%s/orders/%s", settle, orderID)

	resp, err := o.signAndRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	return o.parseOrder(data, symbol), nil
}

func (o *gatePerpOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Gate 合约 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: Gate perp API does not support fetching order history directly")
}

func (o *gatePerpOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// 获取市场信息
	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	settle := strings.ToLower(market.Settle)
	path := fmt.Sprintf("/api/v4/futures/%s/orders", settle)
	params := map[string]interface{}{
		"contract": gateSymbol,
		"status":   "open",
	}

	resp, err := o.signAndRequest(ctx, "GET", path, params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	orders := make([]*types.Order, 0, len(data))
	for _, item := range data {
		orders = append(orders, o.parseOrder(item, symbol))
	}

	return orders, nil
}

func (o *gatePerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	settle := strings.ToLower(market.Settle)
	params := map[string]interface{}{
		"contract": gateSymbol,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["from"] = since.Unix()
	}

	resp, err := o.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/trades", settle), params)
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

func (o *gatePerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	settle := strings.ToLower(market.Settle)
	path := fmt.Sprintf("/api/v4/futures/%s/my_trades", settle)
	params := map[string]interface{}{
		"contract": gateSymbol,
	}
	if limit > 0 {
		params["limit"] = limit
	}
	if !since.IsZero() {
		params["from"] = since.Unix()
	}

	resp, err := o.signAndRequest(ctx, "GET", path, params, nil)
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
		price, _ := strconv.ParseFloat(item.Price, 64)
		amount, _ := strconv.ParseFloat(item.Amount, 64)
		createTime, _ := strconv.ParseInt(item.Timestamp, 10, 64)

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

func (o *gatePerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := o.gate.perp.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("leverage only supported for contracts")
	}

	settle := strings.ToLower(market.Settle)
	gateSymbol := market.ID
	if gateSymbol == "" {
		var err error
		gateSymbol, err = ToGateSymbol(symbol, true)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"contract": gateSymbol,
		"leverage": strconv.Itoa(leverage),
	}

	path := fmt.Sprintf("/api/v4/futures/%s/positions/%s/leverage", settle, gateSymbol)
	_, err = o.signAndRequest(ctx, "POST", path, nil, reqBody)
	return err
}

func (o *gatePerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	// Gate 不支持通过 API 设置保证金模式，需要在网页端设置
	return fmt.Errorf("not supported: Gate does not support setting margin mode via API")
}

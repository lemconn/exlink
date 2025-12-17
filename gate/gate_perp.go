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
	"github.com/lemconn/exlink/option"
	"github.com/lemconn/exlink/types"
	"github.com/shopspring/decimal"
)

// GatePerp Gate 永续合约实现
type GatePerp struct {
	gate *Gate
}

// NewGatePerp 创建 Gate 永续合约实例
func NewGatePerp(g *Gate) *GatePerp {
	return &GatePerp{
		gate: g,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *GatePerp) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	p.gate.mu.RLock()
	if !reload && len(p.gate.perpMarketsBySymbol) > 0 {
		p.gate.mu.RUnlock()
		return nil
	}
	p.gate.mu.RUnlock()

	// 获取永续合约市场信息
	// Gate 永续合约使用 USDT 作为结算货币
	settle := "usdt"
	resp, err := p.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/contracts", settle), nil)
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
	p.gate.mu.Lock()
	if p.gate.perpMarketsBySymbol == nil {
		p.gate.perpMarketsBySymbol = make(map[string]*model.Market)
		p.gate.perpMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		p.gate.perpMarketsBySymbol[market.Symbol] = market
		p.gate.perpMarketsByID[market.ID] = market
	}
	p.gate.mu.Unlock()

	return nil
}

func (p *GatePerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	p.gate.mu.RLock()
	defer p.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.gate.perpMarketsBySymbol))
	for _, market := range p.gate.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *GatePerp) GetMarket(symbol string) (*model.Market, error) {
	p.gate.mu.RLock()
	defer p.gate.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := p.gate.perpMarketsBySymbol[symbol]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := p.gate.perpMarketsByID[symbol]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", symbol)
}

func (p *GatePerp) GetMarkets() ([]*model.Market, error) {
	p.gate.mu.RLock()
	defer p.gate.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.gate.perpMarketsBySymbol))
	for _, market := range p.gate.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *GatePerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
	resp, err := p.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/tickers", settle), map[string]interface{}{
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

func (p *GatePerp) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	settle := "usdt" // Gate 永续合约默认使用 USDT 结算
	resp, err := p.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/tickers", settle), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data gatePerpTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range data {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.Contract)
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

	return tickers, nil
}

func (p *GatePerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
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

	resp, err := p.gate.client.HTTPClient.Get(ctx, fmt.Sprintf("/api/v4/futures/%s/candlesticks", settle), params)
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

func (p *GatePerp) FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error) {
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

func (p *GatePerp) CreateOrder(ctx context.Context, symbol string, side option.PerpOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
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

	// 构建请求结构体
	req := gatePerpCreateOrderRequest{
		Contract: gateSymbol,
	}

	// 从 PerpOrderSide 自动推断 PositionSide 和 reduceOnly
	reduceOnly := side.ToReduceOnly()

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

	// 根据 PerpOrderSide 确定 size 符号
	// 开多: OpenLong -> size正数
	// 平多: CloseLong -> size负数
	// 开空: OpenShort -> size负数
	// 平空: CloseShort -> size正数
	switch side {
	case option.OpenLong:
		req.Size = size
	case option.CloseLong:
		req.Size = -size
	case option.OpenShort:
		req.Size = -size
	case option.CloseShort:
		req.Size = size
	default:
		return nil, fmt.Errorf("invalid PerpOrderSide: %s", side)
	}

	// 设置 reduce_only
	if reduceOnly {
		req.ReduceOnly = true
	}

	// 价格设置
	if orderType == option.Market {
		req.Price = "0"
	} else {
		req.Price = strconv.FormatFloat(priceFloat, 'f', -1, 64)
	}

	// TimeInForce 设置
	if argsOpts.TimeInForce != nil {
		req.Tif = argsOpts.TimeInForce.Lower()
	} else if orderType == option.Market {
		req.Tif = option.IOC.Lower() // 市价单固定 ioc
	} else {
		req.Tif = option.GTC.Lower() // 限价单默认 gtc
	}

	// 客户端订单ID
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		req.Text = *argsOpts.ClientOrderID
	} else {
		// 将 PerpOrderSide 转换为 OrderSide 用于生成订单ID
		orderSide := types.OrderSide(strings.ToLower(side.ToSide()))
		req.Text = common.GenerateClientOrderID(p.gate.Name(), orderSide)
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

	resp, err := p.signAndRequest(ctx, "POST", path, nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var respData gatePerpCreateOrderResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	// 构建 NewOrder 对象
	// Gate 的 text 字段可能包含 "t-" 前缀，需要去除
	clientOrderID := strings.TrimPrefix(respData.Text, "t-")

	perpOrder := &model.NewOrder{
		Symbol:        symbol,
		OrderId:       respData.ID,
		ClientOrderID: clientOrderID,
		Timestamp:     respData.UpdateTime,
	}

	return perpOrder, nil
}

func (p *GatePerp) CancelOrder(ctx context.Context, symbol string, opts ...option.ArgsOption) error {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 判断 OrderId 或 ClientOrderId 必须存在一个
	if (argsOpts.OrderId == nil || *argsOpts.OrderId == "") && (argsOpts.ClientOrderID == nil || *argsOpts.ClientOrderID == "") {
		return fmt.Errorf("either OrderId or ClientOrderID must be provided")
	}

	// 获取市场信息（用于获取 settle）
	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}

	settle := strings.ToLower(market.Settle)

	// Gate API 支持通过 order_id 或 text 参数（clientOrderId）
	var path string
	var params map[string]interface{}
	if argsOpts.OrderId != nil && *argsOpts.OrderId != "" {
		path = fmt.Sprintf("/api/v4/futures/%s/orders/%s", settle, *argsOpts.OrderId)
		params = nil
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		// 使用 text 参数通过 clientOrderId 取消订单
		path = fmt.Sprintf("/api/v4/futures/%s/orders", settle)
		params = map[string]interface{}{
			"text": *argsOpts.ClientOrderID,
		}
	}

	_, err = p.signAndRequest(ctx, "DELETE", path, params, nil)
	return err
}

func (p *GatePerp) FetchOrder(ctx context.Context, symbol string, opts ...option.ArgsOption) (*model.PerpOrder, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 判断 OrderId 或 ClientOrderId 必须存在一个
	if (argsOpts.OrderId == nil || *argsOpts.OrderId == "") && (argsOpts.ClientOrderID == nil || *argsOpts.ClientOrderID == "") {
		return nil, fmt.Errorf("either OrderId or ClientOrderID must be provided")
	}

	// 获取市场信息（用于获取 settle）
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	settle := strings.ToLower(market.Settle)

	// Gate API 支持通过 order_id 或 text 参数（clientOrderId）
	var path string
	var params map[string]interface{}
	if argsOpts.OrderId != nil && *argsOpts.OrderId != "" {
		path = fmt.Sprintf("/api/v4/futures/%s/orders/%s", settle, *argsOpts.OrderId)
		params = nil
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		// 使用 text 参数通过 clientOrderId 查询订单
		path = fmt.Sprintf("/api/v4/futures/%s/orders", settle)
		params = map[string]interface{}{
			"text": *argsOpts.ClientOrderID,
		}
	}

	resp, err := p.signAndRequest(ctx, "GET", path, params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var data gatePerpFetchOrderResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	return p.parsePerpOrder(data, symbol), nil
}

func (p *GatePerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := p.GetMarket(symbol)
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
	_, err = p.signAndRequest(ctx, "POST", path, nil, reqBody)
	return err
}

func (p *GatePerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	// Gate 不支持通过 API 设置保证金模式，需要在网页端设置
	return fmt.Errorf("not supported: Gate does not support setting margin mode via API")
}

var _ exchange.PerpExchange = (*GatePerp)(nil)

// ========== 内部辅助方法 ==========

// signAndRequest 签名并发送请求（Gate API）
func (p *GatePerp) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if p.gate.client.SecretKey == "" {
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
	signature := p.gate.signer.SignRequest(method, path, queryString, bodyStr, timestamp)

	// 设置请求头
	p.gate.client.HTTPClient.SetHeader("KEY", p.gate.client.APIKey)
	p.gate.client.HTTPClient.SetHeader("Timestamp", strconv.FormatInt(timestamp, 10))
	p.gate.client.HTTPClient.SetHeader("SIGN", signature)
	p.gate.client.HTTPClient.SetHeader("Content-Type", "application/json")
	p.gate.client.HTTPClient.SetHeader("X-Gate-Channel-Id", "api")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return p.gate.client.HTTPClient.Get(ctx, path, params)
	} else {
		return p.gate.client.HTTPClient.Post(ctx, path, body)
	}
}

func (p *GatePerp) fetchPositions(ctx context.Context, symbols ...string) (model.Positions, error) {
	// Gate 合约持仓
	resp, err := p.signAndRequest(ctx, "GET", "/api/v4/futures/usdt/positions", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var data gatePerpPositionResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	positions := make([]*model.Position, 0)
	for _, item := range data {
		size, _ := item.Size.Float64()
		if size == 0 {
			continue
		}

		market, err := p.GetMarket(item.Contract)
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
		amount := size
		if size > 0 {
			side = string(types.PositionSideLong)
		} else {
			side = string(types.PositionSideShort)
			amount = -amount
		}

		position := &model.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           types.ExDecimal{Decimal: decimal.NewFromFloat(amount)},
			EntryPrice:       item.EntryPrice,
			MarkPrice:        item.MarkPrice,
			UnrealizedPnl:    item.UnrealisedPnl,
			LiquidationPrice: item.LiqPrice,
			RealizedPnl:      item.RealisedPnl,
			Leverage:         item.Leverage,
			Margin:           item.Margin,
			Percentage:       types.ExDecimal{},
			Timestamp:        item.UpdateTime,
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// parsePerpOrder 将 Gate 响应转换为 model.PerpOrder
func (p *GatePerp) parsePerpOrder(data gatePerpFetchOrderResponse, symbol string) *model.PerpOrder {
	// 计算实际成交数量（size - left）
	//nolint:staticcheck // QF1008: need to access Decimal field for Sub method
	executedQtyDecimal := data.Size.Decimal.Sub(data.Left.Decimal)
	var executedQty types.ExDecimal
	if executedQtyDecimal.IsNegative() {
		executedQty = types.ExDecimal{Decimal: decimal.Zero}
	} else {
		executedQty = types.ExDecimal{Decimal: executedQtyDecimal}
	}

	// 确定订单方向和类型（Gate 的 size 正负表示方向）
	var side string
	if data.Size.IsPositive() {
		side = "buy"
	} else {
		side = "sell"
	}

	// 确定订单类型（Gate 的 tif 为 ioc 时通常是市价单）
	var orderType string
	if strings.ToLower(data.Tif) == "ioc" && data.Price.IsZero() {
		orderType = "market"
	} else {
		orderType = "limit"
	}

	// 确定 positionSide（Gate 没有明确的 positionSide，使用 BOTH）
	positionSide := "BOTH"

	order := &model.PerpOrder{
		ID:           strconv.FormatInt(data.ID, 10),
		ClientID:     data.Text,
		Type:         orderType,
		Side:         side,
		PositionSide: positionSide,
		Symbol:       symbol,
		Price:        data.Price,
		AvgPrice:     data.FillPrice,
		//nolint:staticcheck // QF1008: need to access Decimal field for Abs method
		Quantity:         types.ExDecimal{Decimal: data.Size.Decimal.Abs()}, // 使用绝对值作为数量
		ExecutedQuantity: executedQty,
		Status:           data.Status,
		TimeInForce:      strings.ToUpper(data.Tif),
		ReduceOnly:       data.IsReduceOnly,
		CreateTime:       data.CreateTime,
		UpdateTime:       data.UpdateTime,
	}

	return order
}

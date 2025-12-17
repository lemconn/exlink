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

// BinancePerp Binance 永续合约实现
type BinancePerp struct {
	binance *Binance
}

// NewBinancePerp 创建 Binance 永续合约实例
func NewBinancePerp(b *Binance) *BinancePerp {
	return &BinancePerp{
		binance: b,
	}
}

// ========== PerpExchange 接口实现 ==========

// LoadMarkets 加载市场信息
func (p *BinancePerp) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	p.binance.mu.RLock()
	if !reload && len(p.binance.perpMarketsBySymbol) > 0 {
		p.binance.mu.RUnlock()
		return nil
	}
	p.binance.mu.RUnlock()

	// 获取永续合约市场信息
	resp, err := p.binance.client.PerpClient.Get(ctx, "/fapi/v1/exchangeInfo", map[string]interface{}{
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
	p.binance.mu.Lock()
	if p.binance.perpMarketsBySymbol == nil {
		p.binance.perpMarketsBySymbol = make(map[string]*model.Market)
		p.binance.perpMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		p.binance.perpMarketsBySymbol[market.Symbol] = market
		p.binance.perpMarketsByID[market.ID] = market
	}
	p.binance.mu.Unlock()

	return nil
}

// FetchMarkets 获取市场列表
func (p *BinancePerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	p.binance.mu.RLock()
	defer p.binance.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.binance.perpMarketsBySymbol))
	for _, market := range p.binance.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

// GetMarket 获取单个市场信息
func (p *BinancePerp) GetMarket(symbol string) (*model.Market, error) {
	p.binance.mu.RLock()
	defer p.binance.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := p.binance.perpMarketsBySymbol[symbol]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := p.binance.perpMarketsByID[symbol]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", symbol)
}

// GetMarkets 从内存中获取所有市场信息
func (p *BinancePerp) GetMarkets() ([]*model.Market, error) {
	p.binance.mu.RLock()
	defer p.binance.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.binance.perpMarketsBySymbol))
	for _, market := range p.binance.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

// FetchTicker 获取行情（单个）
func (p *BinancePerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
	resp, err := p.binance.client.PerpClient.Get(ctx, "/fapi/v1/ticker/24hr", map[string]interface{}{
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
func (p *BinancePerp) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := p.binance.client.PerpClient.Get(ctx, "/fapi/v1/ticker/24hr", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var data []binancePerpTickerResponse

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range data {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.Symbol)
		if err != nil {
			// 如果找不到市场信息，使用 Binance 原始格式
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
		} else {
			ticker := &model.Ticker{
				Symbol:    market.Symbol,
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
			tickers[market.Symbol] = ticker
		}
	}

	return tickers, nil
}

// FetchOHLCVs 获取K线数据
func (p *BinancePerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
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

	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
	resp, err := p.binance.client.PerpClient.Get(ctx, "/fapi/v1/klines", params)
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

// FetchPositions 获取持仓
func (p *BinancePerp) FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取参数值（带默认值）
	symbols := []string{} // 默认值
	if argsOpts.Symbols != nil {
		symbols = argsOpts.Symbols
	}

	if p.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	timestamp := common.GetTimestamp()
	params := map[string]interface{}{
		"timestamp": timestamp,
	}

	queryString := BuildQueryString(params)
	signature := p.binance.signer.Sign(queryString)
	params["signature"] = signature

	resp, err := p.binance.client.PerpClient.Get(ctx, "/fapi/v2/positionRisk", params)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var data binancePerpPositionResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	positions := make([]*model.Position, 0)
	for _, item := range data {
		positionAmt, _ := item.PositionAmt.Float64()
		if positionAmt == 0 {
			continue // 跳过空仓
		}

		// 获取市场信息（通过 ID 查找）
		market, err := p.GetMarket(item.Symbol)
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

		var side string
		amount := item.PositionAmt
		if positionAmt > 0 {
			side = string(types.PositionSideLong)
		} else {
			side = string(types.PositionSideShort)
			amount = types.ExDecimal{Decimal: amount.Neg()}
		}

		position := &model.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           amount,
			EntryPrice:       item.EntryPrice,
			MarkPrice:        item.MarkPrice,
			LiquidationPrice: item.LiquidationPrice,
			UnrealizedPnl:    item.UnRealizedProfit,
			RealizedPnl:      types.ExDecimal{},
			Leverage:         item.Leverage,
			Margin:           item.IsolatedMargin,
			Percentage:       types.ExDecimal{},
			Timestamp:        item.UpdateTime,
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// CreateOrder 创建订单
func (p *BinancePerp) CreateOrder(ctx context.Context, symbol string, side option.PerpOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	if p.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 解析订单选项
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
	binanceSymbol := market.ID
	if binanceSymbol == "" {
		var err error
		binanceSymbol, err = ToBinanceSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
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

	// 构建请求结构体
	reqTimestamp := common.GetTimestamp()
	req := binancePerpCreateOrderRequest{
		Symbol: binanceSymbol,
		Side:   side.ToSide(),
		Type:   orderType.Upper(),
	}

	// 格式化数量（合约订单使用更保守的精度策略）
	// 合约订单：如果数量是整数，使用 0 精度；否则使用 1 位小数精度
	var amountPrecision int
	if amountDecimal.IsInteger() && amountDecimal.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		amountPrecision = 0
	} else {
		amountPrecision = 1
	}
	req.Quantity = amountDecimal.StringFixed(int32(amountPrecision))

	// 处理限价单的价格和 timeInForce
	if orderType == option.Limit {
		pricePrecision := market.Precision.Price
		if pricePrecision == 0 {
			pricePrecision = 8 // 默认精度
		}
		req.Price = priceDecimal.StringFixed(int32(pricePrecision))

		// 处理 timeInForce：如果设置了则使用，否则使用默认值 GTC
		if argsOpts.TimeInForce != nil {
			req.TimeInForce = argsOpts.TimeInForce.Upper()
		} else {
			req.TimeInForce = option.GTC.Upper()
		}
	}

	// ========== 永续合约订单处理 ==========
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
		// 开多/平多: positionSide=LONG
		// 开空/平空: positionSide=SHORT
		req.PositionSide = positionSideStr
	} else {
		// 单向持仓模式
		req.PositionSide = "BOTH"

		// 如果是平仓操作，设置 reduceOnly
		if reduceOnly {
			req.ReduceOnly = "true"
		}
	}

	// 生成客户端订单ID（如果未提供）
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		req.NewClientOrderID = *argsOpts.ClientOrderID
	} else {
		// 将 PerpOrderSide 转换为 OrderSide 用于生成订单ID
		orderSide := types.OrderSide(strings.ToLower(side.ToSide()))
		req.NewClientOrderID = common.GenerateClientOrderID(p.binance.Name(), orderSide)
	}

	// 将结构体转换为 map，以便添加 timestamp 和 signature
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var reqParams map[string]interface{}
	if err := json.Unmarshal(reqBytes, &reqParams); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	// 添加 timestamp 和 signature
	reqParams["timestamp"] = reqTimestamp
	queryString := BuildQueryString(reqParams)
	signature := p.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	// 发送请求（合约订单使用 PerpClient）
	resp, err := p.binance.client.PerpClient.Request(ctx, "POST", "/fapi/v1/order", reqParams, nil)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 解析响应（合约订单响应）
	var respData binancePerpCreateOrderResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal contract order response: %w", err)
	}

	// 构建 NewOrder 对象
	perpOrder := &model.NewOrder{
		Symbol:        symbol,
		OrderId:       respData.OrderID,
		ClientOrderID: respData.ClientOrderID,
		Timestamp:     respData.UpdateTime,
	}

	return perpOrder, nil
}

// CancelOrder 取消订单
func (p *BinancePerp) CancelOrder(ctx context.Context, symbol string, opts ...option.ArgsOption) error {
	if p.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 判断 OrderId 或 ClientOrderId 必须存在一个
	if (argsOpts.OrderId == nil || *argsOpts.OrderId == "") && (argsOpts.ClientOrderID == nil || *argsOpts.ClientOrderID == "") {
		return fmt.Errorf("either OrderId or ClientOrderID must be provided")
	}

	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
		"timestamp": timestamp,
	}

	// 优先使用 OrderId，如果没有则使用 ClientOrderID
	if argsOpts.OrderId != nil && *argsOpts.OrderId != "" {
		params["orderId"] = *argsOpts.OrderId
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["origClientOrderId"] = *argsOpts.ClientOrderID
	}

	queryString := BuildQueryString(params)
	signature := p.binance.signer.Sign(queryString)
	params["signature"] = signature

	_, err = p.binance.client.PerpClient.Post(ctx, "/fapi/v1/order", params)
	return err
}

// FetchOrder 查询订单
func (p *BinancePerp) FetchOrder(ctx context.Context, symbol string, opts ...option.ArgsOption) (*model.PerpOrder, error) {
	if p.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 判断 OrderId 或 ClientOrderId 必须存在一个
	if (argsOpts.OrderId == nil || *argsOpts.OrderId == "") && (argsOpts.ClientOrderID == nil || *argsOpts.ClientOrderID == "") {
		return nil, fmt.Errorf("either OrderId or ClientOrderID must be provided")
	}

	// 获取市场信息
	market, err := p.GetMarket(symbol)
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
		"timestamp": timestamp,
	}

	// 优先使用 OrderId，如果没有则使用 ClientOrderID
	if argsOpts.OrderId != nil && *argsOpts.OrderId != "" {
		params["orderId"] = *argsOpts.OrderId
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["origClientOrderId"] = *argsOpts.ClientOrderID
	}

	queryString := BuildQueryString(params)
	signature := p.binance.signer.Sign(queryString)
	params["signature"] = signature

	// 使用合约 API
	resp, err := p.binance.client.PerpClient.Get(ctx, "/fapi/v1/order", params)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	// 解析响应
	var data binancePerpFetchOrderResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	// 检查是否有错误码（通过检查 orderId 是否为 0）
	if data.OrderID == 0 {
		return nil, fmt.Errorf("order not found")
	}

	return p.parsePerpOrder(data, symbol), nil
}

// SetLeverage 设置杠杆
func (p *BinancePerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if p.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	market, err := p.GetMarket(symbol)
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
	signature := p.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	_, err = p.binance.client.PerpClient.Post(ctx, "/fapi/v1/leverage", reqParams)
	return err
}

// SetMarginMode 设置保证金模式
func (p *BinancePerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	if p.binance.client.SecretKey == "" {
		return fmt.Errorf("authentication required")
	}

	market, err := p.GetMarket(symbol)
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
	signature := p.binance.signer.Sign(queryString)
	reqParams["signature"] = signature

	_, err = p.binance.client.PerpClient.Post(ctx, "/fapi/v1/marginType", reqParams)
	return err
}

// ========== 内部辅助方法 ==========

// parsePerpOrder 将 Binance 响应转换为 model.PerpOrder
func (p *BinancePerp) parsePerpOrder(data binancePerpFetchOrderResponse, symbol string) *model.PerpOrder {
	order := &model.PerpOrder{
		ID:               strconv.FormatInt(data.OrderID, 10),
		ClientID:         data.ClientOrderID,
		Type:             data.Type,
		Side:             data.Side,
		PositionSide:     data.PositionSide,
		Symbol:           symbol, // 使用标准化格式
		Price:            data.Price,
		AvgPrice:         data.AvgPrice,
		Quantity:         data.OrigQty,
		ExecutedQuantity: data.ExecutedQty,
		Status:           data.Status,
		TimeInForce:      data.TimeInForce,
		ReduceOnly:       data.ReduceOnly,
		CreateTime:       data.Time,
		UpdateTime:       data.UpdateTime,
	}

	return order
}

// 确保 BinancePerp 实现了 exchange.PerpExchange 接口
var _ exchange.PerpExchange = (*BinancePerp)(nil)

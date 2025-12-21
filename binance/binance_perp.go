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

// signAndRequest 统一处理签名和发送请求
// method: HTTP 方法，支持 "GET", "POST", "DELETE"
// path: API 路径，例如 "/fapi/v1/order"
// req: 已设置好参数的 ExValues 对象（不包含 timestamp 和 signature）
func (p *BinancePerp) signAndRequest(ctx context.Context, method, path string, req *types.ExValues) ([]byte, error) {
	// 检查认证
	if p.binance.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// 添加 timestamp
	req.SetQuery("timestamp", common.GetTimestamp())

	// 生成签名
	queryString := req.EncodeQuery()
	signature := p.binance.signer.Sign(queryString)
	req.SetQuery("signature", signature)

	// 构建完整路径
	reqPath := req.JoinPath(path)

	// 根据方法发送请求
	switch method {
	case "GET":
		return p.binance.client.PerpClient.Get(ctx, reqPath, nil)
	case "POST":
		return p.binance.client.PerpClient.Post(ctx, reqPath, nil)
	case "DELETE":
		return p.binance.client.PerpClient.Delete(ctx, reqPath, map[string]interface{}{}, nil)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
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

	req := types.NewExValues()
	reqPath := req.JoinPath("/fapi/v1/exchangeInfo")
	// 获取永续合约市场信息
	resp, err := p.binance.client.PerpClient.Get(ctx, reqPath, nil)
	if err != nil {
		return fmt.Errorf("fetch fapi exchange info: %w", err)
	}

	var respData struct {
		Symbols []struct {
			Symbol            string `json:"symbol"`
			Pair              string `json:"pair"`
			ContractType      string `json:"contractType"`
			BaseAsset         string `json:"baseAsset"`
			QuoteAsset        string `json:"quoteAsset"`
			MarginAsset       string `json:"marginAsset"`
			Status            string `json:"status"`
			PricePrecision    int    `json:"pricePrecision"`
			QuantityPrecision int    `json:"quantityPrecision"`
			Filters           []struct {
				FilterType  string          `json:"filterType"`
				MinQty      types.ExDecimal `json:"minQty,omitempty"`
				MaxQty      types.ExDecimal `json:"maxQty,omitempty"`
				StepSize    types.ExDecimal `json:"stepSize,omitempty"`
				MinPrice    types.ExDecimal `json:"minPrice,omitempty"`
				MaxPrice    types.ExDecimal `json:"maxPrice,omitempty"`
				TickSize    types.ExDecimal `json:"tickSize,omitempty"`
				MinNotional types.ExDecimal `json:"minNotional,omitempty"`
			} `json:"filters"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(resp, &respData); err != nil {
		return fmt.Errorf("unmarshal fapi exchange info: %w", err)
	}

	markets := make(model.Markets, 0)
	for _, s := range respData.Symbols {
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
func (p *BinancePerp) FetchMarkets(ctx context.Context, opts ...option.ArgsOption) (model.Markets, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	p.binance.mu.RLock()
	defer p.binance.mu.RUnlock()

	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		return model.Markets{market}, nil
	}

	markets := make(model.Markets, 0, len(p.binance.perpMarketsBySymbol))
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
func (p *BinancePerp) FetchTickers(ctx context.Context, opts ...option.ArgsOption) (model.Tickers, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	req := types.NewExValues()

	var querySymbol string
	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		querySymbol = market.ID
	}

	reqPath := req.JoinPath("/fapi/v1/ticker/24hr")
	resp, err := p.binance.client.PerpClient.Get(ctx, reqPath, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var respData []struct {
		Symbol             string            `json:"symbol"`
		PriceChange        types.ExDecimal   `json:"priceChange"`
		PriceChangePercent types.ExDecimal   `json:"priceChangePercent"`
		WeightedAvgPrice   types.ExDecimal   `json:"weightedAvgPrice"`
		LastPrice          types.ExDecimal   `json:"lastPrice"`
		LastQty            types.ExDecimal   `json:"lastQty"`
		OpenPrice          types.ExDecimal   `json:"openPrice"`
		HighPrice          types.ExDecimal   `json:"highPrice"`
		LowPrice           types.ExDecimal   `json:"lowPrice"`
		Volume             types.ExDecimal   `json:"volume"`
		QuoteVolume        types.ExDecimal   `json:"quoteVolume"`
		OpenTime           types.ExTimestamp `json:"openTime"`
		CloseTime          types.ExTimestamp `json:"closeTime"`
		Count              int64             `json:"count"`
	}

	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	tickers := make(model.Tickers, 0, len(respData))
	for _, item := range respData {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.Symbol)
		if err != nil {
			continue
		}
		if querySymbol != "" && market.ID != querySymbol {
			continue
		}
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
		tickers = append(tickers, ticker)
	}

	return tickers, nil
}

// FetchOHLCVs 获取K线数据
func (p *BinancePerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, limit int, opts ...option.ArgsOption) (model.OHLCVs, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	req := types.NewExValues()

	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}
	req.SetQuery("symbol", market.ID)
	req.SetQuery("interval", common.BinanceTimeframe(timeframe))
	req.SetQuery("limit", limit)

	if since, ok := option.GetTime(argsOpts.Since); ok {
		req.SetQuery("startTime", since.UnixMilli())
	}

	reqPath := req.JoinPath("/fapi/v1/klines")
	// 使用合约 API
	resp, err := p.binance.client.PerpClient.Get(ctx, reqPath, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var respData [][]interface{}
	if err = json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	ohlcvs := make(model.OHLCVs, 0, len(respData))
	for _, item := range respData {
		ohlcv := &model.OHLCV{}
		ohlcv.Timestamp = types.ExTimestamp{Time: time.UnixMilli(int64(item[0].(float64)))}
		if openPx, err := decimal.NewFromString(item[1].(string)); err == nil {
			ohlcv.Open = types.ExDecimal{Decimal: openPx}
		}
		if highPx, err := decimal.NewFromString(item[2].(string)); err == nil {
			ohlcv.High = types.ExDecimal{Decimal: highPx}
		}
		if lowPx, err := decimal.NewFromString(item[3].(string)); err == nil {
			ohlcv.Low = types.ExDecimal{Decimal: lowPx}
		}
		if closePx, err := decimal.NewFromString(item[4].(string)); err == nil {
			ohlcv.Close = types.ExDecimal{Decimal: closePx}
		}
		if volume, err := decimal.NewFromString(item[5].(string)); err == nil {
			ohlcv.Volume = types.ExDecimal{Decimal: volume}
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

	req := types.NewExValues()

	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		req.SetQuery("symbol", market.ID)
	}

	resp, err := p.signAndRequest(ctx, "GET", "/fapi/v2/positionRisk", req)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var respData []struct {
		Symbol           string            `json:"symbol"`
		PositionAmt      types.ExDecimal   `json:"positionAmt"`
		EntryPrice       types.ExDecimal   `json:"entryPrice"`
		BreakEvenPrice   types.ExDecimal   `json:"breakEvenPrice"`
		MarkPrice        types.ExDecimal   `json:"markPrice"`
		UnRealizedProfit types.ExDecimal   `json:"unRealizedProfit"`
		LiquidationPrice types.ExDecimal   `json:"liquidationPrice"`
		Leverage         types.ExDecimal   `json:"leverage"`
		MaxNotionalValue types.ExDecimal   `json:"maxNotionalValue"`
		MarginType       string            `json:"marginType"`
		IsolatedMargin   types.ExDecimal   `json:"isolatedMargin"`
		IsAutoAddMargin  string            `json:"isAutoAddMargin"`
		PositionSide     string            `json:"positionSide"`
		Notional         types.ExDecimal   `json:"notional"`
		IsolatedWallet   types.ExDecimal   `json:"isolatedWallet"`
		UpdateTime       types.ExTimestamp `json:"updateTime"`
		Isolated         bool              `json:"isolated"`
		AdlQuantile      int               `json:"adlQuantile"`
	}

	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	positions := make([]*model.Position, 0)
	for _, item := range respData {
		positionAmt, _ := item.PositionAmt.Float64()
		if positionAmt == 0 {
			continue // 跳过空仓
		}

		// 获取市场信息（通过 ID 查找）
		market, err := p.GetMarket(item.Symbol)
		if err != nil {
			continue
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
func (p *BinancePerp) CreateOrder(ctx context.Context, symbol string, amount string, orderSide option.PerpOrderSide, orderType option.OrderType, opts ...option.ArgsOption) (*model.NewOrder, error) {
	// 解析订单选项
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	req := types.NewExValues()

	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}
	req.SetQuery("symbol", market.ID)

	// 设置限价单价格
	if orderType == option.Limit {
		price, ok := option.GetDecimalFromString(argsOpts.Price)
		if !ok || price.IsZero() {
			return nil, fmt.Errorf("limit order requires price")
		}
		req.SetQuery("price", price.String())
		// Limit 单默认使用 GTC
		req.SetQuery("timeInForce", option.GTC.Upper())
	}

	if argsOpts.TimeInForce != nil {
		req.SetQuery("timeInForce", argsOpts.TimeInForce.Upper())
	}

	// 设置数量
	if quantity, ok := option.GetDecimalFromString(&amount); ok {
		req.SetQuery("quantity", quantity.String())
	} else {
		return nil, fmt.Errorf("amount is required and must be a valid decimal")
	}

	// 设置订单方向和类型
	req.SetQuery("side", orderSide.ToSide())
	if orderSide.ToReduceOnly() {
		req.SetQuery("reduceOnly", "true")
	} else {
		req.SetQuery("reduceOnly", "false")
	}
	req.SetQuery("type", orderType.Upper())

	if hedgeMode, ok := option.GetBool(argsOpts.HedgeMode); hedgeMode && ok {
		// 双向持仓模式
		// 开多/平多: positionSide=LONG
		// 开空/平空: positionSide=SHORT
		req.SetQuery("positionSide", orderSide.ToPositionSide())
	} else {
		req.SetQuery("positionSide", "BOTH")
	}

	if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetQuery("newClientOrderId", clientOrderId)
	} else {
		// 生成订单 ID
		generatedID := common.GenerateClientOrderID(p.binance.Name(), orderSide.ToSide())
		req.SetQuery("newClientOrderId", generatedID)
	}

	resp, err := p.signAndRequest(ctx, "POST", "/fapi/v1/order", req)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 解析响应（合约订单响应）
	var respData struct {
		OrderID       int64             `json:"orderId"`       // 系统订单号
		ClientOrderID string            `json:"clientOrderId"` // 客户端订单ID
		UpdateTime    types.ExTimestamp `json:"updateTime"`    // 更新时间（毫秒时间戳）
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal contract order response: %w", err)
	}

	// 构建 NewOrder 对象
	perpOrder := &model.NewOrder{
		Symbol:        symbol,
		OrderId:       strconv.FormatInt(respData.OrderID, 10),
		ClientOrderID: respData.ClientOrderID,
		Timestamp:     respData.UpdateTime,
	}

	return perpOrder, nil
}

// CancelOrder 取消订单
func (p *BinancePerp) CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	req := types.NewExValues()

	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}
	req.SetQuery("symbol", market.ID)

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		req.SetQuery("orderId", orderId)
	} else if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetQuery("origClientOrderId", clientOrderId)
	} else {
		return fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	_, err = p.signAndRequest(ctx, "DELETE", "/fapi/v1/order", req)
	return err
}

// FetchOrder 查询订单
func (p *BinancePerp) FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.PerpOrder, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	req := types.NewExValues()

	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}
	req.SetQuery("symbol", market.ID)

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		req.SetQuery("orderId", orderId)
	} else if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetQuery("origClientOrderId", clientOrderId)
	} else {
		return nil, fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	resp, err := p.signAndRequest(ctx, "GET", "/fapi/v1/order", req)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	// 解析响应
	var respData struct {
		OrderID       int64             `json:"orderId"`       // 订单ID（交易所唯一）
		ClientOrderID string            `json:"clientOrderId"` // 客户端自定义订单ID
		Symbol        string            `json:"symbol"`        // 交易对 / 合约标的
		Price         types.ExDecimal   `json:"price"`         // 下单价格（市价单通常为0）
		AvgPrice      types.ExDecimal   `json:"avgPrice"`      // 成交均价
		OrigQty       types.ExDecimal   `json:"origQty"`       // 下单数量
		ExecutedQty   types.ExDecimal   `json:"executedQty"`   // 实际成交数量
		Status        string            `json:"status"`        // 订单状态
		TimeInForce   string            `json:"timeInForce"`   // 订单有效方式
		ReduceOnly    bool              `json:"reduceOnly"`    // 是否只减仓
		Time          types.ExTimestamp `json:"time"`          // 创建时间（毫秒）
		Type          string            `json:"type"`          // 订单类型
		Side          string            `json:"side"`          // 订单方向
		PositionSide  string            `json:"positionSide"`  // 单向持仓 BOTH，双向持仓 LONG / SHORT
		UpdateTime    types.ExTimestamp `json:"updateTime"`    // 更新时间（毫秒）
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	// 检查是否有错误码（通过检查 orderId 是否为 0）
	if respData.OrderID == 0 {
		return nil, fmt.Errorf("order not found")
	}

	// 将 Binance 响应转换为 model.PerpOrder
	order := &model.PerpOrder{
		ID:               strconv.FormatInt(respData.OrderID, 10),
		ClientID:         respData.ClientOrderID,
		Type:             respData.Type,
		Side:             respData.Side,
		PositionSide:     respData.PositionSide,
		Symbol:           symbol, // 使用标准化格式
		Price:            respData.Price,
		AvgPrice:         respData.AvgPrice,
		Quantity:         respData.OrigQty,
		ExecutedQuantity: respData.ExecutedQty,
		Status:           respData.Status,
		TimeInForce:      respData.TimeInForce,
		ReduceOnly:       respData.ReduceOnly,
		CreateTime:       respData.Time,
		UpdateTime:       respData.UpdateTime,
	}

	return order, nil
}

// SetLeverage 设置杠杆
func (p *BinancePerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	req := types.NewExValues()

	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}
	req.SetQuery("symbol", market.ID)

	if leverage < 1 || leverage > 125 {
		return fmt.Errorf("leverage must be between 1 and 125")
	}
	req.SetQuery("leverage", leverage)

	_, err = p.signAndRequest(ctx, "POST", "/fapi/v1/leverage", req)
	return err
}

// SetMarginType 设置保证金类型
func (p *BinancePerp) SetMarginType(ctx context.Context, symbol string, marginType option.MarginType) error {
	req := types.NewExValues()

	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}
	req.SetQuery("symbol", market.ID)
	req.SetQuery("marginType", marginType.Upper())

	_, err = p.signAndRequest(ctx, "POST", "/fapi/v1/marginType", req)
	return err
}

// 确保 BinancePerp 实现了 exchange.PerpExchange 接口
var _ exchange.PerpExchange = (*BinancePerp)(nil)

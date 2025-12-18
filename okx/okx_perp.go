package okx

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

// OKXPerp OKX 永续合约实现
type OKXPerp struct {
	okx *OKX
}

// NewOKXPerp 创建 OKX 永续合约实例
func NewOKXPerp(o *OKX) *OKXPerp {
	return &OKXPerp{
		okx: o,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *OKXPerp) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	p.okx.mu.RLock()
	if !reload && len(p.okx.perpMarketsBySymbol) > 0 {
		p.okx.mu.RUnlock()
		return nil
	}
	p.okx.mu.RUnlock()

	// 获取永续合约市场信息
	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/public/instruments", map[string]interface{}{
		"instType": "SWAP",
	})
	if err != nil {
		return fmt.Errorf("fetch swap instruments: %w", err)
	}

	var result okxPerpMarketsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("unmarshal swap instruments: %w", err)
	}

	if result.Code != "0" {
		return fmt.Errorf("okx api error: %s", result.Msg)
	}

	markets := make([]*model.Market, 0)
	for _, item := range result.Data {
		if item.State != "live" {
			continue
		}

		// 对于非现货市场，如果 uly (underlying) 不为空，则从 uly 中解析 base 和 quote
		baseCcy := item.BaseCcy
		quoteCcy := item.QuoteCcy

		if item.Uly != "" {
			// 从 underlying 解析，例如 "BTC-USDT" -> base="BTC", quote="USDT"
			parts := strings.Split(item.Uly, "-")
			if len(parts) >= 2 {
				baseCcy = parts[0]
				quoteCcy = parts[1]
			}
		}

		// 如果 baseCcy 或 quoteCcy 仍为空，从 instId 解析
		if baseCcy == "" || quoteCcy == "" {
			parts := strings.Split(item.InstID, "-")
			if len(parts) >= 2 {
				if baseCcy == "" {
					baseCcy = parts[0]
				}
				if quoteCcy == "" {
					quoteCcy = parts[1]
				}
			}
		}

		// 对于 OKX 永续合约，settleCcy 可能为空，需要从 instId 或根据 ctType 推断
		settle := item.SettleCcy
		if settle == "" {
			// 根据 ctType 判断：linear (U本位) settle=quote, inverse (币本位) settle=base
			switch item.CtType {
			case "linear":
				settle = quoteCcy
			case "inverse":
				settle = baseCcy
			default:
				// 默认 U 本位
				settle = quoteCcy
			}
		}

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(baseCcy, quoteCcy, settle)

		market := &model.Market{
			ID:            item.InstID,
			Symbol:        normalizedSymbol,
			Base:          baseCcy,
			Quote:         quoteCcy,
			Settle:        settle,
			Type:          model.MarketTypeSwap,
			Active:        item.State == "live",
			Contract:      true,
			ContractValue: item.CtVal,               // 合约面值（每张合约等于多少个币）
			Linear:        item.CtType == "linear",  // U本位
			Inverse:       item.CtType == "inverse", // 币本位
		}

		// 解析精度和限制
		if !item.MinSz.IsZero() {
			market.Limits.Amount.Min = item.MinSz
		}
		if !item.MaxSz.IsZero() {
			market.Limits.Amount.Max = item.MaxSz
		}
		if !item.MinSzVal.IsZero() {
			market.Limits.Cost.Min = item.MinSzVal
		}

		// 计算精度
		if !item.LotSz.IsZero() {
			lotSzStr := item.LotSz.String()
			parts := strings.Split(lotSzStr, ".")
			if len(parts) > 1 {
				market.Precision.Amount = len(strings.TrimRight(parts[1], "0"))
			}
		}
		if !item.TickSz.IsZero() {
			tickSzStr := item.TickSz.String()
			parts := strings.Split(tickSzStr, ".")
			if len(parts) > 1 {
				market.Precision.Price = len(strings.TrimRight(parts[1], "0"))
			}
		}

		markets = append(markets, market)
	}

	// 存储市场信息
	p.okx.mu.Lock()
	if p.okx.perpMarketsBySymbol == nil {
		p.okx.perpMarketsBySymbol = make(map[string]*model.Market)
		p.okx.perpMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		p.okx.perpMarketsBySymbol[market.Symbol] = market
		p.okx.perpMarketsByID[market.ID] = market
	}
	p.okx.mu.Unlock()

	return nil
}

func (p *OKXPerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	p.okx.mu.RLock()
	defer p.okx.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.okx.perpMarketsBySymbol))
	for _, market := range p.okx.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *OKXPerp) GetMarket(symbol string) (*model.Market, error) {
	p.okx.mu.RLock()
	defer p.okx.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := p.okx.perpMarketsBySymbol[symbol]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := p.okx.perpMarketsByID[symbol]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", symbol)
}

func (p *OKXPerp) GetMarkets() ([]*model.Market, error) {
	p.okx.mu.RLock()
	defer p.okx.mu.RUnlock()

	markets := make([]*model.Market, 0, len(p.okx.perpMarketsBySymbol))
	for _, market := range p.okx.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *OKXPerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/market/ticker", map[string]interface{}{
		"instId": okxSymbol,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var result okxPerpTickerResponse

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	data := result.Data[0]
	ticker := &model.Ticker{
		Symbol:    symbol,
		Timestamp: data.Ts,
	}

	ticker.Bid = data.BidPx
	ticker.Ask = data.AskPx
	ticker.Last = data.Last
	ticker.Open = data.Open24h
	ticker.High = data.High24h
	ticker.Low = data.Low24h
	ticker.Volume = data.Vol24h
	ticker.QuoteVolume = data.VolCcy24h

	return ticker, nil
}

func (p *OKXPerp) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/market/tickers", map[string]interface{}{
		"instType": "SWAP",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var result okxPerpTickerResponse

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range result.Data {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.InstID)
		if err != nil {
			continue
		}
		ticker := &model.Ticker{
			Symbol:    market.Symbol,
			Timestamp: item.Ts,
		}
		ticker.Bid = item.BidPx
		ticker.Ask = item.AskPx
		ticker.Last = item.Last
		ticker.Open = item.Open24h
		ticker.High = item.High24h
		ticker.Low = item.Low24h
		ticker.Volume = item.Vol24h
		ticker.QuoteVolume = item.VolCcy24h
		tickers[market.Symbol] = ticker
	}

	return tickers, nil
}

func (p *OKXPerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
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
	normalizedTimeframe := common.OKXTimeframe(timeframe)

	params := map[string]interface{}{
		"instId": market.ID,
		"bar":    normalizedTimeframe,
		"limit":  limit,
	}
	if !since.IsZero() {
		params["after"] = since.UnixMilli()
	}

	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/market/candles", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var result okxPerpKlineResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	ohlcvs := make(model.OHLCVs, 0, len(result.Data))
	for _, item := range result.Data {
		ohlcv := &model.OHLCV{
			Timestamp: item.Ts,
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

func (p *OKXPerp) FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error) {
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

func (p *OKXPerp) CreateOrder(ctx context.Context, symbol string, amount string, orderSide option.PerpOrderSide, orderType option.OrderType, opts ...option.ArgsOption) (*model.NewOrder, error) {
	// 解析选项
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
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

	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// 确定交易模式（合约默认全仓）
	tdMode := "cross"

	// 计算 sz（数量）
	// 合约需要将币数量转换为张数
	// 转换公式：张数 = 币的个数 / ctVal
	sz := amount
	if market.ContractValue != "" {
		exContractValue, err := decimal.NewFromString(market.ContractValue)
		if err != nil || exContractValue.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("invalid contract value: %s", market.ContractValue)
		}

		amountDecimal, err := decimal.NewFromString(amount)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %w", err)
		}

		contractSizeDecimal := amountDecimal.Div(exContractValue)
		szPrecision := market.Precision.Amount
		if szPrecision == 0 {
			szPrecision = 8
		}
		sz = contractSizeDecimal.StringFixed(int32(szPrecision))
	}

	// 构建请求结构体
	req := okxPerpCreateOrderRequest{
		InstID:  okxSymbol,
		TdMode:  tdMode,
		Side:    strings.ToLower(orderSide.ToSide()),
		OrdType: orderType.Lower(),
		Sz:      sz,
	}

	// 限价单设置价格
	if orderType == option.Limit {
		req.Px = priceStr
	}

	// 从 PerpOrderSide 自动推断 PositionSide 和 reduceOnly
	positionSideStr := orderSide.ToPositionSide()
	reduceOnly := orderSide.ToReduceOnly()

	// 获取持仓模式：从 hedgeMode 选项获取，未设置时默认为 net_mode（单向持仓模式）
	posMode := "net_mode"
	if argsOpts.HedgeMode != nil {
		if *argsOpts.HedgeMode {
			posMode = "long_short_mode"
		} else {
			posMode = "net_mode"
		}
	}

	if posMode == "long_short_mode" {
		// 双向持仓模式
		// 开多/平多: posSide=long
		// 开空/平空: posSide=short
		req.PosSide = strings.ToLower(positionSideStr)
	} else {
		// 单向持仓模式
		req.PosSide = "net"

		// 如果是平仓操作，设置 reduceOnly
		if reduceOnly {
			req.ReduceOnly = true
		}
	}

	// 客户端订单ID
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		req.ClOrdID = *argsOpts.ClientOrderID
	} else {
		// 将 PerpOrderSide 转换为 OrderSide 用于生成订单ID
		orderSideForID := model.OrderSide(strings.ToLower(orderSide.ToSide()))
		req.ClOrdID = common.GenerateClientOrderID(p.okx.Name(), orderSideForID)
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

	resp, err := p.signAndRequest(ctx, "POST", "/api/v5/trade/order", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result okxPerpCreateOrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.Code != "0" {
		errMsg := result.Msg
		if len(result.Data) > 0 {
			errMsg = result.Msg
		}
		return nil, fmt.Errorf("okx api error: %s", errMsg)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: no order data returned")
	}

	data := result.Data[0]

	// 构建 NewOrder 对象
	perpOrder := &model.NewOrder{
		Symbol:        symbol,
		OrderId:       data.OrdID,
		ClientOrderID: data.ClOrdID,
		Timestamp:     data.TS,
	}

	return perpOrder, nil
}

func (p *OKXPerp) CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error {
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
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, true)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"instId": okxSymbol,
	}

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		reqBody["ordId"] = orderId
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		reqBody["clOrdId"] = *argsOpts.ClientOrderID
	} else {
		return fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	_, err = p.signAndRequest(ctx, "POST", "/api/v5/trade/cancel-order", nil, reqBody)
	return err
}

func (p *OKXPerp) FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.PerpOrder, error) {
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
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"instType": "SWAP",
		"instId":   okxSymbol,
	}

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		params["ordId"] = orderId
	} else if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["clOrdId"] = *argsOpts.ClientOrderID
	} else {
		return nil, fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	resp, err := p.signAndRequest(ctx, "GET", "/api/v5/trade/order", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var result okxPerpFetchOrderResponse

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	// 将 OKX 响应转换为 model.PerpOrder
	item := result.Data[0]
	// 转换 reduceOnly 字符串为 bool
	reduceOnly := strings.ToLower(item.ReduceOnly) == "true"

	order := &model.PerpOrder{
		ID:               item.OrdID,
		ClientID:         item.ClOrdID,
		Type:             item.OrdType,
		Side:             item.Side,
		PositionSide:     item.PosSide,
		Symbol:           symbol,
		Price:            item.Px,
		AvgPrice:         item.AvgPx,
		Quantity:         item.Sz,
		ExecutedQuantity: item.AccFillSz,
		Status:           item.State,
		TimeInForce:      "", // OKX 响应中没有 timeInForce 字段
		ReduceOnly:       reduceOnly,
		CreateTime:       item.CTime,
		UpdateTime:       item.UTime,
	}

	return order, nil
}

func (p *OKXPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("leverage only supported for contracts")
	}

	reqBody := map[string]interface{}{
		"instId": market.ID,
		"lever":  strconv.Itoa(leverage),
	}

	_, err = p.signAndRequest(ctx, "POST", "/api/v5/account/set-leverage", nil, reqBody)
	return err
}

func (p *OKXPerp) SetMarginType(ctx context.Context, symbol string, marginType option.MarginType) error {
	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}

	if !market.Contract {
		return fmt.Errorf("margin mode only supported for contracts")
	}

	// 验证类型
	if marginType != option.ISOLATED && marginType != option.CROSSED {
		return fmt.Errorf("invalid margin type: %s, must be 'ISOLATED' or 'CROSSED'", marginType)
	}

	reqBody := map[string]interface{}{
		"instId":  market.ID,
		"mgnMode": marginType.Upper(),
	}

	_, err = p.signAndRequest(ctx, "POST", "/api/v5/account/set-margin-mode", nil, reqBody)
	return err
}

var _ exchange.PerpExchange = (*OKXPerp)(nil)

// ========== 内部辅助方法 ==========

// signAndRequest 签名并发送请求（OKX API）
func (p *OKXPerp) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if p.okx.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
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

	// 生成时间戳和签名
	timestamp := common.GetISO8601Timestamp()
	signature := p.okx.signer.SignRequest(method, path, timestamp, bodyStr, params)

	// 设置请求头
	p.okx.client.HTTPClient.SetHeader("OK-ACCESS-SIGN", signature)
	p.okx.client.HTTPClient.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	p.okx.client.HTTPClient.SetHeader("OK-ACCESS-PASSPHRASE", p.okx.client.Passphrase)
	p.okx.client.HTTPClient.SetHeader("OK-ACCESS-KEY", p.okx.client.APIKey)
	if p.okx.client.Sandbox {
		p.okx.client.HTTPClient.SetHeader("x-simulated-trading", "1")
	}
	p.okx.client.HTTPClient.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return p.okx.client.HTTPClient.Get(ctx, path, params)
	} else {
		return p.okx.client.HTTPClient.Post(ctx, path, body)
	}
}

func (p *OKXPerp) fetchPositions(ctx context.Context, symbols ...string) (model.Positions, error) {
	params := map[string]interface{}{
		"instType": "SWAP",
	}

	if len(symbols) > 0 {
		// 获取市场信息
		market, err := p.GetMarket(symbols[0])
		if err == nil {
			params["instId"] = market.ID
		} else {
			// 如果市场未加载，尝试转换
			okxSymbol, err := ToOKXSymbol(symbols[0], true)
			if err == nil {
				params["instId"] = okxSymbol
			}
		}
	}

	resp, err := p.signAndRequest(ctx, "GET", "/api/v5/account/positions", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var result okxPerpPositionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	positions := make([]*model.Position, 0)
	for _, item := range result.Data {
		pos, _ := item.Pos.Float64()
		if pos == 0 {
			continue
		}

		market, err := p.GetMarket(item.InstID)
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
		// 根据 posSide 确定持仓方向
		if item.PosSide == "long" || (item.PosSide == "net" && pos > 0) {
			side = string(types.PositionSideLong)
		} else {
			side = string(types.PositionSideShort)
		}

		position := &model.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           item.Pos,
			EntryPrice:       item.AvgPx,
			MarkPrice:        item.MarkPx,
			UnrealizedPnl:    item.Upl,
			LiquidationPrice: item.LiqPx,
			RealizedPnl:      item.RealizedPnl,
			Leverage:         item.Lever,
			Margin:           item.Margin,
			Percentage:       types.ExDecimal{},
			Timestamp:        item.UTime,
		}

		positions = append(positions, position)
	}

	return positions, nil
}

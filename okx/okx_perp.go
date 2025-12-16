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
	okx    *OKX
	market *okxPerpMarket
	order  *okxPerpOrder
}

// NewOKXPerp 创建 OKX 永续合约实例
func NewOKXPerp(o *OKX) *OKXPerp {
	return &OKXPerp{
		okx:    o,
		market: &okxPerpMarket{okx: o},
		order:  &okxPerpOrder{okx: o},
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *OKXPerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

func (p *OKXPerp) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return p.market.FetchMarkets(ctx)
}

func (p *OKXPerp) GetMarket(symbol string) (*model.Market, error) {
	return p.market.GetMarket(symbol)
}

func (p *OKXPerp) GetMarkets() ([]*model.Market, error) {
	return p.market.GetMarkets()
}

func (p *OKXPerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

func (p *OKXPerp) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	return p.market.FetchTickers(ctx)
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
	return p.market.FetchOHLCVs(ctx, symbol, timeframe, since, limit)
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
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *OKXPerp) CreateOrder(ctx context.Context, symbol string, side option.PerpOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *OKXPerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *OKXPerp) FetchOrder(ctx context.Context, orderID, symbol string) (*model.PerpOrder, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *OKXPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *OKXPerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

var _ exchange.PerpExchange = (*OKXPerp)(nil)

// ========== 内部实现 ==========

type okxPerpMarket struct {
	okx *OKX
}

func (m *okxPerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.okx.mu.RLock()
	if !reload && len(m.okx.perpMarketsBySymbol) > 0 {
		m.okx.mu.RUnlock()
		return nil
	}
	m.okx.mu.RUnlock()

	// 获取永续合约市场信息
	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/public/instruments", map[string]interface{}{
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
	m.okx.mu.Lock()
	if m.okx.perpMarketsBySymbol == nil {
		m.okx.perpMarketsBySymbol = make(map[string]*model.Market)
		m.okx.perpMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.okx.perpMarketsBySymbol[market.Symbol] = market
		m.okx.perpMarketsByID[market.ID] = market
	}
	m.okx.mu.Unlock()

	return nil
}

func (m *okxPerpMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.okx.perpMarketsBySymbol))
	for _, market := range m.okx.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *okxPerpMarket) GetMarket(key string) (*model.Market, error) {
	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := m.okx.perpMarketsBySymbol[key]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := m.okx.perpMarketsByID[key]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", key)
}

func (m *okxPerpMarket) GetMarkets() ([]*model.Market, error) {
	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.okx.perpMarketsBySymbol))
	for _, market := range m.okx.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *okxPerpMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
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

	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/market/ticker", map[string]interface{}{
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

func (m *okxPerpMarket) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/market/tickers", map[string]interface{}{
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
		market, err := m.GetMarket(item.InstID)
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

func (m *okxPerpMarket) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
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

	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/market/candles", params)
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

type okxPerpOrder struct {
	okx             *OKX
	positionMode    *string   // 持仓模式缓存: "long_short_mode"=双向, "net_mode"=单向
	positionModeExp time.Time // 持仓模式缓存过期时间
}

// signAndRequest 签名并发送请求（OKX API）
func (o *okxPerpOrder) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if o.okx.client.SecretKey == "" {
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
	signature := o.okx.signer.SignRequest(method, path, timestamp, bodyStr, params)

	// 设置请求头
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-SIGN", signature)
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-PASSPHRASE", o.okx.client.Passphrase)
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-KEY", o.okx.client.APIKey)
	if o.okx.client.Sandbox {
		o.okx.client.HTTPClient.SetHeader("x-simulated-trading", "1")
	}
	o.okx.client.HTTPClient.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return o.okx.client.HTTPClient.Get(ctx, path, params)
	} else {
		return o.okx.client.HTTPClient.Post(ctx, path, body)
	}
}

// getPositionMode 获取持仓模式（带缓存）
// 返回: "long_short_mode"=双向持仓, "net_mode"=单向持仓
func (o *okxPerpOrder) getPositionMode(ctx context.Context) (string, error) {
	// 检查缓存是否有效（5分钟）
	if o.positionMode != nil && time.Now().Before(o.positionModeExp) {
		return *o.positionMode, nil
	}

	// 查询账户配置
	timestamp := common.GetISO8601Timestamp()
	signature := o.okx.signer.SignRequest("GET", "/api/v5/account/config", timestamp, "", nil)

	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-SIGN", signature)
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-PASSPHRASE", o.okx.client.Passphrase)
	o.okx.client.HTTPClient.SetHeader("OK-ACCESS-KEY", o.okx.client.APIKey)
	if o.okx.client.Sandbox {
		o.okx.client.HTTPClient.SetHeader("x-simulated-trading", "1")
	}

	resp, err := o.okx.client.HTTPClient.Get(ctx, "/api/v5/account/config", nil)
	if err != nil {
		return "", fmt.Errorf("get account config: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			PosMode string `json:"posMode"` // long_short_mode 或 net_mode
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("unmarshal account config: %w", err)
	}

	if result.Code != "0" {
		return "", fmt.Errorf("okx api error: %s", result.Msg)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("no account config data")
	}

	posMode := result.Data[0].PosMode
	if posMode == "" {
		posMode = "net_mode" // 默认单向持仓
	}

	// 缓存结果
	o.positionMode = &posMode
	o.positionModeExp = time.Now().Add(5 * time.Minute)

	return posMode, nil
}

func (o *okxPerpOrder) FetchPositions(ctx context.Context, symbols ...string) (model.Positions, error) {
	params := map[string]interface{}{
		"instType": "SWAP",
	}

	if len(symbols) > 0 {
		// 获取市场信息
		market, err := o.okx.perp.market.GetMarket(symbols[0])
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

	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/account/positions", params, nil)
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

		market, err := o.okx.perp.market.GetMarket(item.InstID)
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

func (o *okxPerpOrder) CreateOrder(ctx context.Context, symbol string, side option.PerpOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
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

	// 获取市场信息
	market, err := o.okx.perp.market.GetMarket(symbol)
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
		Side:    strings.ToLower(side.ToSide()),
		OrdType: orderType.Lower(),
		Sz:      sz,
	}

	// 限价单设置价格
	if orderType == option.Limit {
		req.Px = priceStr
	}

	// 从 PerpOrderSide 自动推断 PositionSide 和 reduceOnly
	positionSideStr := side.ToPositionSide()
	reduceOnly := side.ToReduceOnly()

	// 获取持仓模式：如果提供了 hedgeMode 选项，使用它；否则查询 API
	var posMode string
	if argsOpts.HedgeMode != nil {
		if *argsOpts.HedgeMode {
			posMode = "long_short_mode"
		} else {
			posMode = "net_mode"
		}
	} else {
		var err error
		posMode, err = o.getPositionMode(ctx)
		if err != nil {
			return nil, fmt.Errorf("get position mode: %w", err)
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
		orderSide := types.OrderSide(strings.ToLower(side.ToSide()))
		req.ClOrdID = common.GenerateClientOrderID(o.okx.Name(), orderSide)
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

	resp, err := o.signAndRequest(ctx, "POST", "/api/v5/trade/order", nil, reqBody)
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

// parseOrder 解析订单数据（合约版本）
// parsePerpOrder 将 OKX 响应转换为 model.PerpOrder
func (o *okxPerpOrder) parsePerpOrder(item okxPerpFetchOrderItem, symbol string) *model.PerpOrder {
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

	return order
}

func (o *okxPerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// 获取市场信息
	market, err := o.okx.perp.market.GetMarket(symbol)
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
		"ordId":  orderID,
	}

	_, err = o.signAndRequest(ctx, "POST", "/api/v5/trade/cancel-order", nil, reqBody)
	return err
}

func (o *okxPerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*model.PerpOrder, error) {
	// 获取市场信息
	market, err := o.okx.perp.market.GetMarket(symbol)
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
		"ordId":    orderID,
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/trade/order", params, nil)
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

	return o.parsePerpOrder(result.Data[0], symbol), nil
}

func (o *okxPerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := o.okx.perp.market.GetMarket(symbol)
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

	_, err = o.signAndRequest(ctx, "POST", "/api/v5/account/set-leverage", nil, reqBody)
	return err
}

func (o *okxPerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	market, err := o.okx.perp.market.GetMarket(symbol)
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

	reqBody := map[string]interface{}{
		"instId":  market.ID,
		"mgnMode": strings.ToUpper(mode),
	}

	_, err = o.signAndRequest(ctx, "POST", "/api/v5/account/set-margin-mode", nil, reqBody)
	return err
}

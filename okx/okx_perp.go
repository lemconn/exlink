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
	"github.com/lemconn/exlink/types"
	"github.com/shopspring/decimal"
)

// OKXPerp OKX 永续合约实现
type OKXPerp struct {
	okx       *OKX
	market    *okxPerpMarket
	order     *okxPerpOrder
	hedgeMode bool
}

// NewOKXPerp 创建 OKX 永续合约实例
func NewOKXPerp(o *OKX) *OKXPerp {
	return &OKXPerp{
		okx:       o,
		market:    &okxPerpMarket{okx: o},
		order:     &okxPerpOrder{okx: o},
		hedgeMode: false,
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

func (p *OKXPerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	return p.market.FetchOHLCVs(ctx, symbol, timeframe, since, limit)
}

func (p *OKXPerp) FetchPositions(ctx context.Context, symbols ...string) (model.Positions, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *OKXPerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *OKXPerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *OKXPerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *OKXPerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

func (p *OKXPerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

func (p *OKXPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *OKXPerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

func (p *OKXPerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

func (p *OKXPerp) IsHedgeMode() bool {
	return p.hedgeMode
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

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID   string `json:"instId"`
			Pos      string `json:"pos"`
			AvgPx    string `json:"avgPx"`
			MarkPx   string `json:"markPx"`
			Upl      string `json:"upl"`
			UplRatio string `json:"uplRatio"`
			PosSide  string `json:"posSide"` // net, long, short
			MgnMode  string `json:"mgnMode"` // isolated, cross
			Lever    string `json:"lever"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	positions := make([]*model.Position, 0)
	for _, item := range result.Data {
		pos, _ := strconv.ParseFloat(item.Pos, 64)
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

		entryPrice, _ := strconv.ParseFloat(item.AvgPx, 64)
		markPrice, _ := strconv.ParseFloat(item.MarkPx, 64)
		unrealizedPnl, _ := strconv.ParseFloat(item.Upl, 64)

		var side string
		// 根据 posSide 确定持仓方向
		if item.PosSide == "long" || (item.PosSide == "net" && pos > 0) {
			side = string(types.PositionSideLong)
		} else {
			side = string(types.PositionSideShort)
		}

		position := &model.Position{
			Symbol:         market.Symbol,
			Side:           side,
			Amount:         types.ExDecimal{Decimal: decimal.NewFromFloat(pos)},
			EntryPrice:     types.ExDecimal{Decimal: decimal.NewFromFloat(entryPrice)},
			MarkPrice:      types.ExDecimal{Decimal: decimal.NewFromFloat(markPrice)},
			UnrealizedPnl:  types.ExDecimal{Decimal: decimal.NewFromFloat(unrealizedPnl)},
			LiquidationPrice: types.ExDecimal{},
			RealizedPnl:     types.ExDecimal{},
			Leverage:        types.ExDecimal{},
			Margin:          types.ExDecimal{},
			Percentage:      types.ExDecimal{},
			Timestamp:       types.ExTimestamp{Time: time.Now()},
		}

		positions = append(positions, position)
	}

	return positions, nil
}

func (o *okxPerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
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

	reqBody := map[string]interface{}{
		"instId":  okxSymbol,
		"tdMode":  tdMode,
		"side":    strings.ToLower(string(side)),
		"ordType": strings.ToLower(string(orderType)),
		"sz":      sz,
	}

	// 限价单设置价格
	if orderType == types.OrderTypeLimit {
		reqBody["px"] = priceStr
	}

	// 合约订单处理持仓方向
	if options.PositionSide == nil {
		return nil, fmt.Errorf("contract order requires PositionSide (long/short)")
	}

	// 获取持仓模式
	posMode, err := o.getPositionMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("get position mode: %w", err)
	}

	if posMode == "long_short_mode" {
		// 双向持仓模式
		// 开多/平多: posSide=long
		// 开空/平空: posSide=short
		if *options.PositionSide == types.PositionSideLong {
			reqBody["posSide"] = "long"
		} else {
			reqBody["posSide"] = "short"
		}
	} else {
		// 单向持仓模式
		reqBody["posSide"] = "net"

		// 判断是否为平仓操作
		// 平多：PositionSideLong + SideSell -> reduceOnly = true
		// 平空：PositionSideShort + SideBuy -> reduceOnly = true
		if (*options.PositionSide == types.PositionSideLong && side == types.OrderSideSell) ||
			(*options.PositionSide == types.PositionSideShort && side == types.OrderSideBuy) {
			reqBody["reduceOnly"] = true
		}
	}

	// 客户端订单ID
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqBody["clOrdId"] = *options.ClientOrderID
	} else {
		reqBody["clOrdId"] = common.GenerateClientOrderID(o.okx.Name(), side)
	}

	resp, err := o.signAndRequest(ctx, "POST", "/api/v5/trade/order", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			OrdID   string `json:"ordId"`
			ClOrdID string `json:"clOrdId"`
			Tag     string `json:"tag"`
			SCode   string `json:"sCode"`
			SMsg    string `json:"sMsg"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.Code != "0" {
		errMsg := result.Msg
		if len(result.Data) > 0 && result.Data[0].SMsg != "" {
			errMsg = fmt.Sprintf("%s: %s", result.Msg, result.Data[0].SMsg)
		}
		return nil, fmt.Errorf("okx api error: %s", errMsg)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: no order data returned")
	}

	data := result.Data[0]
	if data.SCode != "" && data.SCode != "0" {
		errMsg := data.SMsg
		if errMsg == "" {
			errMsg = result.Msg
		}
		return nil, fmt.Errorf("okx api error: %s (code: %s)", errMsg, data.SCode)
	}

	amountFloat, _ := strconv.ParseFloat(amount, 64)
	var priceFloat float64
	if priceStr != "" {
		priceFloat, _ = strconv.ParseFloat(priceStr, 64)
	}

	order := &types.Order{
		ID:            data.OrdID,
		ClientOrderID: data.ClOrdID,
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
func (o *okxPerpOrder) parseOrder(item struct {
	InstID    string `json:"instId"`
	OrdID     string `json:"ordId"`
	ClOrdID   string `json:"clOrdId"`
	State     string `json:"state"`
	Side      string `json:"side"`
	OrdType   string `json:"ordType"`
	Px        string `json:"px"`
	Sz        string `json:"sz"`
	AccFillSz string `json:"accFillSz"`
	UTime     string `json:"uTime"`
}, symbol string) *types.Order {
	order := &types.Order{
		ID:            item.OrdID,
		ClientOrderID: item.ClOrdID,
		Symbol:        symbol,
		Timestamp:     time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(item.Px, 64)
	order.Amount, _ = strconv.ParseFloat(item.Sz, 64)
	order.Filled, _ = strconv.ParseFloat(item.AccFillSz, 64)
	order.Remaining = order.Amount - order.Filled

	if strings.ToLower(item.Side) == "buy" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToLower(item.OrdType) == "market" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	switch item.State {
	case "live":
		order.Status = types.OrderStatusOpen
	case "partially_filled":
		order.Status = types.OrderStatusPartiallyFilled
	case "filled":
		order.Status = types.OrderStatusFilled
	case "canceled":
		order.Status = types.OrderStatusCanceled
	default:
		order.Status = types.OrderStatusNew
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

func (o *okxPerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
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

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID    string `json:"instId"`
			OrdID     string `json:"ordId"`
			ClOrdID   string `json:"clOrdId"`
			State     string `json:"state"`
			Side      string `json:"side"`
			OrdType   string `json:"ordType"`
			Px        string `json:"px"`
			Sz        string `json:"sz"`
			AccFillSz string `json:"accFillSz"`
			UTime     string `json:"uTime"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	return o.parseOrder(result.Data[0], symbol), nil
}

func (o *okxPerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
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
		"instId": okxSymbol,
		"limit":  limit,
	}
	if !since.IsZero() {
		params["after"] = since.UnixMilli()
	}

	resp, err := o.okx.client.HTTPClient.Get(ctx, "/api/v5/market/trades", params)
	if err != nil {
		return nil, fmt.Errorf("fetch trades: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID  string `json:"instId"`
			TradeID string `json:"tradeId"`
			Px      string `json:"px"`
			Sz      string `json:"sz"`
			Side    string `json:"side"`
			Ts      string `json:"ts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	trades := make([]*types.Trade, 0, len(result.Data))
	for _, item := range result.Data {
		price, _ := strconv.ParseFloat(item.Px, 64)
		sz, _ := strconv.ParseFloat(item.Sz, 64)
		ts, _ := strconv.ParseInt(item.Ts, 10, 64)

		trade := &types.Trade{
			ID:        item.TradeID,
			Symbol:    symbol,
			Price:     price,
			Amount:    sz,
			Cost:      price * sz,
			Timestamp: time.UnixMilli(ts),
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

func (o *okxPerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
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
		"limit":    limit,
	}
	if !since.IsZero() {
		params["after"] = since.UnixMilli()
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/trade/fills", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID  string `json:"instId"`
			TradeID string `json:"tradeId"`
			OrdID   string `json:"ordId"`
			Px      string `json:"px"`
			Sz      string `json:"sz"`
			Side    string `json:"side"`
			Fee     string `json:"fee"`
			FeeCcy  string `json:"feeCcy"`
			Ts      string `json:"ts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	trades := make([]*types.Trade, 0, len(result.Data))
	for _, item := range result.Data {
		price, _ := strconv.ParseFloat(item.Px, 64)
		sz, _ := strconv.ParseFloat(item.Sz, 64)
		ts, _ := strconv.ParseInt(item.Ts, 10, 64)
		fee, _ := strconv.ParseFloat(item.Fee, 64)

		trade := &types.Trade{
			ID:        item.TradeID,
			OrderID:   item.OrdID,
			Symbol:    symbol,
			Price:     price,
			Amount:    sz,
			Cost:      price * sz,
			Timestamp: time.UnixMilli(ts),
		}

		if strings.ToLower(item.Side) == "buy" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		if fee > 0 && item.FeeCcy != "" {
			trade.Fee = &types.Fee{
				Currency: item.FeeCcy,
				Cost:     fee,
			}
		}

		trades = append(trades, trade)
	}

	return trades, nil
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

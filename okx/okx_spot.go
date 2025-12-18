package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/option"
	"github.com/lemconn/exlink/types"
)

// OKXSpot OKX 现货实现
type OKXSpot struct {
	okx    *OKX
	market *okxSpotMarket
	order  *okxSpotOrder
}

// NewOKXSpot 创建 OKX 现货实例
func NewOKXSpot(o *OKX) *OKXSpot {
	return &OKXSpot{
		okx:    o,
		market: &okxSpotMarket{okx: o},
		order:  &okxSpotOrder{okx: o},
	}
}

// ========== SpotExchange 接口实现 ==========

func (s *OKXSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

func (s *OKXSpot) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *OKXSpot) GetMarket(symbol string) (*model.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *OKXSpot) GetMarkets() ([]*model.Market, error) {
	return s.market.GetMarkets()
}

func (s *OKXSpot) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *OKXSpot) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	return s.market.FetchTickers(ctx)
}

func (s *OKXSpot) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
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
	return s.market.FetchOHLCVs(ctx, symbol, timeframe, since, limit)
}

func (s *OKXSpot) FetchBalance(ctx context.Context) (model.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *OKXSpot) CreateOrder(ctx context.Context, symbol string, side option.SpotOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *OKXSpot) CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error {
	return s.order.CancelOrder(ctx, symbol, orderId, opts...)
}

func (s *OKXSpot) FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.SpotOrder, error) {
	return s.order.FetchOrder(ctx, symbol, orderId, opts...)
}

var _ exchange.SpotExchange = (*OKXSpot)(nil)

// ========== 内部实现 ==========

type okxSpotMarket struct {
	okx *OKX
}

func (m *okxSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.okx.mu.RLock()
	if !reload && len(m.okx.spotMarketsBySymbol) > 0 {
		m.okx.mu.RUnlock()
		return nil
	}
	m.okx.mu.RUnlock()

	// 获取现货市场信息
	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/public/instruments", map[string]interface{}{
		"instType": "SPOT",
	})
	if err != nil {
		return fmt.Errorf("fetch instruments: %w", err)
	}

	var result okxSpotMarketsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("unmarshal instruments: %w", err)
	}

	if result.Code != "0" {
		return fmt.Errorf("okx api error: %s", result.Msg)
	}

	markets := make([]*model.Market, 0)
	for _, item := range result.Data {
		if item.State != "live" {
			continue
		}

		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(item.BaseCcy, item.QuoteCcy)

		market := &model.Market{
			ID:     item.InstID, // OKX 使用 InstID 作为市场ID
			Symbol: normalizedSymbol,
			Base:   item.BaseCcy,
			Quote:  item.QuoteCcy,
			Type:   model.MarketTypeSpot,
			Active: item.State == "live",
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
	if m.okx.spotMarketsBySymbol == nil {
		m.okx.spotMarketsBySymbol = make(map[string]*model.Market)
		m.okx.spotMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.okx.spotMarketsBySymbol[market.Symbol] = market
		m.okx.spotMarketsByID[market.ID] = market
	}
	m.okx.mu.Unlock()

	return nil
}

func (m *okxSpotMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.okx.spotMarketsBySymbol))
	for _, market := range m.okx.spotMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *okxSpotMarket) GetMarket(key string) (*model.Market, error) {
	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := m.okx.spotMarketsBySymbol[key]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := m.okx.spotMarketsByID[key]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", key)
}

func (m *okxSpotMarket) GetMarkets() ([]*model.Market, error) {
	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.okx.spotMarketsBySymbol))
	for _, market := range m.okx.spotMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *okxSpotMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, false)
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

	var result okxSpotTickerResponse

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

func (m *okxSpotMarket) FetchTickers(ctx context.Context) (map[string]*model.Ticker, error) {
	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/market/tickers", map[string]interface{}{
		"instType": "SPOT",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var result okxSpotTickerResponse

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

func (m *okxSpotMarket) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
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

	var result okxSpotKlineResponse
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

type okxSpotOrder struct {
	okx *OKX
}

// signAndRequest 签名并发送请求（OKX API）
func (o *okxSpotOrder) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
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

func (o *okxSpotOrder) FetchBalance(ctx context.Context) (model.Balances, error) {
	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/account/balance", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var result okxSpotBalanceResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	balances := make(model.Balances, 0)
	for _, detail := range result.Data[0].Details {
		balance := &model.Balance{
			Currency:  detail.Ccy,
			Available: detail.AvailBal,
			Locked:    detail.FrozenBal,
			Total:     detail.Eq,
			UpdatedAt: detail.UTime,
		}
		balances = append(balances, balance)
	}

	return balances, nil
}

func (o *okxSpotOrder) CreateOrder(ctx context.Context, symbol string, side option.SpotOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error) {
	// 解析选项
	options := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 判断订单类型
	var orderType model.OrderType
	var priceStr string
	if options.Price != nil && *options.Price != "" {
		orderType = model.OrderTypeLimit
		priceStr = *options.Price
	} else {
		orderType = model.OrderTypeMarket
		priceStr = ""
	}

	// 获取市场信息
	market, err := o.okx.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// 确定交易模式（现货默认 cash）
	tdMode := "cash"

	// 计算 sz（数量）
	sz := amount

	reqBody := map[string]interface{}{
		"instId":  okxSymbol,
		"tdMode":  tdMode,
		"side":    strings.ToLower(string(side)),
		"ordType": strings.ToLower(string(orderType)),
		"sz":      sz,
	}

	// 现货订单设置 tgtCcy
	reqBody["tgtCcy"] = "base_ccy"

	// 限价单设置价格
	if orderType == model.OrderTypeLimit {
		reqBody["px"] = priceStr
	}

	// 客户端订单ID
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqBody["clOrdId"] = *options.ClientOrderID
	} else {
		reqBody["clOrdId"] = common.GenerateClientOrderID(o.okx.Name(), side.ToSide())
	}

	resp, err := o.signAndRequest(ctx, "POST", "/api/v5/trade/order", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result okxSpotCreateOrderResponse
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

	order := &model.NewOrder{
		OrderId:       data.OrdId,
		ClientOrderID: data.ClOrdId,
		Symbol:        symbol,
		Timestamp:     data.Ts,
	}

	return order, nil
}

func (o *okxSpotOrder) CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error {
	// 获取市场信息
	market, err := o.okx.spot.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取交易所格式的 symbol ID
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, false)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"instId": okxSymbol,
		"ordId":  orderId,
	}
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		reqBody["clOrdId"] = *argsOpts.ClientOrderID
	}

	_, err = o.signAndRequest(ctx, "POST", "/api/v5/trade/cancel-order", nil, reqBody)
	return err
}

// parseOrder 解析订单数据
func (o *okxSpotOrder) parseOrder(item okxSpotFetchOrderData, symbol string) *model.SpotOrder {
	// 计算剩余数量
	remaining := item.Sz.Sub(item.AccFillSz.Decimal)

	// 转换状态
	var status model.OrderStatus
	switch item.State {
	case "live":
		status = model.OrderStatusOpen
	case "partially_filled":
		status = model.OrderStatusOpen
	case "filled":
		status = model.OrderStatusFilled
	case "canceled":
		status = model.OrderStatusCanceled
	default:
		status = model.OrderStatusNew
	}

	// 转换订单类型
	var orderType model.OrderType
	if strings.ToLower(item.OrdType) == "market" {
		orderType = model.OrderTypeMarket
	} else {
		orderType = model.OrderTypeLimit
	}

	// 转换订单方向
	var side model.OrderSide
	if strings.ToLower(item.Side) == "buy" {
		side = model.OrderSideBuy
	} else {
		side = model.OrderSideSell
	}

	order := &model.SpotOrder{
		ID:            item.OrdID,
		ClientOrderID: item.ClOrdID,
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Price:         item.Px,
		Amount:        item.Sz,
		Filled:        item.AccFillSz,
		Remaining:     types.ExDecimal{Decimal: remaining},
		Average:       item.AvgPx,
		Status:        status,
		CreatedAt:     item.CTime,
		UpdatedAt:     item.UTime,
	}

	return order
}

func (o *okxSpotOrder) FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.SpotOrder, error) {
	// 获取市场信息
	market, err := o.okx.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取交易所格式的 symbol ID
	okxSymbol := market.ID
	if okxSymbol == "" {
		var err error
		okxSymbol, err = ToOKXSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"instId": okxSymbol,
		"ordId":  orderId,
	}
	if argsOpts.ClientOrderID != nil && *argsOpts.ClientOrderID != "" {
		params["clOrdId"] = *argsOpts.ClientOrderID
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/trade/order", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var result okxSpotFetchOrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	return o.parseOrder(result.Data[0], symbol), nil
}

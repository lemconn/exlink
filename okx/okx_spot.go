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

func (s *OKXSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *OKXSpot) GetMarket(symbol string) (*types.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *OKXSpot) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *OKXSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

func (s *OKXSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (s *OKXSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *OKXSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *OKXSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *OKXSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

func (s *OKXSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

func (s *OKXSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

func (s *OKXSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

func (s *OKXSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

var _ exchange.SpotExchange = (*OKXSpot)(nil)

// ========== 内部实现 ==========

type okxSpotMarket struct {
	okx *OKX
}

func (m *okxSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.okx.mu.RLock()
	if !reload && len(m.okx.spotMarkets) > 0 {
		m.okx.mu.RUnlock()
		return nil
	}
	m.okx.mu.RUnlock()

	// 加载现货市场
	markets, err := m.loadSpotMarkets(ctx)
	if err != nil {
		return fmt.Errorf("load spot markets: %w", err)
	}

	// 存储市场信息
	m.okx.mu.Lock()
	if m.okx.spotMarkets == nil {
		m.okx.spotMarkets = make(map[string]*types.Market)
	}
	for _, market := range markets {
		m.okx.spotMarkets[market.Symbol] = market
	}
	m.okx.mu.Unlock()

	return nil
}

// loadSpotMarkets 加载现货市场
func (m *okxSpotMarket) loadSpotMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/public/instruments", map[string]interface{}{
		"instType": "SPOT",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch instruments: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstType string `json:"instType"`
			InstID   string `json:"instId"`
			BaseCcy  string `json:"baseCcy"`
			QuoteCcy string `json:"quoteCcy"`
			State    string `json:"state"`
			MinSz    string `json:"minSz"`
			MaxSz    string `json:"maxSz"`
			LotSz    string `json:"lotSz"`
			TickSz   string `json:"tickSz"`
			MinSzVal string `json:"minSzVal"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal instruments: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	markets := make([]*types.Market, 0)
	for _, item := range result.Data {
		if item.State != "live" {
			continue
		}

		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(item.BaseCcy, item.QuoteCcy)

		market := &types.Market{
			ID:     item.InstID, // OKX 使用 InstID 作为市场ID
			Symbol: normalizedSymbol,
			Base:   item.BaseCcy,
			Quote:  item.QuoteCcy,
			Type:   types.MarketTypeSpot,
			Active: item.State == "live",
		}

		// 解析精度和限制
		if item.MinSz != "" {
			market.Limits.Amount.Min, _ = strconv.ParseFloat(item.MinSz, 64)
		}
		if item.MaxSz != "" {
			market.Limits.Amount.Max, _ = strconv.ParseFloat(item.MaxSz, 64)
		}
		if item.MinSzVal != "" {
			market.Limits.Cost.Min, _ = strconv.ParseFloat(item.MinSzVal, 64)
		}

		// 计算精度
		if item.LotSz != "" {
			parts := strings.Split(item.LotSz, ".")
			if len(parts) > 1 {
				market.Precision.Amount = len(strings.TrimRight(parts[1], "0"))
			}
		}
		if item.TickSz != "" {
			parts := strings.Split(item.TickSz, ".")
			if len(parts) > 1 {
				market.Precision.Price = len(strings.TrimRight(parts[1], "0"))
			}
		}

		markets = append(markets, market)
	}

	return markets, nil
}

func (m *okxSpotMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	markets := make([]*types.Market, 0, len(m.okx.spotMarkets))
	for _, market := range m.okx.spotMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *okxSpotMarket) GetMarket(symbol string) (*types.Market, error) {
	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	market, ok := m.okx.spotMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

func (m *okxSpotMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
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

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID    string `json:"instId"`
			Last      string `json:"last"`
			LastSz    string `json:"lastSz"`
			AskPx     string `json:"askPx"`
			AskSz     string `json:"askSz"`
			BidPx     string `json:"bidPx"`
			BidSz     string `json:"bidSz"`
			Open24h   string `json:"open24h"`
			High24h   string `json:"high24h"`
			Low24h    string `json:"low24h"`
			Vol24h    string `json:"vol24h"`
			VolCcy24h string `json:"volCcy24h"`
			Ts        string `json:"ts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ticker: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	data := result.Data[0]
	ticker := &types.Ticker{
		Symbol:    symbol,
		Timestamp: time.Now(),
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

func (m *okxSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	resp, err := m.okx.client.HTTPClient.Get(ctx, "/api/v5/market/tickers", map[string]interface{}{
		"instType": "SPOT",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID    string `json:"instId"`
			Last      string `json:"last"`
			AskPx     string `json:"askPx"`
			BidPx     string `json:"bidPx"`
			Open24h   string `json:"open24h"`
			High24h   string `json:"high24h"`
			Low24h    string `json:"low24h"`
			Vol24h    string `json:"vol24h"`
			VolCcy24h string `json:"volCcy24h"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	// 如果需要过滤特定 symbols，先转换为 OKX 格式
	var okxSymbols map[string]string
	if len(symbols) > 0 {
		okxSymbols = make(map[string]string)
		for _, s := range symbols {
			market, err := m.GetMarket(s)
			if err == nil {
				okxSymbols[market.ID] = s
			} else {
				// 如果市场未加载，尝试转换
				okxSymbol, err := ToOKXSymbol(s, false)
				if err == nil {
					okxSymbols[okxSymbol] = s
				}
			}
		}
	}

	tickers := make(map[string]*types.Ticker)
	for _, item := range result.Data {
		// 如果指定了 symbols，进行过滤
		if len(symbols) > 0 {
			normalizedSymbol, ok := okxSymbols[item.InstID]
			if !ok {
				continue
			}
			ticker := &types.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: time.Now(),
			}
			ticker.Bid = item.BidPx
			ticker.Ask = item.AskPx
			ticker.Last = item.Last
			ticker.Open = item.Open24h
			ticker.High = item.High24h
			ticker.Low = item.Low24h
			ticker.Volume = item.Vol24h
			ticker.QuoteVolume = item.VolCcy24h
			tickers[normalizedSymbol] = ticker
		} else {
			// 如果没有指定 symbols，尝试从市场信息中查找
			market, err := m.getMarketByID(item.InstID)
			if err != nil {
				continue
			}
			ticker := &types.Ticker{
				Symbol:    market.Symbol,
				Timestamp: time.Now(),
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
	}

	return tickers, nil
}

// getMarketByID 通过交易所ID获取市场信息
func (m *okxSpotMarket) getMarketByID(id string) (*types.Market, error) {
	m.okx.mu.RLock()
	defer m.okx.mu.RUnlock()

	for _, market := range m.okx.spotMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (m *okxSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
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

	var result struct {
		Code string     `json:"code"`
		Msg  string     `json:"msg"`
		Data [][]string `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	ohlcvs := make(types.OHLCVs, 0, len(result.Data))
	for _, item := range result.Data {
		if len(item) < 6 {
			continue
		}

		ohlcv := types.OHLCV{}
		ts, _ := strconv.ParseInt(item[0], 10, 64)
		ohlcv.Timestamp = time.UnixMilli(ts)
		ohlcv.Open, _ = strconv.ParseFloat(item[1], 64)
		ohlcv.High, _ = strconv.ParseFloat(item[2], 64)
		ohlcv.Low, _ = strconv.ParseFloat(item[3], 64)
		ohlcv.Close, _ = strconv.ParseFloat(item[4], 64)
		ohlcv.Volume, _ = strconv.ParseFloat(item[5], 64)

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

func (o *okxSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/account/balance", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Details []struct {
				Ccy       string `json:"ccy"`
				AvailBal  string `json:"availBal"`
				FrozenBal string `json:"frozenBal"`
				Eq        string `json:"eq"`
			} `json:"details"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	balances := make(types.Balances)
	for _, detail := range result.Data[0].Details {
		free, _ := strconv.ParseFloat(detail.AvailBal, 64)
		used, _ := strconv.ParseFloat(detail.FrozenBal, 64)
		total, _ := strconv.ParseFloat(detail.Eq, 64)

		balances[detail.Ccy] = &types.Balance{
			Currency:  detail.Ccy,
			Free:      free,
			Used:      used,
			Total:     total,
			Available: free,
		}
	}

	return balances, nil
}

func (o *okxSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
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
	if orderType == types.OrderTypeLimit {
		reqBody["px"] = priceStr
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

func (o *okxSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// 获取市场信息
	market, err := o.okx.spot.market.GetMarket(symbol)
	if err != nil {
		return err
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
		"ordId":  orderID,
	}

	_, err = o.signAndRequest(ctx, "POST", "/api/v5/trade/cancel-order", nil, reqBody)
	return err
}

// parseOrder 解析订单数据
func (o *okxSpotOrder) parseOrder(item struct {
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

func (o *okxSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
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

	params := map[string]interface{}{
		"instId": okxSymbol,
		"ordId":  orderID,
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

func (o *okxSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// OKX 现货 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: OKX spot API does not support fetching order history directly")
}

func (o *okxSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	params := map[string]interface{}{
		"instType": "SPOT",
	}
	if symbol != "" {
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
		params["instId"] = okxSymbol
	}

	resp, err := o.signAndRequest(ctx, "GET", "/api/v5/trade/orders-pending", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
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
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	orders := make([]*types.Order, 0, len(result.Data))
	for _, item := range result.Data {
		normalizedSymbol := symbol
		if symbol == "" {
			// 如果没有提供symbol，尝试从市场信息中查找
			market, err := o.okx.spot.market.getMarketByID(item.InstID)
			if err == nil {
				normalizedSymbol = market.Symbol
			} else {
				normalizedSymbol = item.InstID // 临时使用原格式
			}
		}
		// 创建临时结构体以匹配 parseOrder 的签名
		orderItem := struct {
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
		}{
			InstID:    item.InstID,
			OrdID:     item.OrdID,
			ClOrdID:   item.ClOrdID,
			State:     item.State,
			Side:      item.Side,
			OrdType:   item.OrdType,
			Px:        item.Px,
			Sz:        item.Sz,
			AccFillSz: item.AccFillSz,
			UTime:     "",
		}
		orders = append(orders, o.parseOrder(orderItem, normalizedSymbol))
	}

	return orders, nil
}

func (o *okxSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
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

func (o *okxSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
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

	params := map[string]interface{}{
		"instId": okxSymbol,
		"limit":  limit,
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

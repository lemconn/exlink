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
	"github.com/lemconn/exlink/types"
	"github.com/shopspring/decimal"
)

// BybitSpot Bybit 现货实现
type BybitSpot struct {
	bybit  *Bybit
	market *bybitSpotMarket
	order  *bybitSpotOrder
}

// NewBybitSpot 创建 Bybit 现货实例
func NewBybitSpot(b *Bybit) *BybitSpot {
	return &BybitSpot{
		bybit:  b,
		market: &bybitSpotMarket{bybit: b},
		order:  &bybitSpotOrder{bybit: b},
	}
}

// ========== SpotExchange 接口实现 ==========

func (s *BybitSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

func (s *BybitSpot) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *BybitSpot) GetMarket(symbol string) (*model.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *BybitSpot) GetMarkets() ([]*model.Market, error) {
	return s.market.GetMarkets()
}

func (s *BybitSpot) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *BybitSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

func (s *BybitSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (s *BybitSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *BybitSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *BybitSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *BybitSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

func (s *BybitSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

func (s *BybitSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

func (s *BybitSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

func (s *BybitSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

var _ exchange.SpotExchange = (*BybitSpot)(nil)

// ========== 内部实现 ==========

type bybitSpotMarket struct {
	bybit *Bybit
}

func (m *bybitSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	m.bybit.mu.RLock()
	if !reload && len(m.bybit.spotMarkets) > 0 {
		m.bybit.mu.RUnlock()
		return nil
	}
	m.bybit.mu.RUnlock()

	// 获取现货市场信息
	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/instruments-info", map[string]interface{}{
		"category": "spot",
	})
	if err != nil {
		return fmt.Errorf("fetch spot markets: %w", err)
	}

	var result bybitSpotMarketsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("unmarshal spot markets: %w", err)
	}

	if result.RetCode != 0 {
		return fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	markets := make([]*model.Market, 0)
	for _, s := range result.Result.List {
		if s.Status != "Trading" {
			continue
		}

		// 转换为标准化格式 BTC/USDT
		normalizedSymbol := common.NormalizeSymbol(s.BaseCoin, s.QuoteCoin)

		market := &model.Market{
			ID:     s.Symbol,         // Bybit 原始格式 (BTCUSDT)
			Symbol: normalizedSymbol, // 标准化格式 (BTC/USDT)
			Base:   s.BaseCoin,
			Quote:  s.QuoteCoin,
			Type:   model.MarketTypeSpot,
			Active: s.Status == "Trading",
		}

		// 解析精度
		basePrecision := s.LotSizeFilter.BasePrecision.InexactFloat64()
		tickSize := s.PriceFilter.TickSize.InexactFloat64()
		quotePrecision := s.LotSizeFilter.QuotePrecision.InexactFloat64()

		// 计算精度位数
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
	m.bybit.mu.Lock()
	if m.bybit.spotMarkets == nil {
		m.bybit.spotMarkets = make(map[string]*model.Market)
	}
	for _, market := range markets {
		m.bybit.spotMarkets[market.Symbol] = market
	}
	m.bybit.mu.Unlock()

	return nil
}

func (m *bybitSpotMarket) FetchMarkets(ctx context.Context) ([]*model.Market, error) {
	// 确保市场已加载
	if err := m.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.bybit.spotMarkets))
	for _, market := range m.bybit.spotMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *bybitSpotMarket) GetMarket(symbol string) (*model.Market, error) {
	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	market, ok := m.bybit.spotMarkets[symbol]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", symbol)
	}

	return market, nil
}

func (m *bybitSpotMarket) GetMarkets() ([]*model.Market, error) {
	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	markets := make([]*model.Market, 0, len(m.bybit.spotMarkets))
	for _, market := range m.bybit.spotMarkets {
		markets = append(markets, market)
	}

	return markets, nil
}

func (m *bybitSpotMarket) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"symbol":   bybitSymbol,
		"category": "spot",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var result bybitSpotTickerResponse

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

	return ticker, nil
}

func (m *bybitSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*model.Ticker, error) {
	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"category": "spot",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var result bybitSpotTickerResponse

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	// 如果需要过滤特定 symbols，先转换为 Bybit 格式
	var bybitSymbols map[string]string
	if len(symbols) > 0 {
		bybitSymbols = make(map[string]string)
		for _, s := range symbols {
			market, err := m.GetMarket(s)
			if err == nil {
				bybitSymbols[market.ID] = s
			} else {
				// 如果市场未加载，尝试转换
				bybitSymbol, err := ToBybitSymbol(s, false)
				if err == nil {
					bybitSymbols[bybitSymbol] = s
				}
			}
		}
	}

	tickers := make(map[string]*model.Ticker)
	for _, item := range result.Result.List {
		// 如果指定了 symbols，进行过滤
		if len(symbols) > 0 {
			normalizedSymbol, ok := bybitSymbols[item.Symbol]
			if !ok {
				continue
			}
			ticker := &model.Ticker{
				Symbol:    normalizedSymbol,
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
			tickers[normalizedSymbol] = ticker
		} else {
			// 如果没有指定 symbols，尝试从市场信息中查找
			market, err := m.getMarketByID(item.Symbol)
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
			tickers[market.Symbol] = ticker
		}
	}

	return tickers, nil
}

// getMarketByID 通过交易所ID获取市场信息
func (m *bybitSpotMarket) getMarketByID(id string) (*model.Market, error) {
	m.bybit.mu.RLock()
	defer m.bybit.mu.RUnlock()

	for _, market := range m.bybit.spotMarkets {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found: %s", id)
}

func (m *bybitSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error) {
	// 获取市场信息
	market, err := m.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 标准化时间框架
	normalizedTimeframe := common.BybitTimeframe(timeframe)

	params := map[string]interface{}{
		"symbol":   market.ID,
		"category": "spot",
		"interval": normalizedTimeframe,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["start"] = since.UnixMilli()
	}

	resp, err := m.bybit.client.HTTPClient.Get(ctx, "/v5/market/kline", params)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var result bybitSpotKlineResponse
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

type bybitSpotOrder struct {
	bybit *Bybit
}

// signAndRequest 签名并发送请求（Bybit v5 API）
func (o *bybitSpotOrder) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if o.bybit.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	signature, timestamp := o.bybit.signer.SignRequest(method, params, body)
	recvWindow := "5000"

	// 设置请求头
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-API-KEY", o.bybit.client.APIKey)
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-TIMESTAMP", timestamp)
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-RECV-WINDOW", recvWindow)
	o.bybit.client.HTTPClient.SetHeader("X-BAPI-SIGN", signature)
	o.bybit.client.HTTPClient.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return o.bybit.client.HTTPClient.Get(ctx, path, params)
	} else {
		return o.bybit.client.HTTPClient.Post(ctx, path, body)
	}
}

func (o *bybitSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	// Bybit v5 统一账户余额
	resp, err := o.signAndRequest(ctx, "GET", "/v5/account/wallet-balance", map[string]interface{}{
		"accountType": "SPOT",
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Coin []struct {
					Coin                string `json:"coin"`
					Equity              string `json:"equity"`
					AvailableToWithdraw string `json:"availableToWithdraw"`
					WalletBalance       string `json:"walletBalance"`
				} `json:"coin"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal balance: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	balances := make(types.Balances)
	if len(result.Result.List) > 0 {
		for _, coin := range result.Result.List[0].Coin {
			equity, _ := strconv.ParseFloat(coin.Equity, 64)
			available, _ := strconv.ParseFloat(coin.AvailableToWithdraw, 64)
			walletBalance, _ := strconv.ParseFloat(coin.WalletBalance, 64)

			balances[coin.Coin] = &types.Balance{
				Currency:  coin.Coin,
				Total:     equity,
				Free:      available,
				Used:      walletBalance - available,
				Available: available,
			}
		}
	}

	return balances, nil
}

func (o *bybitSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
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

	market, err := o.bybit.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	// Bybit API requires "Buy" or "Sell" (capitalized)
	sideStr := string(side)
	if len(sideStr) > 0 {
		sideStr = strings.ToUpper(sideStr[:1]) + strings.ToLower(sideStr[1:])
	}

	reqBody := map[string]interface{}{
		"category": "spot",
		"symbol":   bybitSymbol,
		"side":     sideStr,
	}

	// 现货市价买单特殊处理
	if orderType == types.OrderTypeMarket && side == types.OrderSideBuy {
		// Calculate cost: amount * price (use current ask price if price not provided)
		amountDecimal, err := decimal.NewFromString(amount)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %w", err)
		}

		var costDecimal decimal.Decimal
		if priceStr != "" {
			priceDecimal, err := decimal.NewFromString(priceStr)
			if err != nil {
				return nil, fmt.Errorf("invalid price: %w", err)
			}
			costDecimal = amountDecimal.Mul(priceDecimal)
		} else {
			// Fetch current price to calculate cost
			ticker, err := o.bybit.spot.market.FetchTicker(ctx, symbol)
			if err == nil && !ticker.Ask.IsZero() {
				costDecimal = amountDecimal.Mul(ticker.Ask.Decimal)
			} else {
				costDecimal = amountDecimal
			}
		}

		reqBody["marketUnit"] = "quoteCoin"
		precision := market.Precision.Price
		if precision <= 0 {
			precision = 8
		}
		reqBody["qty"] = costDecimal.StringFixed(int32(precision))
		reqBody["orderType"] = "Market"
	} else {
		// 其他订单类型
		precision := market.Precision.Amount
		if precision <= 0 {
			precision = 8
		}
		amountFloat, err := strconv.ParseFloat(amount, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %w", err)
		}
		reqBody["qty"] = strconv.FormatFloat(amountFloat, 'f', precision, 64)

		if orderType == types.OrderTypeLimit {
			reqBody["orderType"] = "Limit"
			priceFloat, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid price: %w", err)
			}
			pricePrecision := market.Precision.Price
			if pricePrecision <= 0 {
				pricePrecision = 8
			}
			reqBody["price"] = strconv.FormatFloat(priceFloat, 'f', pricePrecision, 64)

			// 处理 timeInForce
			if options.TimeInForce != nil {
				reqBody["timeInForce"] = strings.ToUpper(string(*options.TimeInForce))
			} else {
				reqBody["timeInForce"] = "GTC"
			}
		} else {
			reqBody["orderType"] = "Market"
		}
	}

	// 客户端订单ID
	if options.ClientOrderID != nil && *options.ClientOrderID != "" {
		reqBody["orderLinkId"] = *options.ClientOrderID
	} else {
		reqBody["orderLinkId"] = common.GenerateClientOrderID(o.bybit.Name(), side)
	}

	resp, err := o.signAndRequest(ctx, "POST", "/v5/order/create", nil, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			OrderID     string `json:"orderId"`
			OrderLinkID string `json:"orderLinkId"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	amountFloat, _ := strconv.ParseFloat(amount, 64)
	var priceFloat float64
	if priceStr != "" {
		priceFloat, _ = strconv.ParseFloat(priceStr, 64)
	}

	order := &types.Order{
		ID:            result.Result.OrderID,
		ClientOrderID: result.Result.OrderLinkID,
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

func (o *bybitSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// 获取市场信息
	market, err := o.bybit.spot.market.GetMarket(symbol)
	if err != nil {
		return err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, false)
		if err != nil {
			return fmt.Errorf("get market ID: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"category": "spot",
		"symbol":   bybitSymbol,
		"orderId":  orderID,
	}

	_, err = o.signAndRequest(ctx, "POST", "/v5/order/cancel", nil, reqBody)
	return err
}

// parseOrder 解析订单数据
func (o *bybitSpotOrder) parseOrder(item struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	OrderStatus string `json:"orderStatus"`
	Side        string `json:"side"`
	OrderType   string `json:"orderType"`
	Price       string `json:"price"`
	Qty         string `json:"qty"`
	CumExecQty  string `json:"cumExecQty"`
	CreatedTime string `json:"createdTime"`
}, symbol string) *types.Order {
	order := &types.Order{
		ID:            item.OrderID,
		ClientOrderID: item.OrderLinkID,
		Symbol:        symbol,
		Timestamp:     time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(item.Price, 64)
	order.Amount, _ = strconv.ParseFloat(item.Qty, 64)
	order.Filled, _ = strconv.ParseFloat(item.CumExecQty, 64)
	order.Remaining = order.Amount - order.Filled

	if strings.ToUpper(item.Side) == "BUY" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToUpper(item.OrderType) == "MARKET" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	switch item.OrderStatus {
	case "New":
		order.Status = types.OrderStatusNew
	case "PartiallyFilled":
		order.Status = types.OrderStatusPartiallyFilled
	case "Filled":
		order.Status = types.OrderStatusFilled
	case "Cancelled":
		order.Status = types.OrderStatusCanceled
	default:
		order.Status = types.OrderStatusNew
	}

	return order
}

func (o *bybitSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// 获取市场信息
	market, err := o.bybit.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"category": "spot",
		"symbol":   bybitSymbol,
		"orderId":  orderID,
	}

	// First try to fetch from open orders (realtime)
	resp, err := o.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
	if err == nil {
		var realtimeResult struct {
			RetCode int    `json:"retCode"`
			RetMsg  string `json:"retMsg"`
			Result  struct {
				List []struct {
					OrderID     string `json:"orderId"`
					OrderLinkID string `json:"orderLinkId"`
					OrderStatus string `json:"orderStatus"`
					Side        string `json:"side"`
					OrderType   string `json:"orderType"`
					Price       string `json:"price"`
					Qty         string `json:"qty"`
					CumExecQty  string `json:"cumExecQty"`
					CreatedTime string `json:"createdTime"`
				} `json:"list"`
			} `json:"result"`
		}

		if err := json.Unmarshal(resp, &realtimeResult); err == nil && realtimeResult.RetCode == 0 {
			for _, item := range realtimeResult.Result.List {
				if item.OrderID == orderID {
					return o.parseOrder(item, symbol), nil
				}
			}
		}
	}

	// If not found in open orders, try history
	resp, err = o.signAndRequest(ctx, "GET", "/v5/order/history", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OrderID     string `json:"orderId"`
				OrderLinkID string `json:"orderLinkId"`
				OrderStatus string `json:"orderStatus"`
				Side        string `json:"side"`
				OrderType   string `json:"orderType"`
				Price       string `json:"price"`
				Qty         string `json:"qty"`
				CumExecQty  string `json:"cumExecQty"`
				CreatedTime string `json:"createdTime"`
			} `json:"list"`
		} `json:"result"`
	}

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
		if item.OrderID == orderID {
			return o.parseOrder(item, symbol), nil
		}
	}

	return nil, fmt.Errorf("order not found")
}

func (o *bybitSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// Bybit 现货 API 不支持直接查询历史订单列表
	// 可以通过 FetchOpenOrders 获取未成交订单
	// 历史订单需要通过其他方式获取（如通过订单ID逐个查询）
	return nil, fmt.Errorf("not implemented: Bybit spot API does not support fetching order history directly")
}

func (o *bybitSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	params := map[string]interface{}{
		"category": "spot",
	}
	if symbol != "" {
		// 获取市场信息
		market, err := o.bybit.spot.market.GetMarket(symbol)
		if err != nil {
			return nil, err
		}

		// 获取交易所格式的 symbol ID
		bybitSymbol := market.ID
		if bybitSymbol == "" {
			var err error
			bybitSymbol, err = ToBybitSymbol(symbol, false)
			if err != nil {
				return nil, fmt.Errorf("get market ID: %w", err)
			}
		}
		params["symbol"] = bybitSymbol
	}

	resp, err := o.signAndRequest(ctx, "GET", "/v5/order/realtime", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch open orders: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OrderID     string `json:"orderId"`
				OrderLinkID string `json:"orderLinkId"`
				OrderStatus string `json:"orderStatus"`
				Side        string `json:"side"`
				OrderType   string `json:"orderType"`
				Price       string `json:"price"`
				Qty         string `json:"qty"`
				CumExecQty  string `json:"cumExecQty"`
				CreatedTime string `json:"createdTime"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal orders: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	orders := make([]*types.Order, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		normalizedSymbol := symbol
		if symbol == "" {
			// 如果没有提供symbol，尝试从市场信息中查找
			market, err := o.bybit.spot.market.getMarketByID(item.OrderID)
			if err == nil {
				normalizedSymbol = market.Symbol
			} else {
				normalizedSymbol = item.OrderID // 临时使用原格式
			}
		}
		orders = append(orders, o.parseOrder(item, normalizedSymbol))
	}

	return orders, nil
}

func (o *bybitSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.bybit.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"category": "spot",
		"symbol":   bybitSymbol,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := o.bybit.client.HTTPClient.Get(ctx, "/v5/market/recent-trade", params)
	if err != nil {
		return nil, fmt.Errorf("fetch trades: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				ExecTime  string `json:"execTime"`
				ExecPrice string `json:"execPrice"`
				ExecQty   string `json:"execQty"`
				Side      string `json:"side"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	trades := make([]*types.Trade, 0, len(result.Result.List))
	for i, item := range result.Result.List {
		price, _ := strconv.ParseFloat(item.ExecPrice, 64)
		qty, _ := strconv.ParseFloat(item.ExecQty, 64)
		execTime, _ := strconv.ParseInt(item.ExecTime, 10, 64)

		trade := &types.Trade{
			ID:        strconv.FormatInt(execTime, 10) + "-" + strconv.Itoa(i),
			Symbol:    symbol,
			Price:     price,
			Amount:    qty,
			Cost:      price * qty,
			Timestamp: time.UnixMilli(execTime),
		}

		if strings.ToUpper(item.Side) == "BUY" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func (o *bybitSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取市场信息
	market, err := o.bybit.spot.market.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, false)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	params := map[string]interface{}{
		"category": "spot",
		"symbol":   bybitSymbol,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["startTime"] = since.UnixMilli()
	}

	resp, err := o.signAndRequest(ctx, "GET", "/v5/execution/list", params, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch my trades: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				ExecID      string `json:"execId"`
				OrderID     string `json:"orderId"`
				ExecPrice   string `json:"execPrice"`
				ExecQty     string `json:"execQty"`
				ExecTime    string `json:"execTime"`
				Side        string `json:"side"`
				Fee         string `json:"execFee"`
				FeeCurrency string `json:"feeCurrencyId"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal trades: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	trades := make([]*types.Trade, 0, len(result.Result.List))
	for _, item := range result.Result.List {
		price, _ := strconv.ParseFloat(item.ExecPrice, 64)
		qty, _ := strconv.ParseFloat(item.ExecQty, 64)
		execTime, _ := strconv.ParseInt(item.ExecTime, 10, 64)
		fee, _ := strconv.ParseFloat(item.Fee, 64)

		trade := &types.Trade{
			ID:        item.ExecID,
			OrderID:   item.OrderID,
			Symbol:    symbol,
			Price:     price,
			Amount:    qty,
			Cost:      price * qty,
			Timestamp: time.UnixMilli(execTime),
		}

		if strings.ToUpper(item.Side) == "BUY" {
			trade.Side = "buy"
		} else {
			trade.Side = "sell"
		}

		if fee > 0 && item.FeeCurrency != "" {
			trade.Fee = &types.Fee{
				Currency: item.FeeCurrency,
				Cost:     fee,
			}
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

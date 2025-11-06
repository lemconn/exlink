package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/common"
	"github.com/lemconn/exlink/types"
)

const (
	okxName       = "okx"
	okxBaseURL    = "https://www.okx.com"
	okxSandboxURL = "https://www.okx.com" // OKX使用同一个域名，通过header区分
)

// OKX OKX交易所实现
type OKX struct {
	*exlink.BaseExchange
	client     *common.HTTPClient
	apiKey     string
	secretKey  string
	passphrase string
}

// NewOKX 创建OKX交易所实例
func NewOKX(apiKey, secretKey string, options map[string]interface{}) (exlink.Exchange, error) {
	baseURL := okxBaseURL
	sandbox := false
	proxyURL := ""
	passphrase := ""

	if v, ok := options["baseURL"].(string); ok {
		baseURL = v
	}
	if v, ok := options["sandbox"].(bool); ok {
		sandbox = v
	}
	if v, ok := options["proxy"].(string); ok {
		proxyURL = v
	}
	if v, ok := options["passphrase"].(string); ok {
		passphrase = v
	}

	exchange := &OKX{
		BaseExchange: exlink.NewBaseExchange(okxName),
		client:       common.NewHTTPClient(baseURL),
		apiKey:       apiKey,
		secretKey:    secretKey,
		passphrase:   passphrase,
	}

	// 设置模拟盘模式
	exchange.SetSandbox(sandbox)

	// 设置代理
	if proxyURL != "" {
		exchange.SetProxy(proxyURL)
		if err := exchange.client.SetProxy(proxyURL); err != nil {
			return nil, fmt.Errorf("set proxy: %w", err)
		}
	}

	// 设置请求头
	if apiKey != "" {
		exchange.client.SetHeader("OK-ACCESS-KEY", apiKey)
		if sandbox {
			exchange.client.SetHeader("x-simulated-trading", "1")
		}
	}

	// 设置其他选项
	for k, v := range options {
		if k != "baseURL" && k != "sandbox" && k != "proxy" && k != "passphrase" {
			exchange.SetOption(k, v)
		}
	}

	return exchange, nil
}

// LoadMarkets 加载市场信息
func (o *OKX) LoadMarkets(ctx context.Context, reload bool) error {
	markets := make([]*types.Market, 0)

	// 获取要加载的市场类型
	fetchMarketsTypes := []string{"spot"}
	if v, ok := o.GetOption("fetchMarkets").([]string); ok && len(v) > 0 {
		fetchMarketsTypes = v
	} else if v, ok := o.GetOption("fetchMarkets").(string); ok {
		fetchMarketsTypes = []string{v}
	}

	// 加载现货市场
	if contains(fetchMarketsTypes, "spot") {
		spotMarkets, err := o.loadSpotMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load spot markets: %w", err)
		}
		markets = append(markets, spotMarkets...)
	}

	// 加载永续合约市场
	if contains(fetchMarketsTypes, "swap") {
		swapMarkets, err := o.loadSwapMarkets(ctx)
		if err != nil {
			return fmt.Errorf("load swap markets: %w", err)
		}
		markets = append(markets, swapMarkets...)
	}

	o.SetMarkets(markets)
	return nil
}

// loadSpotMarkets 加载现货市场
func (o *OKX) loadSpotMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := o.client.Get(ctx, "/api/v5/public/instruments", map[string]interface{}{
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

// loadSwapMarkets 加载永续合约市场
func (o *OKX) loadSwapMarkets(ctx context.Context) ([]*types.Market, error) {
	resp, err := o.client.Get(ctx, "/api/v5/public/instruments", map[string]interface{}{
		"instType": "SWAP",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch swap instruments: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstType  string `json:"instType"`
			InstID    string `json:"instId"`
			BaseCcy   string `json:"baseCcy"`
			QuoteCcy  string `json:"quoteCcy"`
			SettleCcy string `json:"settleCcy"`
			Uly       string `json:"uly"`    // underlying，用于合约市场
			CtType    string `json:"ctType"` // linear, inverse
			State     string `json:"state"`
			MinSz     string `json:"minSz"`
			MaxSz     string `json:"maxSz"`
			LotSz     string `json:"lotSz"`
			TickSz    string `json:"tickSz"`
			MinSzVal  string `json:"minSzVal"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal swap instruments: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	markets := make([]*types.Market, 0)
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
		// 如果 settle 不为空，则 symbol = base/quote:settle
		settle := item.SettleCcy
		if settle == "" {
			// 根据 ctType 判断：linear (U本位) settle=quote, inverse (币本位) settle=base
			if item.CtType == "linear" {
				settle = quoteCcy
			} else if item.CtType == "inverse" {
				settle = baseCcy
			} else {
				// 默认 U 本位
				settle = quoteCcy
			}
		}

		// 转换为标准化格式 BTC/USDT:USDT
		// 对于合约市场，如果 settle 不为空，则 symbol = base/quote:settle
		normalizedSymbol := common.NormalizeContractSymbol(baseCcy, quoteCcy, settle)

		market := &types.Market{
			ID:       item.InstID,
			Symbol:   normalizedSymbol,
			Base:     baseCcy,
			Quote:    quoteCcy,
			Settle:   settle,
			Type:     types.MarketTypeSwap,
			Active:   item.State == "live",
			Contract: true,
			Linear:   item.CtType == "linear",  // U本位
			Inverse:  item.CtType == "inverse", // 币本位
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

// contains 检查字符串切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// FetchTicker 获取行情
func (o *OKX) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// 获取市场信息
	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	resp, err := o.client.Get(ctx, "/api/v5/market/ticker", map[string]interface{}{
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

	ticker.Bid, _ = strconv.ParseFloat(data.BidPx, 64)
	ticker.Ask, _ = strconv.ParseFloat(data.AskPx, 64)
	ticker.Last, _ = strconv.ParseFloat(data.Last, 64)
	ticker.Open, _ = strconv.ParseFloat(data.Open24h, 64)
	ticker.High, _ = strconv.ParseFloat(data.High24h, 64)
	ticker.Low, _ = strconv.ParseFloat(data.Low24h, 64)
	ticker.Volume, _ = strconv.ParseFloat(data.Vol24h, 64)
	ticker.QuoteVolume, _ = strconv.ParseFloat(data.VolCcy24h, 64)

	if ticker.Open > 0 {
		ticker.Change = ticker.Last - ticker.Open
		ticker.ChangePercent = (ticker.Change / ticker.Open) * 100
	}

	return ticker, nil
}

// FetchTickers 批量获取行情
func (o *OKX) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// 根据symbols判断instType，如果未指定则获取所有类型
	instTypes := []string{"SPOT"}
	if len(symbols) > 0 {
		// 检查symbols中是否有合约市场
		hasContract := false
		for _, s := range symbols {
			market, err := o.GetMarket(s)
			if err == nil && market.Contract {
				hasContract = true
				break
			}
		}
		if hasContract {
			instTypes = []string{"SPOT", "SWAP"}
		}
	} else {
		// 未指定symbols，获取所有类型
		instTypes = []string{"SPOT", "SWAP"}
	}

	tickers := make(map[string]*types.Ticker)
	for _, instType := range instTypes {
		resp, err := o.client.Get(ctx, "/api/v5/market/tickers", map[string]interface{}{
			"instType": instType,
		})
		if err != nil {
			continue // 跳过错误，继续处理其他类型
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
			continue
		}

		if result.Code != "0" {
			continue
		}

		for _, item := range result.Data {
			// 获取市场信息以确定标准化symbol
			market, err := o.GetMarketByID(item.InstID)
			if err != nil {
				// 如果市场未加载，跳过（要求先加载市场）
				continue
			}
			normalizedSymbol := market.Symbol

			if len(symbols) > 0 {
				found := false
				for _, s := range symbols {
					if s == normalizedSymbol {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			ticker := &types.Ticker{
				Symbol:    normalizedSymbol,
				Timestamp: time.Now(),
			}

			ticker.Bid, _ = strconv.ParseFloat(item.BidPx, 64)
			ticker.Ask, _ = strconv.ParseFloat(item.AskPx, 64)
			ticker.Last, _ = strconv.ParseFloat(item.Last, 64)
			ticker.Open, _ = strconv.ParseFloat(item.Open24h, 64)
			ticker.High, _ = strconv.ParseFloat(item.High24h, 64)
			ticker.Low, _ = strconv.ParseFloat(item.Low24h, 64)
			ticker.Volume, _ = strconv.ParseFloat(item.Vol24h, 64)
			ticker.QuoteVolume, _ = strconv.ParseFloat(item.VolCcy24h, 64)

			if ticker.Open > 0 {
				ticker.Change = ticker.Last - ticker.Open
				ticker.ChangePercent = (ticker.Change / ticker.Open) * 100
			}

			tickers[normalizedSymbol] = ticker
		}
	}

	return tickers, nil
}

// FetchOHLCV 获取K线数据
func (o *OKX) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// 获取市场信息
	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	// 标准化时间框架
	normalizedTimeframe := common.OKXTimeframe(timeframe)

	params := map[string]interface{}{
		"instId": okxSymbol,
		"bar":    normalizedTimeframe,
		"limit":  limit,
	}
	if !since.IsZero() {
		params["after"] = since.UnixMilli()
	}

	resp, err := o.client.Get(ctx, "/api/v5/market/candles", params)
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
		if ts, err := strconv.ParseInt(item[0], 10, 64); err == nil {
			ohlcv.Timestamp = time.UnixMilli(ts)
		}
		ohlcv.Open, _ = strconv.ParseFloat(item[1], 64)
		ohlcv.High, _ = strconv.ParseFloat(item[2], 64)
		ohlcv.Low, _ = strconv.ParseFloat(item[3], 64)
		ohlcv.Close, _ = strconv.ParseFloat(item[4], 64)
		ohlcv.Volume, _ = strconv.ParseFloat(item[5], 64)

		ohlcvs = append(ohlcvs, ohlcv)
	}

	return ohlcvs, nil
}

// FetchBalance 获取余额
func (o *OKX) FetchBalance(ctx context.Context) (types.Balances, error) {
	if o.secretKey == "" {
		return nil, exlink.ErrAuthenticationRequired
	}

	// OKX需要签名
	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("GET", "/api/v5/account/balance", timestamp, "")

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	resp, err := o.client.Get(ctx, "/api/v5/account/balance", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Details []struct {
				Ccy    string `json:"ccy"`
				Avail  string `json:"avail"`
				Frozen string `json:"frozen"`
				Eq     string `json:"eq"`
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
		free, _ := strconv.ParseFloat(detail.Avail, 64)
		used, _ := strconv.ParseFloat(detail.Frozen, 64)
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

// CreateOrder 创建订单
func (o *OKX) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price float64, params map[string]interface{}) (*types.Order, error) {
	if o.secretKey == "" {
		return nil, exlink.ErrAuthenticationRequired
	}

	// 获取市场信息
	market, err := o.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	// 确定交易模式
	tdMode := "cash" // 现货
	if market.Contract {
		tdMode = "cross" // 合约默认使用全仓模式
		if v, ok := params["tdMode"].(string); ok {
			tdMode = v
		}
	}

	reqBody := map[string]interface{}{
		"instId":  okxSymbol,
		"tdMode":  tdMode,
		"side":    strings.ToLower(string(side)),
		"ordType": strings.ToLower(string(orderType)),
		"sz":      strconv.FormatFloat(amount, 'f', -1, 64),
	}

	if orderType == types.OrderTypeLimit {
		reqBody["px"] = strconv.FormatFloat(price, 'f', -1, 64)
	}

	// 合并额外参数
	for k, v := range params {
		reqBody[k] = v
	}

	bodyStr, _ := json.Marshal(reqBody)
	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("POST", "/api/v5/trade/order", timestamp, string(bodyStr))

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	resp, err := o.client.Post(ctx, "/api/v5/trade/order", reqBody)
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

	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	data := result.Data[0]
	order := &types.Order{
		ID:            data.OrdID,
		ClientOrderID: data.ClOrdID,
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Amount:        amount,
		Price:         price,
		Timestamp:     time.Now(),
		Status:        types.OrderStatusNew,
	}

	return order, nil
}

// CancelOrder 取消订单
func (o *OKX) CancelOrder(ctx context.Context, orderID, symbol string) error {
	if o.secretKey == "" {
		return exlink.ErrAuthenticationRequired
	}

	// 获取市场信息
	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return fmt.Errorf("get market ID: %w", err)
	}

	reqBody := map[string]interface{}{
		"instId": okxSymbol,
		"ordId":  orderID,
	}

	bodyStr, _ := json.Marshal(reqBody)
	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("POST", "/api/v5/trade/cancel-order", timestamp, string(bodyStr))

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	_, err = o.client.Post(ctx, "/api/v5/trade/cancel-order", reqBody)
	return err
}

// FetchOrder 查询订单
func (o *OKX) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	if o.secretKey == "" {
		return nil, exlink.ErrAuthenticationRequired
	}

	// 获取市场信息
	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("GET", "/api/v5/trade/order", timestamp, "")

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	resp, err := o.client.Get(ctx, "/api/v5/trade/order", map[string]interface{}{
		"instId": okxSymbol,
		"ordId":  orderID,
	})
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

	data := result.Data[0]
	order := &types.Order{
		ID:            data.OrdID,
		ClientOrderID: data.ClOrdID,
		Symbol:        symbol,
		Timestamp:     time.Now(),
	}

	order.Price, _ = strconv.ParseFloat(data.Px, 64)
	order.Amount, _ = strconv.ParseFloat(data.Sz, 64)
	order.Filled, _ = strconv.ParseFloat(data.AccFillSz, 64)
	order.Remaining = order.Amount - order.Filled

	if strings.ToLower(data.Side) == "buy" {
		order.Side = types.OrderSideBuy
	} else {
		order.Side = types.OrderSideSell
	}

	if strings.ToLower(data.OrdType) == "market" {
		order.Type = types.OrderTypeMarket
	} else {
		order.Type = types.OrderTypeLimit
	}

	// 转换状态
	switch data.State {
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

	return order, nil
}

// FetchOrders 查询订单列表
func (o *OKX) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

// FetchOpenOrders 查询未成交订单
func (o *OKX) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	if o.secretKey == "" {
		return nil, exlink.ErrAuthenticationRequired
	}

	params := map[string]interface{}{}
	if symbol != "" {
		// 获取市场信息
		market, err := o.GetMarket(symbol)
		if err != nil {
			return nil, err
		}

		// 确定instType
		instType := "SPOT"
		if market.Contract {
			instType = "SWAP"
		}
		params["instType"] = instType
		params["instId"], err = o.GetMarketID(symbol)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	} else {
		// 未指定symbol，获取所有类型
		params["instType"] = "SPOT,SWAP"
	}

	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("GET", "/api/v5/trade/orders-pending", timestamp, "")

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	resp, err := o.client.Get(ctx, "/api/v5/trade/orders-pending", params)
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
		// 获取市场信息以确定标准化symbol
		market, err := o.GetMarketByID(item.InstID)
		if err != nil {
			// 如果市场未加载，跳过（要求先加载市场）
			continue
		}
		normalizedSymbol := market.Symbol

		if symbol != "" && normalizedSymbol != symbol {
			continue
		}

		order := &types.Order{
			ID:            item.OrdID,
			ClientOrderID: item.ClOrdID,
			Symbol:        normalizedSymbol,
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

		order.Status = types.OrderStatusOpen

		orders = append(orders, order)
	}

	return orders, nil
}

// FetchTrades 获取交易记录
func (o *OKX) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	params := map[string]interface{}{
		"instId": okxSymbol,
		"limit":  limit,
	}
	if !since.IsZero() {
		params["after"] = since.UnixMilli()
	}

	resp, err := o.client.Get(ctx, "/api/v5/market/trades", params)
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
			Side:      strings.ToLower(item.Side),
			Timestamp: time.UnixMilli(ts),
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchMyTrades 获取我的交易记录
func (o *OKX) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	if o.secretKey == "" {
		return nil, exlink.ErrAuthenticationRequired
	}

	// 获取市场信息
	market, err := o.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	okxSymbol, err := o.GetMarketID(symbol)
	if err != nil {
		return nil, fmt.Errorf("get market ID: %w", err)
	}

	// 确定instType
	instType := "SPOT"
	if market.Contract {
		instType = "SWAP"
	}

	params := map[string]interface{}{
		"instType": instType,
		"instId":   okxSymbol,
		"limit":    limit,
	}
	if !since.IsZero() {
		params["after"] = since.UnixMilli()
	}

	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("GET", "/api/v5/trade/fills", timestamp, "")

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	resp, err := o.client.Get(ctx, "/api/v5/trade/fills", params)
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
		fee, _ := strconv.ParseFloat(item.Fee, 64)
		ts, _ := strconv.ParseInt(item.Ts, 10, 64)

		trade := &types.Trade{
			ID:        item.TradeID,
			OrderID:   item.OrdID,
			Symbol:    symbol,
			Price:     price,
			Amount:    sz,
			Cost:      price * sz,
			Side:      strings.ToLower(item.Side),
			Timestamp: time.UnixMilli(ts),
		}

		if fee > 0 {
			trade.Fee = &types.Fee{
				Currency: item.FeeCcy,
				Cost:     fee,
			}
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// FetchPositions 获取持仓（合约）
func (o *OKX) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	if o.secretKey == "" {
		return nil, exlink.ErrAuthenticationRequired
	}

	params := map[string]interface{}{}
	if len(symbols) > 0 {
		instIds := make([]string, 0, len(symbols))
		for _, s := range symbols {
			market, err := o.GetMarket(s)
			if err != nil {
				continue
			}
			instIds = append(instIds, market.ID)
		}
		if len(instIds) > 0 {
			params["instId"] = strings.Join(instIds, ",")
		}
	}

	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("GET", "/api/v5/account/positions", timestamp, "")

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	resp, err := o.client.Get(ctx, "/api/v5/account/positions", params)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID      string `json:"instId"`
			PosID       string `json:"posId"`
			PosSide     string `json:"posSide"` // net, long, short
			Pos         string `json:"pos"`
			AvgPx       string `json:"avgPx"`
			MarkPx      string `json:"markPx"`
			LiqPx       string `json:"liqPx"`
			Upl         string `json:"upl"`
			UplRatio    string `json:"uplRatio"`
			Lever       string `json:"lever"`
			Margin      string `json:"margin"`
			MgnRatio    string `json:"mgnRatio"`
			MMR         string `json:"mmr"`
			Liab        string `json:"liab"`
			Interest    string `json:"interest"`
			NotionalUsd string `json:"notionalUsd"`
			OptVal      string `json:"optVal"`
			Adl         string `json:"adl"`
			CCy         string `json:"ccy"`
			Last        string `json:"last"`
			CcyEq       string `json:"ccyEq"`
			Imr         string `json:"imr"`
			UTime       string `json:"uTime"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	positions := make([]*types.Position, 0)
	for _, item := range result.Data {
		pos, _ := strconv.ParseFloat(item.Pos, 64)
		if pos == 0 {
			continue // 跳过空仓
		}

		// 获取市场信息
		market, err := o.GetMarketByID(item.InstID)
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

		avgPx, _ := strconv.ParseFloat(item.AvgPx, 64)
		markPx, _ := strconv.ParseFloat(item.MarkPx, 64)
		liqPx, _ := strconv.ParseFloat(item.LiqPx, 64)
		upl, _ := strconv.ParseFloat(item.Upl, 64)
		leverage, _ := strconv.ParseFloat(item.Lever, 64)
		margin, _ := strconv.ParseFloat(item.Margin, 64)
		uTime, _ := strconv.ParseInt(item.UTime, 10, 64)

		var side types.PositionSide
		if item.PosSide == "long" || (item.PosSide == "net" && pos > 0) {
			side = types.PositionSideLong
		} else {
			side = types.PositionSideShort
			pos = -pos
		}

		position := &types.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           pos,
			EntryPrice:       avgPx,
			MarkPrice:        markPx,
			LiquidationPrice: liqPx,
			UnrealizedPnl:    upl,
			Leverage:         leverage,
			Margin:           margin,
			Timestamp:        time.UnixMilli(uTime),
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// GetMarketByID 通过交易所ID获取市场信息
func (o *OKX) GetMarketByID(id string) (*types.Market, error) {
	for _, market := range o.BaseExchange.GetMarketsMap() {
		if market.ID == id {
			return market, nil
		}
	}
	return nil, exlink.ErrMarketNotFound
}

// GetMarketID 获取OKX格式的 symbol ID
// 优先从已加载的市场中查找，如果未找到则使用后备转换函数
func (o *OKX) GetMarketID(symbol string) (string, error) {
	// 优先从已加载的市场中查找
	market, ok := o.BaseExchange.GetMarketsMap()[symbol]
	if ok {
		return market.ID, nil
	}

	// 如果市场未加载，使用后备转换函数
	return common.ToOKXSymbol(symbol)
}

// SetLeverage 设置杠杆
func (o *OKX) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if o.secretKey == "" {
		return exlink.ErrAuthenticationRequired
	}

	market, err := o.GetMarket(symbol)
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

	bodyStr, _ := json.Marshal(reqBody)
	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("POST", "/api/v5/account/set-leverage", timestamp, string(bodyStr))

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	_, err = o.client.Post(ctx, "/api/v5/account/set-leverage", reqBody)
	return err
}

// SetMarginMode 设置保证金模式
func (o *OKX) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	if o.secretKey == "" {
		return exlink.ErrAuthenticationRequired
	}

	market, err := o.GetMarket(symbol)
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

	bodyStr, _ := json.Marshal(reqBody)
	timestamp := common.GetISO8601Timestamp()
	signature := o.signRequest("POST", "/api/v5/account/set-margin-mode", timestamp, string(bodyStr))

	o.client.SetHeader("OK-ACCESS-SIGN", signature)
	o.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	o.client.SetHeader("OK-ACCESS-PASSPHRASE", o.passphrase)

	_, err = o.client.Post(ctx, "/api/v5/account/set-margin-mode", reqBody)
	return err
}

// GetMarkets 获取市场列表
func (o *OKX) GetMarkets(ctx context.Context, marketType types.MarketType) ([]*types.Market, error) {
	if err := o.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	markets := make([]*types.Market, 0)
	for _, market := range o.BaseExchange.GetMarketsMap() {
		if marketType == "" || market.Type == marketType {
			markets = append(markets, market)
		}
	}
	return markets, nil
}

// signRequest OKX签名
func (o *OKX) signRequest(method, path string, timestamp string, body string) string {
	message := timestamp + method + path + body
	return common.SignHMAC256Base64(message, o.secretKey)
}

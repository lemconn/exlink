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
	"github.com/lemconn/exlink/option"
	"github.com/lemconn/exlink/types"
	"github.com/shopspring/decimal"
)

// BybitPerp Bybit 永续合约实现
type BybitPerp struct {
	bybit *Bybit
}

// NewBybitPerp 创建 Bybit 永续合约实例
func NewBybitPerp(b *Bybit) *BybitPerp {
	return &BybitPerp{
		bybit: b,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *BybitPerp) LoadMarkets(ctx context.Context, reload bool) error {
	// 如果已加载且不需要重新加载，直接返回
	p.bybit.mu.RLock()
	if !reload && len(p.bybit.perpMarketsBySymbol) > 0 {
		p.bybit.mu.RUnlock()
		return nil
	}
	p.bybit.mu.RUnlock()

	// 获取永续合约市场信息
	req := types.NewExValues()
	req.SetQuery("category", "linear")
	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/instruments-info", req.ToQueryMap())
	if err != nil {
		return fmt.Errorf("fetch swap markets: %w", err)
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol        string `json:"symbol"`
				BaseCoin      string `json:"baseCoin"`
				QuoteCoin     string `json:"quoteCoin"`
				Status        string `json:"status"`
				ContractType  string `json:"contractType"`
				LotSizeFilter struct {
					BasePrecision  types.ExDecimal `json:"basePrecision"`
					QuotePrecision types.ExDecimal `json:"quotePrecision"`
					MinOrderQty    types.ExDecimal `json:"minOrderQty"`
					MaxOrderQty    types.ExDecimal `json:"maxOrderQty"`
					MinOrderAmt    types.ExDecimal `json:"minOrderAmt"`
					MaxOrderAmt    types.ExDecimal `json:"maxOrderAmt"`
				} `json:"lotSizeFilter"`
				PriceFilter struct {
					TickSize types.ExDecimal `json:"tickSize"`
				} `json:"priceFilter"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return fmt.Errorf("unmarshal swap markets: %w", err)
	}

	if respData.RetCode != 0 {
		return fmt.Errorf("bybit api error: %s", respData.RetMsg)
	}

	markets := make([]*model.Market, 0)
	for _, s := range respData.Result.List {
		if s.Status != "Trading" {
			continue
		}

		if s.ContractType != "LinearPerpetual" && s.ContractType != "InversePerpetual" {
			continue
		}

		// Bybit linear 合约的 settle 通常是 quoteCoin
		settle := s.QuoteCoin

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(s.BaseCoin, s.QuoteCoin, settle)

		market := &model.Market{
			ID:       s.Symbol,
			Symbol:   normalizedSymbol,
			Base:     s.BaseCoin,
			Quote:    s.QuoteCoin,
			Settle:   settle,
			Type:     model.MarketTypeSwap,
			Active:   s.Status == "Trading",
			Contract: true,
		}

		// U本位永续合约
		if s.ContractType == "LinearPerpetual" {
			market.Linear = true
		}

		// 币本位永续合约
		if s.ContractType == "InversePerpetual" {
			market.Inverse = true
		}

		// 解析精度
		basePrecision := s.LotSizeFilter.BasePrecision.InexactFloat64()
		tickSize := s.PriceFilter.TickSize.InexactFloat64()
		quotePrecision := s.LotSizeFilter.QuotePrecision.InexactFloat64()

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
	p.bybit.mu.Lock()
	if p.bybit.perpMarketsBySymbol == nil {
		p.bybit.perpMarketsBySymbol = make(map[string]*model.Market)
		p.bybit.perpMarketsByID = make(map[string]*model.Market)
	}
	for _, market := range markets {
		p.bybit.perpMarketsBySymbol[market.Symbol] = market
		p.bybit.perpMarketsByID[market.ID] = market
	}
	p.bybit.mu.Unlock()

	return nil
}

func (p *BybitPerp) FetchMarkets(ctx context.Context, opts ...option.ArgsOption) (model.Markets, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	p.bybit.mu.RLock()
	defer p.bybit.mu.RUnlock()

	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		return model.Markets{market}, nil
	}

	markets := make(model.Markets, 0, len(p.bybit.perpMarketsBySymbol))
	for _, market := range p.bybit.perpMarketsBySymbol {
		markets = append(markets, market)
	}

	return markets, nil
}

func (p *BybitPerp) GetMarket(symbol string) (*model.Market, error) {
	p.bybit.mu.RLock()
	defer p.bybit.mu.RUnlock()

	// 先尝试标准化格式
	if market, ok := p.bybit.perpMarketsBySymbol[symbol]; ok {
		return market, nil
	}
	// 再尝试原始格式
	if market, ok := p.bybit.perpMarketsByID[symbol]; ok {
		return market, nil
	}

	return nil, fmt.Errorf("market not found: %s", symbol)
}

func (p *BybitPerp) FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error) {
	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	// 获取交易所格式的 symbol ID
	bybitSymbol := market.ID
	if bybitSymbol == "" {
		var err error
		bybitSymbol, err = ToBybitSymbol(symbol, true)
		if err != nil {
			return nil, fmt.Errorf("get market ID: %w", err)
		}
	}

	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", map[string]interface{}{
		"symbol":   bybitSymbol,
		"category": "linear",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ticker: %w", err)
	}

	var result bybitPerpTickerResponse

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
	ticker.Timestamp = result.Time

	return ticker, nil
}

func (p *BybitPerp) FetchTickers(ctx context.Context, opts ...option.ArgsOption) (model.Tickers, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	var querySymbol string
	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		querySymbol = market.ID
	}

	req := types.NewExValues()
	req.SetQuery("category", "linear")
	if querySymbol != "" {
		req.SetQuery("symbol", querySymbol)
	}

	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/tickers", req.ToQueryMap())
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol                 string            `json:"symbol"`
				LastPrice              types.ExDecimal   `json:"lastPrice"`
				IndexPrice             types.ExDecimal   `json:"indexPrice"`
				MarkPrice              types.ExDecimal   `json:"markPrice"`
				PrevPrice24h           types.ExDecimal   `json:"prevPrice24h"`
				Price24hPcnt           types.ExDecimal   `json:"price24hPcnt"`
				HighPrice24h           types.ExDecimal   `json:"highPrice24h"`
				LowPrice24h            types.ExDecimal   `json:"lowPrice24h"`
				PrevPrice1h            types.ExDecimal   `json:"prevPrice1h"`
				OpenInterest           types.ExDecimal   `json:"openInterest"`
				OpenInterestValue      types.ExDecimal   `json:"openInterestValue"`
				Turnover24h            types.ExDecimal   `json:"turnover24h"`
				Volume24h              types.ExDecimal   `json:"volume24h"`
				FundingRate            types.ExDecimal   `json:"fundingRate"`
				NextFundingTime        types.ExTimestamp `json:"nextFundingTime"`
				PredictedDeliveryPrice types.ExDecimal   `json:"predictedDeliveryPrice"`
				BasisRate              types.ExDecimal   `json:"basisRate"`
				DeliveryFeeRate        types.ExDecimal   `json:"deliveryFeeRate"`
				DeliveryTime           types.ExTimestamp `json:"deliveryTime"`
				Ask1Size               types.ExDecimal   `json:"ask1Size"`
				Bid1Price              types.ExDecimal   `json:"bid1Price"`
				Ask1Price              types.ExDecimal   `json:"ask1Price"`
				Bid1Size               types.ExDecimal   `json:"bid1Size"`
				Basis                  types.ExDecimal   `json:"basis"`
				PreOpenPrice           types.ExDecimal   `json:"preOpenPrice"`
				PreQty                 types.ExDecimal   `json:"preQty"`
				CurPreListingPhase     string            `json:"curPreListingPhase"`
				FundingIntervalHour    string            `json:"fundingIntervalHour"`
				BasisRateYear          types.ExDecimal   `json:"basisRateYear"`
				FundingCap             types.ExDecimal   `json:"fundingCap"`
			} `json:"list"`
		} `json:"result"`
		RetExtInfo map[string]interface{} `json:"retExtInfo"`
		Time       types.ExTimestamp      `json:"time"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	if respData.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", respData.RetMsg)
	}

	tickers := make(model.Tickers, 0, len(respData.Result.List))
	for _, item := range respData.Result.List {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.Symbol)
		if err != nil {
			continue
		}
		ticker := &model.Ticker{
			Symbol:    market.Symbol,
			Timestamp: respData.Time,
		}
		ticker.Bid = item.Bid1Price
		ticker.Ask = item.Ask1Price
		ticker.Last = item.LastPrice
		ticker.Open = item.PrevPrice24h
		ticker.High = item.HighPrice24h
		ticker.Low = item.LowPrice24h
		ticker.Volume = item.Volume24h
		ticker.QuoteVolume = item.Turnover24h
		ticker.Timestamp = respData.Time
		tickers = append(tickers, ticker)
	}

	return tickers, nil
}

func (p *BybitPerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, limit int, opts ...option.ArgsOption) (model.OHLCVs, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}
	req := types.NewExValues()
	req.SetQuery("category", "linear")
	req.SetQuery("symbol", market.ID)
	req.SetQuery("interval", common.BybitTimeframe(timeframe))
	req.SetQuery("limit", limit)
	if since, ok := option.GetTime(argsOpts.Since); ok {
		req.SetQuery("start", since.UnixMilli())
	}

	resp, err := p.bybit.client.HTTPClient.Get(ctx, "/v5/market/kline", req.ToQueryMap())
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string          `json:"category"`
			Symbol   string          `json:"symbol"`
			List     [][]interface{} `json:"list"`
		} `json:"result"`
		RetExtInfo map[string]interface{} `json:"retExtInfo"`
		Time       types.ExTimestamp      `json:"time"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	if respData.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", respData.RetMsg)
	}

	ohlcvs := make(model.OHLCVs, 0, len(respData.Result.List))
	for _, item := range respData.Result.List {
		ohlcv := &model.OHLCV{}
		ts, err := strconv.ParseInt(item[0].(string), 10, 64)
		if err != nil {
			return nil, err
		}
		ohlcv.Timestamp = types.ExTimestamp{Time: time.UnixMilli(ts)}
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

func (p *BybitPerp) FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	var querySymbol string
	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		querySymbol = market.ID
	}
	req := types.NewExValues()
	req.SetQuery("category", "linear")
	if querySymbol != "" {
		req.SetQuery("symbol", querySymbol)
	} else {
		req.SetQuery("settleCoin", "USDT")
	}

	resp, err := p.signAndRequest(ctx, "GET", "/v5/position/list", req.ToQueryMap(), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol                 string            `json:"symbol"`
				Leverage               types.ExDecimal   `json:"leverage"`
				AutoAddMargin          int               `json:"autoAddMargin"`
				AvgPrice               types.ExDecimal   `json:"avgPrice"`
				LiqPrice               types.ExDecimal   `json:"liqPrice"`
				RiskLimitValue         types.ExDecimal   `json:"riskLimitValue"`
				TakeProfit             types.ExDecimal   `json:"takeProfit"`
				PositionValue          types.ExDecimal   `json:"positionValue"`
				IsReduceOnly           bool              `json:"isReduceOnly"`
				PositionIMByMp         types.ExDecimal   `json:"positionIMByMp"`
				TpslMode               string            `json:"tpslMode"`
				RiskId                 int               `json:"riskId"`
				TrailingStop           types.ExDecimal   `json:"trailingStop"`
				UnrealisedPnl          types.ExDecimal   `json:"unrealisedPnl"`
				MarkPrice              types.ExDecimal   `json:"markPrice"`
				AdlRankIndicator       int               `json:"adlRankIndicator"`
				CumRealisedPnl         types.ExDecimal   `json:"cumRealisedPnl"`
				PositionMM             types.ExDecimal   `json:"positionMM"`
				CreatedTime            types.ExTimestamp `json:"createdTime"`
				PositionIdx            int               `json:"positionIdx"`
				PositionIM             types.ExDecimal   `json:"positionIM"`
				PositionMMByMp         types.ExDecimal   `json:"positionMMByMp"`
				Seq                    int64             `json:"seq"`
				UpdatedTime            types.ExTimestamp `json:"updatedTime"`
				Side                   string            `json:"side"`
				BustPrice              types.ExDecimal   `json:"bustPrice"`
				PositionBalance        types.ExDecimal   `json:"positionBalance"`
				LeverageSysUpdatedTime types.ExTimestamp `json:"leverageSysUpdatedTime"`
				CurRealisedPnl         types.ExDecimal   `json:"curRealisedPnl"`
				Size                   types.ExDecimal   `json:"size"`
				PositionStatus         string            `json:"positionStatus"`
				MmrSysUpdatedTime      types.ExTimestamp `json:"mmrSysUpdatedTime"`
				StopLoss               types.ExDecimal   `json:"stopLoss"`
				TradeMode              int               `json:"tradeMode"`
				SessionAvgPrice        types.ExDecimal   `json:"sessionAvgPrice"`
			} `json:"list"`
		} `json:"result"`
		Time types.ExTimestamp `json:"time"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if respData.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", respData.RetMsg)
	}

	positions := make([]*model.Position, 0)
	for _, item := range respData.Result.List {
		if item.Size.IsZero() {
			continue
		}

		market, err := p.GetMarket(item.Symbol)
		if err != nil {
			continue
		}

		var side string
		if strings.ToUpper(item.Side) == "BUY" {
			side = string(types.PositionSideLong)
		} else {
			side = string(types.PositionSideShort)
		}

		position := &model.Position{
			Symbol:           market.Symbol,
			Side:             side,
			Amount:           item.Size,
			EntryPrice:       item.AvgPrice,
			MarkPrice:        item.MarkPrice,
			UnrealizedPnl:    item.UnrealisedPnl,
			LiquidationPrice: item.LiqPrice,
			RealizedPnl:      item.CumRealisedPnl,
			Leverage:         item.Leverage,
			Margin:           item.PositionIM,
			Percentage:       types.ExDecimal{},
			Timestamp:        item.UpdatedTime,
		}

		positions = append(positions, position)
	}

	return positions, nil
}

func (p *BybitPerp) CreateOrder(ctx context.Context, symbol string, amount string, orderSide option.PerpOrderSide, orderType option.OrderType, opts ...option.ArgsOption) (*model.NewOrder, error) {
	// 解析选项
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 获取市场信息
	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}

	req := types.NewExValues()
	req.SetBody("category", "linear")
	req.SetBody("symbol", market.ID)

	// 设置限价单价格
	if orderType == option.Limit {
		price, ok := option.GetDecimalFromString(argsOpts.Price)
		if !ok || price.IsZero() {
			return nil, fmt.Errorf("limit order requires price")
		}
		req.SetBody("price", price.String())
		// Limit 单默认使用 GTC
		req.SetBody("timeInForce", option.GTC.Upper())
	}

	if argsOpts.TimeInForce != nil {
		req.SetBody("timeInForce", argsOpts.TimeInForce.Upper())
	}

	// 设置数量
	if quantity, ok := option.GetDecimalFromString(&amount); ok {
		req.SetBody("qty", quantity.String())
	} else {
		return nil, fmt.Errorf("amount is required and must be a valid decimal")
	}

	// Bybit API requires "Buy" or "Sell" (capitalized)
	sideStr := orderSide.ToSide()
	if len(sideStr) > 0 {
		sideStr = strings.ToUpper(sideStr[:1]) + strings.ToLower(sideStr[1:])
	}
	req.SetBody("side", sideStr)
	req.SetBody("orderType", orderType.Capitalize())
	req.SetBody("reduceOnly", orderSide.ToReduceOnly())

	if hedgeMode, ok := option.GetBool(argsOpts.HedgeMode); hedgeMode && ok {
		// 双向持仓模式
		// 开多/平多: positionIdx=1
		// 开空/平空: positionIdx=2
		if orderSide.ToPositionSide() == "LONG" {
			req.SetBody("positionIdx", 1)
		} else {
			req.SetBody("positionIdx", 2)
		}
	} else {
		req.SetBody("positionIdx", 0)
	}

	if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetBody("orderLinkId", clientOrderId)
	} else {
		// 生成订单 ID
		generatedID := common.GenerateClientOrderID(p.bybit.Name(), orderSide.ToSide())
		req.SetBody("orderLinkId", generatedID)
	}

	resp, err := p.signAndRequest(ctx, "POST", "/v5/order/create", nil, req.ToBodyMap())
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var respData struct {
		RetCode int    `json:"retCode"` // 返回码，0 表示成功
		RetMsg  string `json:"retMsg"`  // 返回消息
		Result  struct {
			OrderID     string `json:"orderId"`     // 系统订单号
			OrderLinkID string `json:"orderLinkId"` // 客户端订单ID
		} `json:"result"`                                     // 订单结果
		RetExtInfo map[string]interface{} `json:"retExtInfo"` // 扩展信息
		Time       types.ExTimestamp      `json:"time"`       // 时间戳（毫秒）
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if respData.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", respData.RetMsg)
	}

	// 构建 NewOrder 对象
	perpOrder := &model.NewOrder{
		Symbol:        symbol,
		OrderId:       respData.Result.OrderID,
		ClientOrderID: respData.Result.OrderLinkID,
		Timestamp:     respData.Time,
	}

	return perpOrder, nil
}

func (p *BybitPerp) CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error {
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

	req := types.NewExValues()
	req.SetBody("category", "linear")
	req.SetBody("symbol", market.ID)

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		req.SetBody("orderId", orderId)
	} else if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetBody("orderLinkId", clientOrderId)
	} else {
		return fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	resp, err := p.signAndRequest(ctx, "POST", "/v5/order/cancel", nil, req.ToBodyMap())
	if err != nil {
		return err
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return fmt.Errorf("cancel order fail: %s", err.Error())
	}

	if respData.RetCode != 0 {
		return fmt.Errorf("cancel order fail: %d %s", respData.RetCode, respData.RetMsg)
	}

	return nil
}

func (p *BybitPerp) FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.PerpOrder, error) {
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

	req := types.NewExValues()
	req.SetQuery("category", "linear")
	req.SetQuery("symbol", market.ID)

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		req.SetQuery("orderId", orderId)
	} else if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetQuery("orderLinkId", clientOrderId)
	} else {
		return nil, fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	// First try to fetch from open orders (realtime)
	resp, err := p.signAndRequest(ctx, "GET", "/v5/order/realtime", req.ToQueryMap(), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OrderID     string            `json:"orderId"`     // 订单ID
				OrderLinkID string            `json:"orderLinkId"` // 客户端自定义订单ID
				Symbol      string            `json:"symbol"`      // 交易对 / 合约标的
				Price       types.ExDecimal   `json:"price"`       // 下单价格
				AvgPrice    types.ExDecimal   `json:"avgPrice"`    // 成交均价
				Qty         types.ExDecimal   `json:"qty"`         // 下单数量
				CumExecQty  types.ExDecimal   `json:"cumExecQty"`  // 实际成交数量
				OrderStatus string            `json:"orderStatus"` // 订单状态
				TimeInForce string            `json:"timeInForce"` // 订单有效方式
				ReduceOnly  bool              `json:"reduceOnly"`  // 是否只减仓
				OrderType   string            `json:"orderType"`   // 订单类型
				Side        string            `json:"side"`        // 订单方向
				PositionIdx int               `json:"positionIdx"` // 单向持仓 positionIdx 等于 0，双向持仓 开多/平多 → positionIdx 等于 1，开空/平空 → positionIdx 等于 2
				CreatedTime types.ExTimestamp `json:"createdTime"` // 创建时间（毫秒）
				UpdatedTime types.ExTimestamp `json:"updatedTime"` // 更新时间（毫秒）
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	if respData.RetCode != 0 {
		return nil, fmt.Errorf("fetch order: %d %s", respData.RetCode, respData.RetMsg)
	}

	if len(respData.Result.List) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	originOrder := respData.Result.List[0]
	var positionSide string
	switch originOrder.PositionIdx {
	case 1:
		positionSide = "LONG"
	case 2:
		positionSide = "SHORT"
	default:
		positionSide = "NET"
	}

	order := &model.PerpOrder{
		ID:               originOrder.OrderID,
		ClientID:         originOrder.OrderLinkID,
		Type:             originOrder.OrderType,
		Side:             originOrder.Side,
		PositionSide:     positionSide,
		Symbol:           symbol,
		Price:            originOrder.Price,
		AvgPrice:         originOrder.AvgPrice,
		Quantity:         originOrder.Qty,
		ExecutedQuantity: originOrder.CumExecQty,
		Status:           originOrder.OrderStatus,
		TimeInForce:      originOrder.TimeInForce,
		ReduceOnly:       originOrder.ReduceOnly,
		CreateTime:       originOrder.CreatedTime,
		UpdateTime:       originOrder.UpdatedTime,
	}

	return order, nil
}

func (p *BybitPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}

	if leverage < 1 || leverage > 100 {
		return fmt.Errorf("leverage must be between 1 and 100")
	}

	req := types.NewExValues()
	req.SetBody("category", "linear")
	req.SetBody("symbol", market.ID)
	req.SetBody("buyLeverage", leverage)
	req.SetBody("sellLeverage", leverage)

	resp, err := p.signAndRequest(ctx, "POST", "/v5/position/set-leverage", nil, req.ToBodyMap())
	if err != nil {
		return err
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return fmt.Errorf("set leverage fail: %s", err.Error())
	}

	if respData.RetCode != 0 {
		return fmt.Errorf("set leverage fail: %d %s", respData.RetCode, respData.RetMsg)
	}

	return nil
}

func (p *BybitPerp) SetMarginType(ctx context.Context, symbol string, marginType option.MarginType) error {
	market, err := p.GetMarket(symbol)
	if err != nil || market == nil {
		return err
	}

	req := types.NewExValues()

	switch marginType {
	case option.ISOLATED:
		req.SetBody("setMarginMode", "ISOLATED_MARGIN")
	case option.CROSSED:
		req.SetBody("setMarginMode", "REGULAR_MARGIN")
	default:
		return fmt.Errorf("margin type not supported")
	}

	resp, err := p.signAndRequest(ctx, "POST", "/v5/account/set-margin-mode", nil, req.ToBodyMap())
	if err != nil {
		return err
	}

	var respData struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return fmt.Errorf("set margin type fail: %s", err.Error())
	}

	if respData.RetCode != 0 {
		return fmt.Errorf("set margin type fail: %d %s", respData.RetCode, respData.RetMsg)
	}

	return nil
}

// ========== 内部辅助方法 ==========

// signAndRequest 签名并发送请求（Bybit v5 API）
func (p *BybitPerp) signAndRequest(ctx context.Context, method, path string, params map[string]interface{}, body map[string]interface{}) ([]byte, error) {
	if p.bybit.client.SecretKey == "" {
		return nil, fmt.Errorf("authentication required")
	}

	signature, timestamp := p.bybit.signer.SignRequest(method, params, body)
	recvWindow := "5000"

	// 设置请求头
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-API-KEY", p.bybit.client.APIKey)
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-TIMESTAMP", timestamp)
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-RECV-WINDOW", recvWindow)
	p.bybit.client.HTTPClient.SetHeader("X-BAPI-SIGN", signature)
	p.bybit.client.HTTPClient.SetHeader("Content-Type", "application/json")

	// 发送请求
	if method == "GET" || method == "DELETE" {
		return p.bybit.client.HTTPClient.Get(ctx, path, params)
	} else {
		return p.bybit.client.HTTPClient.Post(ctx, path, body)
	}
}

var _ exchange.PerpExchange = (*BybitPerp)(nil)

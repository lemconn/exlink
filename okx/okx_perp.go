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

	req := types.NewExValues()
	req.SetQuery("instType", "SWAP")

	// 获取永续合约市场信息
	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/public/instruments", req.ToQueryMap())
	if err != nil {
		return fmt.Errorf("fetch swap instruments: %w", err)
	}

	var respData struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstType   string          `json:"instType"`
			InstID     string          `json:"instId"`
			BaseCcy    string          `json:"baseCcy"`
			QuoteCcy   string          `json:"quoteCcy"`
			SettleCcy  string          `json:"settleCcy"`
			Uly        string          `json:"uly"`    // underlying，用于合约市场
			CtType     string          `json:"ctType"` // linear, inverse
			CtVal      string          `json:"ctVal"`  // 合约面值（1张合约等于多少个币）
			CtValCcy   string          `json:"ctValCcy"`
			InstFamily string          `json:"instFamily"`
			State      string          `json:"state"`
			MinSz      types.ExDecimal `json:"minSz"`
			MaxSz      types.ExDecimal `json:"maxSz"`
			LotSz      types.ExDecimal `json:"lotSz"`
			TickSz     types.ExDecimal `json:"tickSz"`
			MinSzVal   types.ExDecimal `json:"minSzVal"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return fmt.Errorf("unmarshal swap instruments: %w", err)
	}

	if respData.Code != "0" {
		return fmt.Errorf("okx api error: %s", respData.Msg)
	}

	markets := make([]*model.Market, 0)
	for _, item := range respData.Data {
		if item.State != "live" {
			continue
		}

		ccy := strings.Split(item.InstFamily, "-")
		if len(ccy) != 2 {
			continue
		}
		baseCcy := ccy[0]
		quoteCcy := ccy[1]
		settleCcy := item.SettleCcy

		// 转换为标准化格式 BTC/USDT:USDT
		normalizedSymbol := common.NormalizeContractSymbol(baseCcy, quoteCcy, settleCcy)

		market := &model.Market{
			ID:            item.InstID,
			Symbol:        normalizedSymbol,
			Base:          baseCcy,
			Quote:         quoteCcy,
			Settle:        settleCcy,
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

func (p *OKXPerp) FetchMarkets(ctx context.Context, opts ...option.ArgsOption) (model.Markets, error) {
	// 解析参数
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	// 确保市场已加载
	if err := p.LoadMarkets(ctx, false); err != nil {
		return nil, err
	}

	if symbol, ok := option.GetString(argsOpts.Symbol); ok {
		market, err := p.GetMarket(symbol)
		if err != nil {
			return nil, err
		}
		return model.Markets{market}, nil
	}

	p.okx.mu.RLock()
	defer p.okx.mu.RUnlock()

	markets := make(model.Markets, 0, len(p.okx.perpMarketsBySymbol))
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

func (p *OKXPerp) FetchTickers(ctx context.Context, opts ...option.ArgsOption) (model.Tickers, error) {
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
	req.SetQuery("instType", "SWAP")

	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/market/tickers", req.ToQueryMap())
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}

	var respData struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstType  string            `json:"instType"`
			InstID    string            `json:"instId"`
			Last      types.ExDecimal   `json:"last"`
			LastSz    types.ExDecimal   `json:"lastSz"`
			AskPx     types.ExDecimal   `json:"askPx"`
			AskSz     types.ExDecimal   `json:"askSz"`
			BidPx     types.ExDecimal   `json:"bidPx"`
			BidSz     types.ExDecimal   `json:"bidSz"`
			Open24h   types.ExDecimal   `json:"open24h"`
			High24h   types.ExDecimal   `json:"high24h"`
			Low24h    types.ExDecimal   `json:"low24h"`
			VolCcy24h types.ExDecimal   `json:"volCcy24h"`
			Vol24h    types.ExDecimal   `json:"vol24h"`
			Ts        types.ExTimestamp `json:"ts"`
			SodUtc0   types.ExDecimal   `json:"sodUtc0"`
			SodUtc8   types.ExDecimal   `json:"sodUtc8"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal tickers: %w", err)
	}

	if respData.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", respData.Msg)
	}

	tickers := make(model.Tickers, 0, len(respData.Data))
	for _, item := range respData.Data {
		// 尝试从市场信息中查找标准化格式
		market, err := p.GetMarket(item.InstID)
		if err != nil {
			continue
		}

		if querySymbol != "" && market.ID != querySymbol {
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
		tickers = append(tickers, ticker)
	}

	return tickers, nil
}

func (p *OKXPerp) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, limit int, opts ...option.ArgsOption) (model.OHLCVs, error) {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	market, err := p.GetMarket(symbol)
	if err != nil {
		return nil, err
	}
	req := types.NewExValues()
	req.SetQuery("instId", market.ID)
	req.SetQuery("bar", common.OKXTimeframe(timeframe))
	req.SetQuery("limit", limit)
	if since, ok := option.GetTime(argsOpts.Since); ok {
		req.SetQuery("after", since.UnixMilli())
	}

	resp, err := p.okx.client.HTTPClient.Get(ctx, "/api/v5/market/candles", req.ToQueryMap())
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	var respData struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data [][]interface{} `json:"data"`
	}
	if err = json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal ohlcv: %w", err)
	}

	if respData.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", respData.Msg)
	}

	ohlcvs := make(model.OHLCVs, 0, len(respData.Data))
	for _, item := range respData.Data {
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

func (p *OKXPerp) FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error) {
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
	req.SetQuery("instType", "SWAP")
	if querySymbol != "" {
		req.SetQuery("instId", querySymbol)
	}

	resp, err := p.signAndRequest(ctx, "GET", "/api/v5/account/positions", req.ToQueryMap(), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}

	var result okxPerpPositionResponse
	var respData struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Adl                    types.ExDecimal   `json:"adl"`
			AvailPos               types.ExDecimal   `json:"availPos"`
			AvgPx                  types.ExDecimal   `json:"avgPx"`
			BaseBal                types.ExDecimal   `json:"baseBal"`
			BaseBorrowed           types.ExDecimal   `json:"baseBorrowed"`
			BaseInterest           types.ExDecimal   `json:"baseInterest"`
			BePx                   types.ExDecimal   `json:"bePx"`
			BizRefId               string            `json:"bizRefId"`
			BizRefType             string            `json:"bizRefType"`
			CTime                  types.ExTimestamp `json:"cTime"`
			Ccy                    string            `json:"ccy"`
			ClSpotInUseAmt         types.ExDecimal   `json:"clSpotInUseAmt"`
			CloseOrderAlgo         []interface{}     `json:"closeOrderAlgo"`
			DeltaBS                types.ExDecimal   `json:"deltaBS"`
			DeltaPA                types.ExDecimal   `json:"deltaPA"`
			Fee                    types.ExDecimal   `json:"fee"`
			FundingFee             types.ExDecimal   `json:"fundingFee"`
			GammaBS                types.ExDecimal   `json:"gammaBS"`
			GammaPA                types.ExDecimal   `json:"gammaPA"`
			HedgedPos              types.ExDecimal   `json:"hedgedPos"`
			IdxPx                  types.ExDecimal   `json:"idxPx"`
			Imr                    types.ExDecimal   `json:"imr"`
			InstID                 string            `json:"instId"`
			InstType               string            `json:"instType"`
			Interest               types.ExDecimal   `json:"interest"`
			Last                   types.ExDecimal   `json:"last"`
			Lever                  types.ExDecimal   `json:"lever"`
			Liab                   types.ExDecimal   `json:"liab"`
			LiabCcy                string            `json:"liabCcy"`
			LiqPenalty             types.ExDecimal   `json:"liqPenalty"`
			LiqPx                  types.ExDecimal   `json:"liqPx"`
			Margin                 types.ExDecimal   `json:"margin"`
			MarkPx                 types.ExDecimal   `json:"markPx"`
			MaxSpotInUseAmt        types.ExDecimal   `json:"maxSpotInUseAmt"`
			MgnMode                string            `json:"mgnMode"`
			MgnRatio               types.ExDecimal   `json:"mgnRatio"`
			Mmr                    types.ExDecimal   `json:"mmr"`
			NonSettleAvgPx         types.ExDecimal   `json:"nonSettleAvgPx"`
			NotionalUsd            types.ExDecimal   `json:"notionalUsd"`
			OptVal                 types.ExDecimal   `json:"optVal"`
			PendingCloseOrdLiabVal types.ExDecimal   `json:"pendingCloseOrdLiabVal"`
			Pnl                    types.ExDecimal   `json:"pnl"`
			Pos                    types.ExDecimal   `json:"pos"`
			PosCcy                 string            `json:"posCcy"`
			PosID                  string            `json:"posId"`
			PosSide                string            `json:"posSide"`
			QuoteBal               types.ExDecimal   `json:"quoteBal"`
			QuoteBorrowed          types.ExDecimal   `json:"quoteBorrowed"`
			QuoteInterest          types.ExDecimal   `json:"quoteInterest"`
			RealizedPnl            types.ExDecimal   `json:"realizedPnl"`
			SettledPnl             types.ExDecimal   `json:"settledPnl"`
			SpotInUseAmt           types.ExDecimal   `json:"spotInUseAmt"`
			SpotInUseCcy           string            `json:"spotInUseCcy"`
			ThetaBS                types.ExDecimal   `json:"thetaBS"`
			ThetaPA                types.ExDecimal   `json:"thetaPA"`
			TradeID                string            `json:"tradeId"`
			UTime                  types.ExTimestamp `json:"uTime"`
			Upl                    types.ExDecimal   `json:"upl"`
			UplLastPx              types.ExDecimal   `json:"uplLastPx"`
			UplRatio               types.ExDecimal   `json:"uplRatio"`
			UplRatioLastPx         types.ExDecimal   `json:"uplRatioLastPx"`
			UsdPx                  types.ExDecimal   `json:"usdPx"`
			VegaBS                 types.ExDecimal   `json:"vegaBS"`
			VegaPA                 types.ExDecimal   `json:"vegaPA"`
		} `json:"data"`
	}
	if err = json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	if respData.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", result.Msg)
	}

	positions := make([]*model.Position, 0)
	for _, item := range respData.Data {
		if item.Pos.IsZero() {
			continue
		}

		market, err := p.GetMarket(item.InstID)
		if err != nil {
			continue
		}

		var side string
		// 根据 posSide 确定持仓方向
		if item.PosSide == "long" || (item.PosSide == "net" && item.Pos.GreaterThan(decimal.Zero)) {
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

func (p *OKXPerp) CreateOrder(ctx context.Context, symbol string, amount string, orderSide option.PerpOrderSide, orderType option.OrderType, opts ...option.ArgsOption) (*model.NewOrder, error) {
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
	req.SetBody("instId", market.ID)

	marginType := argsOpts.MarginType
	if marginType == nil {
		return nil, fmt.Errorf("MarginType cannot be empty, use option.WithMarginType to setting it")
	}
	switch *marginType {
	case option.ISOLATED:
		req.SetBody("tdMode", "isolated")
	case option.CROSSED:
		req.SetBody("tdMode", "cross")
	default:
		return nil, fmt.Errorf("MarginType must be either ISOLATED or CROSSED")
	}

	timeInForce := argsOpts.TimeInForce
	// 限价单，必须设置价格
	if orderType == option.Limit || *timeInForce == option.FOK || *timeInForce == option.IOC {
		price, ok := option.GetDecimalFromString(argsOpts.Price)
		if !ok || price.IsZero() {
			return nil, fmt.Errorf("limit order requires price")
		}
		req.SetBody("px", price)
		if timeInForce != nil {
			req.SetBody("ordType", timeInForce.Lower())
		}
	}

	// 设置数量（合约张数）
	if quantity, ok := option.GetDecimalFromString(&amount); ok {
		req.SetBody("sz", quantity.String())
	} else {
		return nil, fmt.Errorf("amount is required and must be a valid decimal")
	}

	if !req.HasBody("ordType") {
		req.SetBody("ordType", orderType.Lower())
	}
	req.SetBody("side", strings.ToLower(orderSide.ToSide()))
	req.SetBody("reduceOnly", orderSide.ToReduceOnly())

	if hedgeMode, ok := option.GetBool(argsOpts.HedgeMode); hedgeMode && ok {
		// 双向持仓模式
		// 开多/平多: posSide=long
		// 开空/平空: posSide=short
		if orderSide.ToPositionSide() == "LONG" {
			req.SetBody("posSide", "long")
		} else {
			req.SetBody("posSide", "short")
		}
	} else {
		req.SetBody("posSide", "net")
	}

	if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetBody("clOrdId", clientOrderId)
	} else {
		// 生成订单 ID
		generatedID := common.GenerateClientOrderID(p.okx.Name(), orderSide.ToSide())
		req.SetBody("clOrdId", generatedID)
	}

	resp, err := p.signAndRequest(ctx, "POST", "/api/v5/trade/order", nil, req.ToBodyMap())
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var respData struct {
		Code string `json:"code"` // 返回码，"0" 表示成功
		Data []struct {
			ClOrdID string            `json:"clOrdId"` // 客户端订单ID
			OrdID   string            `json:"ordId"`   // 系统订单号
			TS      types.ExTimestamp `json:"ts"`      // 时间戳（毫秒）
		} `json:"data"` // 订单数据数组
		Msg string `json:"msg,omitempty"` // 返回消息
	}
	if err = json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if respData.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", respData.Msg)
	}

	if len(respData.Data) == 0 {
		return nil, fmt.Errorf("okx api error: no order data returned")
	}

	data := respData.Data[0]

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

	req := types.NewExValues()
	req.SetBody("instId", market.ID)

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		req.SetBody("ordId", orderId)
	} else if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetBody("clOrdId", clientOrderId)
	} else {
		return fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	resp, err := p.signAndRequest(ctx, "POST", "/api/v5/trade/cancel-order", nil, req.ToBodyMap())
	if err != nil {
		return err
	}

	var respData struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	if err = json.Unmarshal(resp, &respData); err != nil {
		return err
	}

	if respData.Code != "0" {
		return fmt.Errorf("okx api error: %s", respData.Msg)
	}

	return nil
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

	req := types.NewExValues()
	req.SetQuery("instId", market.ID)

	// 优先使用 orderId 参数，如果没有则使用 ClientOrderID
	if orderId != "" {
		req.SetQuery("ordId", orderId)
	} else if clientOrderId, ok := option.GetString(argsOpts.ClientOrderID); ok {
		req.SetQuery("clOrdId", clientOrderId)
	} else {
		return nil, fmt.Errorf("either orderId parameter or ClientOrderID option must be provided")
	}

	resp, err := p.signAndRequest(ctx, "GET", "/api/v5/trade/order", req.ToQueryMap(), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	var respData struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			OrdID      string            `json:"ordId"`      // 订单ID
			ClOrdID    string            `json:"clOrdId"`    // 客户端自定义订单ID
			InstID     string            `json:"instId"`     // 合约标的
			Px         types.ExDecimal   `json:"px"`         // 下单价格（市价单为空）
			AvgPx      types.ExDecimal   `json:"avgPx"`      // 成交均价
			Sz         types.ExDecimal   `json:"sz"`         // 下单数量
			AccFillSz  types.ExDecimal   `json:"accFillSz"`  // 实际成交数量
			State      string            `json:"state"`      // 订单状态
			ReduceOnly string            `json:"reduceOnly"` // 是否只减仓（字符串 "true"/"false"）
			OrdType    string            `json:"ordType"`    // 订单类型
			Side       string            `json:"side"`       // 订单方向
			PosSide    string            `json:"posSide"`    // 单向持仓 net, 双向持仓 long / short
			CTime      types.ExTimestamp `json:"cTime"`      // 创建时间（毫秒）
			UTime      types.ExTimestamp `json:"uTime"`      // 更新时间（毫秒）
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	if respData.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", respData.Msg)
	}

	if len(respData.Data) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	// 将 OKX 响应转换为 model.PerpOrder
	item := respData.Data[0]
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

func (p *OKXPerp) SetLeverage(ctx context.Context, symbol string, leverage int, opts ...option.ArgsOption) error {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}

	market, err := p.GetMarket(symbol)
	if err != nil {
		return err
	}

	req := types.NewExValues()
	req.SetBody("instId", market.ID)
	if leverage < 1 || leverage > 125 {
		return fmt.Errorf("leverage must be between 1 and 125")
	}
	req.SetBody("lever", leverage)

	switch *argsOpts.MarginType {
	case option.ISOLATED:
		req.SetBody("mgnMode", "isolated")
	case option.CROSSED:
		req.SetBody("mgnMode", "cross")
	default:
		return fmt.Errorf("MarginType cannot be empty, use option.WithMarginType to setting it")
	}

	resp, err := p.signAndRequest(ctx, "POST", "/api/v5/account/set-leverage", nil, req.ToBodyMap())
	if err != nil {
		return err
	}

	var respData struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	if err = json.Unmarshal(resp, &respData); err != nil {
		return err
	}

	if respData.Code != "0" {
		return fmt.Errorf(respData.Msg)
	}

	return nil
}

func (p *OKXPerp) SetMarginType(ctx context.Context, symbol string, marginType option.MarginType, opts ...option.ArgsOption) error {
	argsOpts := &option.ExchangeArgsOptions{}
	for _, opt := range opts {
		opt(argsOpts)
	}
	return fmt.Errorf("not supported: OKX does not support setting margin type via API")
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

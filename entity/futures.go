package entity

type FuturesBalance struct {
	Asset            string `json:"asset" bson:"asset"`
	Balance          string `json:"balance" bson:"balance"`
	Equity           string `json:"equity" bson:"equity"`
	Available        string `json:"available" bson:"available"`
	UnrealizedProfit string `json:"unrealizedProfit" bson:"unrealizedProfit"`
}

type Futures_MarketCandle struct {
	OpenPrice    string `json:"openPrice" bson:"openPrice"`
	HighestPrice string `json:"highestPrice" bson:"highestPrice"`
	LowestPrice  string `json:"lowestPrice" bson:"lowestPrice"`
	ClosePrice   string `json:"closePrice" bson:"closePrice"`
	Volume       string `json:"volume" bson:"volume"`
	Time         int64  `json:"time" bson:"time"`
	Complete     bool   `json:"complete" bson:"complete"`
}

type Futures_InstrumentsInfo struct {
	Symbol         string `json:"symbol" bson:"symbol"`
	Base           string `json:"base" bson:"base"`
	Quote          string `json:"quote" bson:"quote"`
	MinQty         string `json:"minQty" bson:"minQty"`
	MinNotional    string `json:"minNotional" bson:"minNotional"`
	PricePrecision string `json:"pricePrecision" bson:"pricePrecision"`
	SizePrecision  string `json:"sizePrecision" bson:"sizePrecision"`
	State          string `json:"state" bson:"state"`
	MaxLeverage    string `json:"maxLeverage" bson:"maxLeverage"`
	Multiplier     string `json:"multiplier" bson:"multiplier"`
	ContractSize   string `json:"contractSize" bson:"contractSize"`
	IsSizeContract bool   `json:"isSizeContract" bson:"isSizeContract"`
	TokenId        string `json:"tokenId,omitempty" bson:"tokenId,omitempty"`
}

type Futures_PositionsMode struct {
	HedgeMode bool `json:"hedgeMode" bson:"hedgeMode"`
}

type Futures_Leverage struct {
	Symbol        string `json:"symbol" bson:"symbol"`
	Leverage      string `json:"leverage" bson:"leverage"`
	LongLeverage  string `json:"longLeverage" bson:"longLeverage"`
	ShortLeverage string `json:"shortLeverage" bson:"shortLeverage"`
	MarginMode    string `json:"marginMode,omitempty" bson:"marginMode,omitempty"`
}

type Futures_MarginMode struct {
	MarginMode string `json:"marginMode" bson:"marginMode"`
}

type Futures_ListenKey struct {
	ListenKey string `json:"listenKey" bson:"listenKey"`
}

type Futures_PositionsHistory struct {
	Symbol              string `json:"symbol" bson:"symbol"`
	PositionID          string `json:"positionID" bson:"positionID"`
	PositionSide        string `json:"positionSide" bson:"positionSide"`
	PositionAmt         string `json:"positionAmt" bson:"positionAmt"`
	ExecutedPositionAmt string `json:"executedPositionAmt" bson:"executedPositionAmt"`
	AvgPrice            string `json:"avgPrice" bson:"avgPrice"`
	ExecutedAvgPrice    string `json:"executedAvgPrice" bson:"executedAvgPrice"`
	RealisedProfit      string `json:"realisedProfit" bson:"realisedProfit"`
	Fee                 string `json:"fee" bson:"fee"`
	Leverage            string `json:"leverage" bson:"leverage"`
	Funding             string `json:"funding" bson:"funding"`
	MarginMode          string `json:"marginMode" bson:"marginMode"`
	CreateTime          int64  `json:"createTime" bson:"createTime"`
	UpdateTime          int64  `json:"updateTime" bson:"updateTime"`
}

type Futures_OrdersHistory struct {
	Symbol         string `json:"symbol" bson:"symbol"`
	OrderID        string `json:"orderId" bson:"orderId"`
	ClientOrderID  string `json:"clientOrderID" bson:"clientOrderID"`
	PositionID     string `json:"positionID" bson:"positionID"`
	Side           string `json:"side" bson:"side"`
	PositionSide   string `json:"positionSide" bson:"positionSide"`
	PositionSize   string `json:"positionSize" bson:"positionSize"`
	ExecutedSize   string `json:"executedSize" bson:"executedSize"`
	Price          string `json:"price" bson:"price"`
	ExecutedPrice  string `json:"executedPrice" bson:"executedPrice"`
	RealisedProfit string `json:"realisedProfit" bson:"realisedProfit"`
	Fee            string `json:"fee" bson:"fee"`
	FeeAsset       string `json:"feeAsset" bson:"feeAsset"`
	Leverage       string `json:"leverage"  bson:"leverage"`
	Type           string `json:"type" bson:"type"`
	Status         string `json:"status" bson:"status"`
	HedgeMode      bool   `json:"hedgeMode" bson:"hedgeMode"`
	MarginMode     string `json:"marginMode" bson:"marginMode"`
	// Cursor         string `json:"cursor,omitempty" bson:"cursor,omitempty"`
	CreateTime int64 `json:"createTime" bson:"createTime"`
	UpdateTime int64 `json:"updateTime" bson:"updateTime"`
	TpOrder    bool  `json:"tpOrder" bson:"tpOrder"`
	SlOrder    bool  `json:"slOrder" bson:"slOrder"`
}

type Futures_ExecutionsHistory struct {
	Symbol        string `json:"symbol" bson:"symbol"`
	OrderID       string `json:"orderId" bson:"orderId"`
	ClientOrderID string `json:"clientOrderID" bson:"clientOrderID"`
	Side          string `json:"side" bson:"side"`
	PositionSize  string `json:"positionSize" bson:"positionSize"`
	ExecutedSize  string `json:"executedSize" bson:"executedSize"`
	Price         string `json:"price" bson:"price"`
	ExecutedPrice string `json:"executedPrice" bson:"executedPrice"`
	Fee           string `json:"fee" bson:"fee"`
	Type          string `json:"type" bson:"type"`
	Status        string `json:"status" bson:"status"`
	CreateTime    int64  `json:"createTime" bson:"createTime"`
	UpdateTime    int64  `json:"updateTime" bson:"updateTime"`
}

type Futures_UserTrades struct {
	TradeID         string `json:"tradeId" bson:"tradeId"`
	OrderID         string `json:"orderId" bson:"orderId"`
	Symbol          string `json:"symbol" bson:"symbol"`
	Side            string `json:"side" bson:"side"`
	PositionSide    string `json:"positionSide" bson:"positionSide"`
	Price           string `json:"price" bson:"price"`
	Qty             string `json:"qty" bson:"qty"`
	QuoteQty        string `json:"quoteQty" bson:"quoteQty"`
	Commission      string `json:"commission" bson:"commission"`
	CommissionAsset string `json:"commissionAsset" bson:"commissionAsset"`
	RealisedProfit  string `json:"realisedProfit" bson:"realisedProfit"`
	Buyer           bool   `json:"buyer" bson:"buyer"`
	Maker           bool   `json:"maker" bson:"maker"`
	Crossed         bool   `json:"crossed" bson:"crossed"`
	Hash            string `json:"hash,omitempty" bson:"hash,omitempty"`
	BuilderFee      string `json:"builderFee,omitempty" bson:"builderFee,omitempty"`
	Time            int64  `json:"time" bson:"time"`
}

type Futures_Positions struct {
	Symbol           string `json:"symbol" bson:"symbol"`
	PositionSide     string `json:"positionSide" bson:"positionSide"`
	PositionSize     string `json:"positionSize" bson:"positionSize"`
	Leverage         string `json:"leverage" bson:"leverage"`
	PositionID       string `json:"positionID" bson:"positionID"`
	EntryPrice       string `json:"entryPrice" bson:"entryPrice"`
	MarkPrice        string `json:"markPrice" bson:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit" bson:"unRealizedProfit"`
	RealizedProfit   string `json:"realizedProfit" bson:"realizedProfit"`
	Notional         string `json:"notional" bson:"notional"`
	HedgeMode        bool   `json:"hedgeMode" bson:"hedgeMode"`
	MarginMode       string `json:"marginMode" bson:"marginMode"`
	CreateTime       int64  `json:"createTime" bson:"createTime"`
	UpdateTime       int64  `json:"updateTime" bson:"updateTime"`
}

type Futures_OrdersList struct {
	Symbol        string `json:"symbol" bson:"symbol"`
	OrderID       string `json:"orderId" bson:"orderId"`
	ClientOrderID string `json:"clientOrderID" bson:"clientOrderID"`
	PositionID    string `json:"positionID" bson:"positionID"`
	Side          string `json:"side" bson:"side"`
	PositionSide  string `json:"positionSide" bson:"positionSide"`
	PositionSize  string `json:"positionSize" bson:"positionSize"`
	ExecutedSize  string `json:"executedSize" bson:"executedSize"`
	Price         string `json:"price" bson:"price"`
	Leverage      string `json:"leverage" bson:"leverage"`
	Type          string `json:"type" bson:"type"`
	Status        string `json:"status" bson:"status"`
	CreateTime    int64  `json:"createTime" bson:"createTime"`
	UpdateTime    int64  `json:"updateTime" bson:"updateTime"`
	MarginMode    string `json:"marginMode" bson:"marginMode"`
	TpOrder       bool   `json:"tpOrder" bson:"tpOrder"`
	SlOrder       bool   `json:"slOrder" bson:"slOrder"`
}

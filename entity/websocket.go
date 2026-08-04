package entity

type WsHandlerMarkPrice func(event *WsPublicMarkPriceEvent)
type WsHandlerPrivatePositions func(event *WsPrivatePositionsEvent)
type WsHandlerPrivateOrders func(event *WsPrivateOrdersEvent)

type WsPublicMarkPriceEvent struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Time   int64   `json:"time"`
}

type WsPrivatePositionsEvent struct {
	Symbol           string  `json:"symbol" bson:"symbol"`
	PositionSide     string  `json:"positionSide" bson:"positionSide"`
	PositionAmt      float64 `json:"positionAmt" bson:"positionAmt"`
	EntryPrice       float64 `json:"entryPrice" bson:"entryPrice"`
	MarkPrice        float64 `json:"markPrice" bson:"markPrice"`
	UnRealizedProfit float64 `json:"unRealizedProfit" bson:"unRealizedProfit"`
	RealizedProfit   float64 `json:"realizedProfit" bson:"realizedProfit"`
	Notional         float64 `json:"notional" bson:"notional"`
	InitialMargin    float64 `json:"initialMargin" bson:"initialMargin"`
	UpdateTime       int64   `json:"updateTime" bson:"updateTime"`
}

type WsPrivateOrdersEvent struct {
	Symbol       string  `json:"symbol" bson:"symbol"`
	OrderID      string  `json:"orderId" bson:"orderId"`
	Side         string  `json:"side" bson:"side"`
	PositionSide string  `json:"positionSide" bson:"positionSide"`
	PositionAmt  float64 `json:"positionAmt" bson:"positionAmt"`
	Price        float64 `json:"price" bson:"price"`
	Notional     float64 `json:"notional" bson:"notional"`
	Type         string  `json:"type" bson:"type"`
	Status       string  `json:"status" bson:"status"`
	Time         int64   `json:"time" bson:"time"`
	UpdateTime   int64   `json:"updateTime" bson:"updateTime"`
}

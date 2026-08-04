package entity

type OrdersHistory struct {
	Symbol        string  `json:"symbol" bson:"symbol"`
	OrderID       string  `json:"orderId" bson:"orderId"`
	ClientOrderID string  `json:"clientOrderID"`
	Side          string  `json:"side" bson:"side"`
	PositionSide  string  `json:"positionSide" bson:"positionSide"`
	Category      string  `json:"category" bson:"category"`
	Price         float64 `json:"price" bson:"price"`
	FillPrice     float64 `json:"fillPrice" bson:"fillPrice"`
	Size          float64 `json:"size" bson:"size"`
	FillSize      float64 `json:"fillSize" bson:"fillSize"`
	Notional      float64 `json:"notional" bson:"notional"`
	Type          string  `json:"type" bson:"type"`
	Status        string  `json:"status" bson:"status"`
	Pnl           float64 `json:"pnl" bson:"pnl"`
	Fee           float64 `json:"fee" bson:"fee"`
	CreateTime    int64   `json:"createTime" bson:"createTime"`
	UpdateTime    int64   `json:"updateTime" bson:"updateTime"`
	Hex           string  `json:"hex" bson:"hex"`
}

type OrderList struct {
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

package entity

type SubAccountBalance struct {
	EquityBalance    float64 `json:"equityBalance" bson:"equityBalance"`
	UnrealizedProfit float64 `json:"unrealizedProfit" bson:"unrealizedProfit"`
}

type SubAccountFundingBalance struct {
	Asset            string  `json:"asset" bson:"asset"`
	Balance          float64 `json:"balance" bson:"balance"`
	FrozenBalance    float64 `json:"frozenBalance" bson:"frozenBalance"`
	AvailableBalance float64 `json:"availableBalance" bson:"availableBalance"`
}

type TransferHistory struct {
	Asset      string  `json:"asset" bson:"asset"`
	SubID      string  `json:"subID" bson:"subID"`
	BillID     string  `json:"billID" bson:"billID"`
	Tag        string  `json:"tag" bson:"tag"`
	Amount     float64 `json:"amount" bson:"amount"`
	Type       string  `json:"type" bson:"type"`
	CreateTime int64   `json:"createTime" bson:"createTime"`
}

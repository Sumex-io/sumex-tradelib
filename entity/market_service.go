package entity

type ContractInfo struct {
	Symbol             string  `json:"symbol" bson:"symbol"`
	ContractSize       float64 `json:"contractSize" bson:"contractSize"`
	ContractMultiplier float64 `json:"contractMultiplier" bson:"contractMultiplier"`
	StepContractSize   float64 `json:"stepContractSize" bson:"stepContractSize"`
	MinContractSize    float64 `json:"minContractSize" bson:"minContractSize"`
	StepTickPrice      float64 `json:"stepTickPrice" bson:"stepTickPrice"`
	MaxLeverage        int     `json:"lever" bson:"lever"`
	State              string  `json:"state" bson:"state"`
	Type               string  `json:"type" bson:"type"`
}

type ContractInfo_Option struct {
	Symbol             string  `json:"symbol" bson:"symbol"`
	ContractSize       float64 `json:"contractSize" bson:"contractSize"`
	ContractMultiplier float64 `json:"contractMultiplier" bson:"contractMultiplier"`
	StepContractSize   float64 `json:"stepContractSize" bson:"stepContractSize"`
	MinContractSize    float64 `json:"minContractSize" bson:"minContractSize"`
	StepTickPrice      float64 `json:"stepTickPrice" bson:"stepTickPrice"`
	Strike             float64 `json:"strike" bson:"strike"`
	MaxLeverage        int     `json:"lever" bson:"lever"`
	State              string  `json:"state" bson:"state"`
	Type               string  `json:"type" bson:"type"`
	ListTime           int64   `json:"listTime" bson:"listTime"`
	ExpTime            int64   `json:"expTime" bson:"expTime"`
}

type Kline struct {
	OpenPrice    float64 `json:"openPrice" bson:"openPrice"`
	HighestPrice float64 `json:"highestPrice" bson:"highestPrice"`
	LowestPrice  float64 `json:"lowestPrice" bson:"lowestPrice"`
	ClosePrice   float64 `json:"closePrice" bson:"closePrice"`
	Time         int64   `json:"time" bson:"time"`
	Complete     bool    `json:"complete" bson:"complete"`
}

type MarkPrice struct {
	Symbol string  `json:"symbol" bson:"symbol"`
	Price  float64 `json:"price" bson:"price"`
}

type MarketData_Option struct {
	Symbol     string  `json:"symbol" bson:"symbol"`
	Delta      float64 `json:"delta" bson:"delta"`
	Gamma      float64 `json:"gamma" bson:"gamma"`
	Vega       float64 `json:"vega" bson:"vega"`
	Theta      float64 `json:"theta" bson:"theta"`
	DeltaBS    float64 `json:"deltaBS" bson:"deltaBS"`
	GammaBS    float64 `json:"gammaBS" bson:"gammaBS"`
	VegaBS     float64 `json:"vegaBS" bson:"vegaBS"`
	ThetaBS    float64 `json:"thetaBS" bson:"thetaBS"`
	Leverage   int     `json:"leverage" bson:"leverage"`
	MarkVol    float64 `json:"markVol" bson:"markVol"`
	BidVol     float64 `json:"bidVol" bson:"bidVol"`
	AskVol     float64 `json:"askVol" bson:"askVol"`
	RealVol    float64 `json:"realVol" bson:"realVol"`
	VolLv      float64 `json:"volLv" bson:"volLv"`
	FwdPx      float64 `json:"fwdPx" bson:"fwdPx"`
	Strike     float64 `json:"strike" bson:"strike"`
	Type       string  `json:"type" bson:"type"`
	UpdateTime int64   `json:"updateTime" bson:"updateTime"`
}

type AsksBids_Option struct {
	Price     float64 `json:"price" bson:"price"`
	Qty       float64 `json:"qty" bson:"qty"`
	NumOrders float64 `json:"numOrders" bson:"numOrders"`
}
type OrderBook_Option struct {
	Asks       []AsksBids_Option `json:"asks" bson:"asks"`
	Bids       []AsksBids_Option `json:"bids" bson:"bids"`
	UpdateTime int64             `json:"updateTime" bson:"updateTime"`
}

type Ticker struct {
	Symbol        string  `json:"symbol" bson:"symbol"`
	Open24hPrice  float64 `json:"open24hPrice" bson:"open24hPrice"`
	LastPrice     float64 `json:"lastPrice" bson:"lastPrice"`
	Volume24hCoin float64 `json:"volume24hCoin" bson:"volume24hCoin"`
	Volume24hUSDT float64 `json:"volume24hUSDT" bson:"volume24hUSDT"`
	Change24h     float64 `json:"change24h" bson:"change24h"`
	Time          int64   `json:"time" bson:"time"`
}

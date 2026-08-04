package weex

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
	symbol  *string
}

func (s *spot_getInstrumentsInfo) Symbol(symbol string) *spot_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *spot_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v3/exchangeInfo",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{}

	if s.symbol != nil && *s.symbol != "" {
		m["symbol"] = *s.symbol
	} else {
		m["symbolStatus"] = "TRADING"
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ spot_exchangeInfoResponse
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err = dec.Decode(&answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertInstrumentsInfo(answ.Symbols), nil
}

type spot_exchangeInfoResponse struct {
	Timezone   string                 `json:"timezone"`
	ServerTime int64                  `json:"serverTime"`
	Symbols    []spot_instrumentsInfo `json:"symbols"`
}

type spot_instrumentsInfo struct {
	Symbol                   string      `json:"symbol"`
	Status                   string      `json:"status"`
	BaseAsset                string      `json:"baseAsset"`
	BaseAssetPrecision       int         `json:"baseAssetPrecision"`
	QuoteAsset               string      `json:"quoteAsset"`
	QuoteAssetPrecision      int         `json:"quoteAssetPrecision"`
	TickSize                 json.Number `json:"tickSize"`
	StepSize                 json.Number `json:"stepSize"`
	MinTradeAmount           json.Number `json:"minTradeAmount"`
	MaxTradeAmount           json.Number `json:"maxTradeAmount"`
	TakerFeeRate             json.Number `json:"takerFeeRate"`
	MakerFeeRate             json.Number `json:"makerFeeRate"`
	BuyLimitPriceRatio       json.Number `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio      json.Number `json:"sellLimitPriceRatio"`
	MarketBuyLimitSize       json.Number `json:"marketBuyLimitSize"`
	MarketSellLimitSize      json.Number `json:"marketSellLimitSize"`
	MarketFallbackPriceRatio json.Number `json:"marketFallbackPriceRatio"`
	EnableTrade              bool        `json:"enableTrade"`
	EnableDisplay            bool        `json:"enableDisplay"`
	DisplayDigitMerge        string      `json:"displayDigitMerge"`
	DisplayNew               bool        `json:"displayNew"`
	DisplayHot               bool        `json:"displayHot"`
}

package weex

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getInstrumentsInfo) Symbol(symbol string) *futures_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *futures_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/capi/v3/market/exchangeInfo",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{}
	if s.symbol != nil && *s.symbol != "" {
		m["symbol"] = *s.symbol
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ futures_exchangeInfoResponse
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err = dec.Decode(&answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertInstrumentsInfo(answ.Symbols), nil
}

type futures_exchangeInfoResponse struct {
	Assets  []futures_asset           `json:"assets"`
	Symbols []futures_instrumentsInfo `json:"symbols"`
}

type futures_asset struct {
	Asset           string `json:"asset"`
	MarginAvailable bool   `json:"marginAvailable"`
}

type futures_instrumentsInfo struct {
	Symbol              string      `json:"symbol"`
	BaseAsset           string      `json:"baseAsset"`
	QuoteAsset          string      `json:"quoteAsset"`
	MarginAsset         string      `json:"marginAsset"`
	PricePrecision      int         `json:"pricePrecision"`
	QuantityPrecision   int         `json:"quantityPrecision"`
	BaseAssetPrecision  int         `json:"baseAssetPrecision"`
	QuotePrecision      int         `json:"quotePrecision"`
	ContractVal         json.Number `json:"contractVal"`
	MinLeverage         int         `json:"minLeverage"`
	MaxLeverage         int         `json:"maxLeverage"`
	BuyLimitPriceRatio  json.Number `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio json.Number `json:"sellLimitPriceRatio"`
	MakerFeeRate        json.Number `json:"makerFeeRate"`
	TakerFeeRate        json.Number `json:"takerFeeRate"`
	MinOrderSize        json.Number `json:"minOrderSize"`
	MaxOrderSize        json.Number `json:"maxOrderSize"`
	MaxPositionSize     json.Number `json:"maxPositionSize"`
	MarketOpenLimitSize json.Number `json:"marketOpenLimitSize"`
	ForwardContractFlag bool        `json:"forwardContractFlag"`
	Delivery            []string    `json:"delivery"`
}

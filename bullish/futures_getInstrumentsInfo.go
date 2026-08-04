package bullish

import (
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
		Endpoint: "/trading-api/v1/markets",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{"marketType": "PERPETUAL"}
	if s.symbol != nil {
		// m["symbol"] = *s.symbol
		r.Endpoint += "/" + *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	if s.symbol != nil {
		answ := futures_instrumentsInfo{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}
		res = s.convert.convertInstrumentsInfo([]futures_instrumentsInfo{answ})
		return res, nil

	} else {
		answ := []futures_instrumentsInfo{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		res = s.convert.convertInstrumentsInfo(answ)
		return res, nil
	}

}

type futures_instrumentsInfo struct {
	Symbol             string `json:"symbol"`
	BaseSymbol         string `json:"baseSymbol"`
	QuoteSymbol        string `json:"quoteSymbol"`
	BasePrecision      string `json:"basePrecision"`
	PricePrecision     string `json:"pricePrecision"`
	MinQuantityLimit   string `json:"minQuantityLimit"`
	TickSize           string `json:"tickSize"`
	MarketType         string `json:"marketType"`
	ContractMultiplier string `json:"contractMultiplier"`
	CreateOrderEnabled bool   `json:"createOrderEnabled"`
}

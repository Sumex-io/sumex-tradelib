package bybit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol *string
}

func (s *spot_getInstrumentsInfo) Symbol(symbol string) *spot_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *spot_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/market/instruments-info",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{
		"category": "spot",
		"limit":    1000,
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			List []spot_instrumentsInfo `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertInstrumentsInfo(answ.Result.List), nil
}

type spot_instrumentsInfo struct {
	Symbol        string `json:"symbol"`
	BaseCoin      string `json:"baseCoin"`
	QuoteCoin     string `json:"quoteCoin"`
	Status        string `json:"status"`
	LotSizeFilter struct {
		BasePrecision  string `json:"basePrecision"`
		QuotePrecision string `json:"quotePrecision"`
		MinOrderQty    string `json:"minOrderQty"`
		MaxOrderQty    string `json:"maxOrderQty"`
		MinOrderAmt    string `json:"minOrderAmt"`
		MaxOrderAmt    string `json:"maxOrderAmt"`
	} `json:"lotSizeFilter"`
	PriceFilter struct {
		TickSize string `json:"tickSize"`
	} `json:"priceFilter"`
}

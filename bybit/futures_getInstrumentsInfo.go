package bybit

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

	symbol   *string
	category *string
}

func (s *futures_getInstrumentsInfo) Symbol(symbol string) *futures_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *futures_getInstrumentsInfo) Category(category string) *futures_getInstrumentsInfo {
	s.category = &category
	return s
}

func (s *futures_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/market/instruments-info",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{
		"category": "linear",
		"limit":    1000,
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.category != nil {
		m["category"] = *s.category
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result struct {
			List []futures_instrumentsInfo `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertInstrumentsInfo(answ.Result.List), nil
}

type futures_instrumentsInfo struct {
	Symbol        string `json:"symbol"`
	BaseCoin      string `json:"baseCoin"`
	QuoteCoin     string `json:"quoteCoin"`
	Status        string `json:"status"`
	PriceScale    string `json:"priceScale"`
	LotSizeFilter struct {
		MinOrderQty      string `json:"minOrderQty"`
		MaxOrderQty      string `json:"maxOrderQty"`
		MinNotionalValue string `json:"minNotionalValue"`
		QtyStep          string `json:"qtyStep"`
	} `json:"lotSizeFilter"`
	PriceFilter struct {
		TickSize string `json:"tickSize"`
	} `json:"priceFilter"`
	LeverageFilter struct {
		MaxLeverage string `json:"maxLeverage"`
	} `json:"leverageFilter"`
}

package kucoin

import (
	"context"
	"encoding/json"
	"fmt"
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
		Endpoint: "/api/v2/symbols",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{}

	if s.symbol != nil {
		r.Endpoint = fmt.Sprintf("%s/%s", r.Endpoint, *s.symbol)
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	if s.symbol != nil {
		var answ struct {
			Result spot_instrumentsInfo `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertInstrumentsInfo([]spot_instrumentsInfo{answ.Result}), nil
	} else {
		var answ struct {
			Result []spot_instrumentsInfo `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertInstrumentsInfo(answ.Result), nil
	}

}

type spot_instrumentsInfo struct {
	Symbol         string `json:"symbol"`
	BaseCurrency   string `json:"baseCurrency"`
	QuoteCurrency  string `json:"quoteCurrency"`
	BaseMinSize    string `json:"baseMinSize"`
	QuoteMinSize   string `json:"quoteMinSize"`
	BaseIncrement  string `json:"baseIncrement"`
	QuoteIncrement string `json:"quoteIncrement"`
	PriceIncrement string `json:"priceIncrement"`
	EnableTrading  bool   `json:"enableTrading"`
}

package kucoin

import (
	"context"
	"encoding/json"
	"fmt"
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
		Endpoint: "/api/v1/contracts/active",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{}

	if s.symbol != nil {
		r.Endpoint = fmt.Sprintf("/api/v1/contracts/%s", *s.symbol)
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	if s.symbol != nil {
		var answ struct {
			Result futures_instrumentsInfo `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}
		return s.convert.convertInstrumentsInfo([]futures_instrumentsInfo{answ.Result}), nil
	} else {
		var answ struct {
			Result []futures_instrumentsInfo `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}
		return s.convert.convertInstrumentsInfo(answ.Result), nil
	}

}

type futures_instrumentsInfo struct {
	Symbol             string  `json:"symbol"`
	BaseCurrency       string  `json:"baseCurrency"`
	QuoteCurrency      string  `json:"quoteCurrency"`
	Status             string  `json:"status"`
	Multiplier         float64 `json:"multiplier"`
	LotSize            float64 `json:"lotSize"`
	TickSize           float64 `json:"tickSize"`
	IndexPriceTickSize float64 `json:"indexPriceTickSize"`
	MaxLeverage        int64   `json:"maxLeverage"`
	EnableTrading      bool    `json:"enableTrading"`
}

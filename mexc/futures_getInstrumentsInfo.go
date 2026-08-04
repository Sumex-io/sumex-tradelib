package mexc

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
		Endpoint: "/api/v1/contract/detail",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
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
	Symbol           string  `json:"symbol"`
	BaseCoin         string  `json:"baseCoin"`
	QuoteCoin        string  `json:"quoteCoin"`
	ContractSize     string  `json:"contractSize"`
	MaxLeverage      int64   `json:"maxLeverage"`
	PriceScale       int64   `json:"priceScale"`
	VolScale         int64   `json:"volScale"`
	State            int64   `json:"state"`
	TradeMinQuantity float64 `json:"tradeMinQuantity"`
}

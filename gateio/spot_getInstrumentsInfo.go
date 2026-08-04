package gateio

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
		Endpoint: "/api/v4/spot/currency_pairs",

		SecType: utils.SecTypeNone,
	}

	if s.symbol != nil {
		r.Endpoint = fmt.Sprintf("%s/%s", r.Endpoint, *s.symbol)
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	if s.symbol != nil {
		answ := spot_instrumentsInfo{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertInstrumentsInfo([]spot_instrumentsInfo{answ}), nil

	} else {
		answ := []spot_instrumentsInfo{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertInstrumentsInfo(answ), nil
	}

}

type spot_instrumentsInfo struct {
	ID               string `json:"id"`
	Base             string `json:"base"`
	Quote            string `json:"quote"`
	Min_base_amount  string `json:"min_base_amount"`
	Min_quote_amount string `json:"min_quote_amount"`
	Amount_precision int64  `json:"amount_precision"`
	Precision        int64  `json:"precision"`
	Trade_status     string `json:"trade_status"`
}

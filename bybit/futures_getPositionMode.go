package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol   *string
	category *string
}

func (s *futures_getPositionMode) Symbol(symbol string) *futures_getPositionMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositionMode) Category(category string) *futures_getPositionMode {
	s.category = &category
	return s
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/position/list",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"category": "linear"}
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
			List []futures_leverage `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result.List) == 0 {
		return res, errors.New("Empty Results")
	} else if len(answ.Result.List) == 1 {
		res.HedgeMode = false
	} else {
		res.HedgeMode = true
	}

	return res, nil
}

// type futures_positionMode struct {
// 	DualSidePosition string `json:"dualSidePosition"`
// }

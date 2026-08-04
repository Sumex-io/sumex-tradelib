package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetPositions=================
type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle *string
}

func (s *futures_getPositions) Settle(settle string) *futures_getPositions {
	s.settle = &settle
	return s
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v4/futures/{settle}/positions",
		SecType:  utils.SecTypeSigned,
	}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := []futures_Position{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertPositions(answ), nil
}

type futures_Position struct {
	Contract             string `json:"contract"`
	Value                string `json:"value"`
	Leverage             string `json:"leverage"`
	Cross_leverage_limit string `json:"cross_leverage_limit"`
	Mode                 string `json:"mode"`
	Entry_price          string `json:"entry_price"`
	Mark_price           string `json:"mark_price"`
	Realised_pnl         string `json:"realised_pnl"`
	Unrealised_pnl       string `json:"unrealised_pnl"`
	Size                 int64  `json:"size"`
	Maintenance_rate     string `json:"maintenance_rate"`
	Initial_margin       string `json:"initial_margin"`
	Open_time            int64  `json:"open_time"`
	Update_time          int64  `json:"update_time"`
}

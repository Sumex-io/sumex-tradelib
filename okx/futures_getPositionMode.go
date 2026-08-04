package okx

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
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/account/config",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []futures_positionMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	b := false

	if answ.Result[0].PosMode == "long_short_mode" {
		b = true
	}
	return entity.Futures_PositionsMode{HedgeMode: b}, nil
}

type futures_positionMode struct {
	PosMode string `json:"posMode"`
}

package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
	convert futures_converts
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/fapi/v1/positionSide/dual",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/positionSide/dual"
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := futures_positionMode{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return entity.Futures_PositionsMode{HedgeMode: answ.DualSidePosition}, nil
}

type futures_positionMode struct {
	DualSidePosition bool `json:"dualSidePosition"`
}

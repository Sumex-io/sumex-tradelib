package weex

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getPositionMode) Symbol(symbol string) *futures_getPositionMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/capi/v3/account/accountConfig",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ futures_accountConfig
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return entity.Futures_PositionsMode{
		HedgeMode: answ.DualSidePosition,
	}, nil
}

type futures_accountConfig struct {
	CanTrade         bool  `json:"canTrade"`
	CanDeposit       bool  `json:"canDeposit"`
	CanWithdraw      bool  `json:"canWithdraw"`
	DualSidePosition bool  `json:"dualSidePosition"`
	UpdateTime       int64 `json:"updateTime"`
}

package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	marginMode *entity.MarginModeType
}

func (s *futures_setMarginMode) MarginMode(marginMode entity.MarginModeType) *futures_setMarginMode {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/v5/account/set-margin-mode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["setMarginMode"] = "REGULAR_MARGIN"
		case entity.MarginModeTypeIsolated:
			m["setMarginMode"] = "ISOLATED_MARGIN"
		}
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		RetMsg  string `json:"retMsg"`
		RetCode int64  `json:"retCode"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.RetCode != 0 {
		return res, errors.New(answ.RetMsg)
	}

	return entity.Futures_MarginMode{MarginMode: string(*s.marginMode)}, nil
}

// type futures_leverage struct {
// 	Symbol           string `json:"symbol"`
// 	LongLeverage     int    `json:"longLeverage"`
// 	MaxLongLeverage  int    `json:"maxLongLeverage"`
// 	MaxShortLeverage int    `json:"maxShortLeverage"`
// }

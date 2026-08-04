package blofin

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
	if s.marginMode == nil {
		return res, errors.New("marginMode is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v1/account/set-margin-mode",
		SecType:  utils.SecTypeSigned,
	}

	// Blofin ожидает "cross" / "isolated" в нижнем регистре
	var mode string
	switch *s.marginMode {
	case entity.MarginModeTypeCross:
		mode = "cross"
	case entity.MarginModeTypeIsolated:
		mode = "isolated"
	default:
		return res, errors.New("unsupported marginMode: " + string(*s.marginMode))
	}

	form := utils.Params{
		"marginMode": mode,
	}

	// для POST используем Form, а не Query
	r.SetFormParams(form)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Data futures_marginMode `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// конвертер сам переведёт в "CROSS"/"ISOLATED"
	return s.convert.convertMarginMode(answ.Data), nil
}

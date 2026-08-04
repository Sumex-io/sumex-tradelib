package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== GetPositionMode (hedge mode) ===================

type futures_getPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/hedge-mode",
		SecType:  utils.SecTypeSigned,
	}

	// По доке: тело — стандартное {request, nonce}, дополнительных полей нет.
	// Наш whitebit request.go как раз сам добавляет request/nonce в body,
	// поэтому достаточно передать пустой form.
	r.SetFormParams(utils.Params{})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		HedgeMode bool `json:"hedgeMode"`
	}

	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return entity.Futures_PositionsMode{
		HedgeMode: answ.HedgeMode,
	}, nil
}

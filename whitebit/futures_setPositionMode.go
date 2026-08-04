package whitebit

import (
	"context"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== SetPositionMode (update hedge mode) ===================

type futures_setPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	mode *entity.PositionModeType
}

// Mode задаёт желаемый режим позиций:
//   - PositionModeTypeHedge  -> hedgeMode = true
//   - PositionModeTypeOneWay -> hedgeMode = false
func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	if s.mode == nil {
		return res, errors.New("position mode is required")
	}

	// маппим наш enum в bool под WhiteBIT
	hedge := false
	if *s.mode == entity.PositionModeTypeHedge {
		hedge = true
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/hedge-mode/update",
		SecType:  utils.SecTypeSigned,
	}

	// тело по доке:
	// { "hedgeMode": true/false }
	// наш whitebit request.go сверху добавит request + nonce.
	m := utils.Params{
		"hedgeMode": hedge,
	}
	r.SetFormParams(m)

	_, _, err = s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// формат успешного ответа не задокументирован,
	// поэтому считаем, что раз ошибки не было — режим применён.
	return entity.Futures_PositionsMode{
		HedgeMode: hedge,
	}, nil
}

package whitebit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== SetLeverage (account-level) ===================

type futures_setLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol     *string
	leverage   *string
	marginMode *entity.MarginModeType
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
	return s
}

func (s *futures_setLeverage) MarginMode(marginMode entity.MarginModeType) *futures_setLeverage {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	if s.leverage == nil {
		return res, errors.New("leverage is required")
	}

	lv, err := strconv.Atoi(*s.leverage)
	if err != nil {
		return res, err
	}

	// по доке:
	// POST /api/v4/collateral-account/leverage
	// { "leverage": 5, "request": "...", "nonce": ... }
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/leverage",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"leverage": lv, // Int по спецификации
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// Ответ по доке:
	// { "leverage": 5 }
	var answ struct {
		Leverage int `json:"leverage"`
	}

	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	levStr := strconv.Itoa(answ.Leverage)

	// Плечо аккаунт-уровня, без конкретного символа
	res.Symbol = ""
	res.Leverage = levStr
	res.LongLeverage = levStr
	res.ShortLeverage = levStr

	return res, nil
}

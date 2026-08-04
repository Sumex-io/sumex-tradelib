package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
	symbol  *string
}

func (s *spot_getInstrumentsInfo) Symbol(symbol string) *spot_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *spot_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v4/public/markets",
		SecType:  utils.SecTypeNone,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []spot_instrumentWB
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// Локальный фильтр по символу, если он задан
	if s.symbol != nil {
		filtered := make([]spot_instrumentWB, 0, 1)
		for _, item := range answ {
			if item.Name == *s.symbol {
				filtered = append(filtered, item)
			}
		}
		answ = filtered
	}

	res = s.convert.convertInstrumentsInfo(answ)
	return res, nil
}

type spot_instrumentWB struct {
	Name          string `json:"name"`
	Stock         string `json:"stock"`
	Money         string `json:"money"`
	StockPrec     string `json:"stockPrec"` // было int
	MoneyPrec     string `json:"moneyPrec"` // было int
	FeePrec       string `json:"feePrec"`   // можно тоже строкой
	MinAmount     string `json:"minAmount"`
	MinTotal      string `json:"minTotal"`
	TradesEnabled bool   `json:"tradesEnabled"`
	Type          string `json:"type"`
	IsCollateral  bool   `json:"isCollateral"`
	MaxTotal      string `json:"maxTotal"`
	MakerFee      string `json:"makerFee"`
	TakerFee      string `json:"takerFee"`
}

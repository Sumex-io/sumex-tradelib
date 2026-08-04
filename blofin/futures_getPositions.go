package blofin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetPositions=================
type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/account/positions", // правильный маршрут Blofin
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_Position `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertPositions(answ.Result), nil
}

// структура под Blofin (см. docs: GET /api/v1/account/positions)
type futures_Position struct {
	PositionID         string `json:"positionId"`
	InstID             string `json:"instId"`
	InstType           string `json:"instType"`
	MarginMode         string `json:"marginMode"`
	PositionSide       string `json:"positionSide"`
	Adl                string `json:"adl"`
	Positions          string `json:"positions"`
	AvailablePositions string `json:"availablePositions"`
	AveragePrice       string `json:"averagePrice"`
	Margin             string `json:"margin"`
	MarkPrice          string `json:"markPrice"`
	MarginRatio        string `json:"marginRatio"`
	LiquidationPrice   string `json:"liquidationPrice"`
	UnrealizedPnl      string `json:"unrealizedPnl"`
	UnrealizedPnlRatio string `json:"unrealizedPnlRatio"`
	MaintenanceMargin  string `json:"maintenanceMargin"`
	CreateTime         string `json:"createTime"`
	UpdateTime         string `json:"updateTime"`
	Leverage           string `json:"leverage"`
}

package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64

	orderID *string
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) StartTime(startTime int64) *futures_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_ordersHistory) EndTime(endTime int64) *futures_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_ordersHistory) Limit(limit int64) *futures_ordersHistory {
	s.limit = &limit
	return s
}

func (s *futures_ordersHistory) Page(page int64) *futures_ordersHistory {
	s.page = &page
	return s
}

func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	// -----------------------------------------
	// 1) Обычная история filled orders
	// -----------------------------------------
	r1 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/trade/orders-history",
		SecType:  utils.SecTypeSigned,
	}

	m1 := utils.Params{
		"instType": "SWAP",
		"state":    "filled",
	}

	if s.symbol != nil {
		m1["instId"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m1["limit"] = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m1["page"] = *s.page
	}
	if s.startTime != nil {
		m1["begin"] = *s.startTime
	}
	if s.endTime != nil {
		m1["end"] = *s.endTime
	}
	if s.orderID != nil {
		m1["before"] = *s.orderID
	}

	r1.SetParams(m1)

	data1, _, err := s.callAPI(ctx, r1, opts...)
	if err != nil {
		return res, err
	}

	var answ1 struct {
		Result []futures_ordersHistory_Response `json:"data"`
	}

	err = json.Unmarshal(data1, &answ1)
	if err != nil {
		return res, err
	}

	out := convertOrdersHistoryOKX(answ1.Result)

	// -----------------------------------------
	// 2) История conditional algo (TP/SL)
	// -----------------------------------------
	r2 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/trade/orders-algo-history",
		SecType:  utils.SecTypeSigned,
	}

	m2 := utils.Params{
		"instType": "SWAP",
		"ordType":  "conditional",
		"state":    "effective",
	}

	if s.symbol != nil {
		m2["instId"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m2["limit"] = *s.limit
	}
	if s.orderID != nil {
		// algo-history пагинация тоже идет по algoId/before-after
		m2["before"] = *s.orderID
	}

	r2.SetParams(m2)

	data2, _, err := s.callAPI(ctx, r2, opts...)
	if err != nil {
		return res, err
	}

	var answ2 struct {
		Result []futures_algoOrderHistory `json:"data"`
	}

	err = json.Unmarshal(data2, &answ2)
	if err != nil {
		return res, err
	}

	algoOut := convertAlgoOrdersHistoryOKX(answ2.Result)

	// -----------------------------------------
	// 3) Merge:
	//    - сначала algoId
	//    - потом algoClOrdId
	//    - потом attachAlgoClOrdId
	//    - если не нашли, добавляем отдельной строкой
	// -----------------------------------------
	indexByAlgoID := make(map[string]int, len(out))
	indexByAlgoClOrdID := make(map[string]int, len(out))
	indexByAttachAlgoClOrdID := make(map[string]int, len(out))

	for i := range out {
		if out[i].PositionID != "" {
			indexByAlgoID[out[i].PositionID] = i
		}
		if out[i].ClientOrderID != "" {
			indexByAlgoClOrdID[out[i].ClientOrderID] = i
		}
	}

	// В обычной history attachAlgoClOrdId отдельным полем,
	// поэтому храним его в карте из исходного ответа.
	for i, item := range answ1.Result {
		if item.AttachAlgoClOrdId != "" {
			indexByAttachAlgoClOrdID[item.AttachAlgoClOrdId] = i
		}
	}

	for _, a := range algoOut {
		if a.PositionID != "" {
			if idx, ok := indexByAlgoID[a.PositionID]; ok {
				if a.TpOrder {
					out[idx].TpOrder = true
				}
				if a.SlOrder {
					out[idx].SlOrder = true
				}
				continue
			}
		}

		if a.ClientOrderID != "" {
			if idx, ok := indexByAlgoClOrdID[a.ClientOrderID]; ok {
				if a.TpOrder {
					out[idx].TpOrder = true
				}
				if a.SlOrder {
					out[idx].SlOrder = true
				}
				continue
			}
			if idx, ok := indexByAttachAlgoClOrdID[a.ClientOrderID]; ok {
				if a.TpOrder {
					out[idx].TpOrder = true
				}
				if a.SlOrder {
					out[idx].SlOrder = true
				}
				continue
			}
		}

		out = append(out, a)
	}

	return out, nil
}

type futures_ordersHistory_Response struct {
	InstId            string `json:"instId"`
	OrdId             string `json:"ordId"`
	ClOrdId           string `json:"clOrdId"`
	AlgoId            string `json:"algoId"`
	AlgoClOrdId       string `json:"algoClOrdId"`
	AttachAlgoClOrdId string `json:"attachAlgoClOrdId"`

	Side      string `json:"side"`
	PosSide   string `json:"posSide"`
	Sz        string `json:"sz"`
	FillSz    string `json:"fillSz"`
	AccFillSz string `json:"accFillSz"`
	Px        string `json:"px"`
	AvgPx     string `json:"avgPx"`

	Fee    string `json:"fee"`
	FeeCcy string `json:"feeCcy"`
	Pnl    string `json:"pnl"`
	Lever  string `json:"lever"`

	OrdType string `json:"ordType"`
	State   string `json:"state"`
	TdMode  string `json:"tdMode"`
	CTime   string `json:"cTime"`
	UTime   string `json:"uTime"`
}

type futures_algoOrderHistory struct {
	AlgoId      string `json:"algoId"`
	AlgoClOrdId string `json:"algoClOrdId"`

	InstId   string `json:"instId"`
	InstType string `json:"instType"`
	OrdType  string `json:"ordType"`
	Side     string `json:"side"`
	PosSide  string `json:"posSide"`
	Sz       string `json:"sz"`
	State    string `json:"state"`

	ActualSide string `json:"actualSide"`
	ActualSz   string `json:"actualSz"`
	ActualPx   string `json:"actualPx"`

	TpTriggerPx string `json:"tpTriggerPx"`
	SlTriggerPx string `json:"slTriggerPx"`
	TriggerPx   string `json:"triggerPx"`

	TdMode string `json:"tdMode"`
	Lever  string `json:"lever"`
	CTime  string `json:"cTime"`
	UTime  string `json:"uTime"`
}

func convertOrdersHistoryOKX(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		if strings.ToLower(item.State) != "filled" {
			continue
		}

		hedgeMode := false
		posSide := item.PosSide
		if posSide == "net" || posSide == "" {
			if strings.ToUpper(item.Side) == "SELL" {
				posSide = "SHORT"
			} else {
				posSide = "LONG"
			}
		} else {
			hedgeMode = true
			posSide = strings.ToUpper(posSide)
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:  item.InstId,
			OrderID: item.OrdId,
			// В обычной history клиентский ID ордера оставляем как есть.
			// Для merge по algo используем PositionID как внутреннее поле связи.
			ClientOrderID:  item.ClOrdId,
			PositionID:     item.AlgoId,
			Side:           strings.ToUpper(item.Side),
			PositionSide:   posSide,
			PositionSize:   item.Sz,
			ExecutedSize:   item.AccFillSz,
			Price:          item.Px,
			ExecutedPrice:  item.AvgPx,
			RealisedProfit: item.Pnl,
			Fee:            item.Fee,
			FeeAsset:       item.FeeCcy,
			Leverage:       item.Lever,
			HedgeMode:      hedgeMode,
			MarginMode:     strings.ToUpper(item.TdMode),
			Type:           strings.ToUpper(item.OrdType),
			Status:         "FILLED",
			CreateTime:     utils.StringToInt64(item.CTime),
			UpdateTime:     utils.StringToInt64(item.UTime),
		})
	}

	return out
}

func convertAlgoOrdersHistoryOKX(in []futures_algoOrderHistory) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, it := range in {
		if strings.ToLower(it.State) != "effective" {
			continue
		}

		istp := it.TpTriggerPx != "" && it.TpTriggerPx != "0" && it.TpTriggerPx != "0.0"
		issl := it.SlTriggerPx != "" && it.SlTriggerPx != "0" && it.SlTriggerPx != "0.0"

		if !istp && !issl {
			continue
		}

		posSide := it.PosSide
		hedgeMode := false
		if posSide == "net" || posSide == "" {
			sideForDetect := it.ActualSide
			if sideForDetect == "" {
				sideForDetect = it.Side
			}
			if strings.ToUpper(sideForDetect) == "SELL" {
				posSide = "SHORT"
			} else {
				posSide = "LONG"
			}
		} else {
			hedgeMode = true
			posSide = strings.ToUpper(posSide)
		}

		side := it.ActualSide
		if side == "" {
			side = it.Side
		}

		price := ""
		if istp {
			price = it.TpTriggerPx
		} else if issl {
			price = it.SlTriggerPx
		} else {
			price = it.TriggerPx
		}

		execSz := it.ActualSz
		if execSz == "" || execSz == "0" {
			execSz = it.Sz
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol: it.InstId,
			// Отдельный fallback-ряд, если не сматчился с обычным filled order
			OrderID:       it.AlgoId,
			ClientOrderID: it.AlgoClOrdId,
			PositionID:    it.AlgoId,
			Side:          strings.ToUpper(side),
			PositionSide:  posSide,
			PositionSize:  it.Sz,
			ExecutedSize:  execSz,
			Price:         price,
			ExecutedPrice: it.ActualPx,
			Leverage:      it.Lever,
			HedgeMode:     hedgeMode,
			MarginMode:    strings.ToUpper(it.TdMode),
			Type:          "CONDITIONAL",
			Status:        "FILLED",
			CreateTime:    utils.StringToInt64(it.CTime),
			UpdateTime:    utils.StringToInt64(it.UTime),
			TpOrder:       istp,
			SlOrder:       issl,
		})
	}

	return out
}

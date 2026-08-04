package whitebit

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ================== FUTURES: GetInstrumentsInfo ==================

type futures_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getInstrumentsInfo) Symbol(symbol string) *futures_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *futures_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_InstrumentsInfo, err error) {
	// 1) Берём общую инфу по рынкам (spot + futures)
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v4/public/markets",
		SecType:  utils.SecTypeNone,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var markets []futures_instrumentsInfo
	if err := json.Unmarshal(data, &markets); err != nil {
		return res, err
	}

	// 2) Берём доп. инфу по фьючам (max_leverage и т.п.)
	rFut := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v4/public/futures",
		SecType:  utils.SecTypeNone,
	}

	dataFut, _, err := s.callAPI(ctx, rFut, opts...)
	if err != nil {
		return res, err
	}

	var futResp struct {
		Success bool                    `json:"success"`
		Message *string                 `json:"message"`
		Result  []whitebit_futuresExtra `json:"result"`
	}

	if err := json.Unmarshal(dataFut, &futResp); err != nil {
		return res, err
	}

	// map: TickerID (например, "BTC_PERP") -> max_leverage (string)
	levMap := make(map[string]string, len(futResp.Result))
	for _, f := range futResp.Result {
		levMap[f.TickerID] = strconv.Itoa(f.MaxLeverage)
	}

	// 3) Фильтруем только futures-рынки и, при необходимости, по symbol
	var filtered []futures_instrumentsInfo
	for _, m := range markets {
		// интересуют только markets, у которых type == "futures"
		if m.Type != "futures" {
			continue
		}

		// если пользователь запросил конкретный symbol
		if s.symbol != nil && *s.symbol != m.Name {
			continue
		}

		if lev, ok := levMap[m.Name]; ok {
			m.MaxLeverage = lev
		}

		filtered = append(filtered, m)
	}

	return s.convert.convertInstrumentsInfo(filtered), nil
}

// Структура под Market Info (фьючерсные рынки из /api/v4/public/markets)
type futures_instrumentsInfo struct {
	Name          string `json:"name"`      // "BTC_PERP"
	Stock         string `json:"stock"`     // "BTC"
	Money         string `json:"money"`     // "USDT"
	StockPrec     string `json:"stockPrec"` // precision для количества
	MoneyPrec     string `json:"moneyPrec"` // precision для цены
	FeePrec       string `json:"feePrec"`   // не используем, но оставим
	MinAmount     string `json:"minAmount"` // минимальное количество
	MinTotal      string `json:"minTotal"`  // минимальная нотионалка
	MaxTotal      string `json:"maxTotal"`  // не используем
	TradesEnabled bool   `json:"tradesEnabled"`
	IsCollateral  bool   `json:"isCollateral"`
	Type          string `json:"type"` // "spot" или "futures"

	// заполняем вручную из /api/v4/public/futures
	MaxLeverage string `json:"-"`
}

// Доп. структура под /api/v4/public/futures
type whitebit_futuresExtra struct {
	TickerID    string `json:"ticker_id"` // "BTC_PERP"
	MaxLeverage int    `json:"max_leverage"`
	// остальные поля нам здесь не нужны
}

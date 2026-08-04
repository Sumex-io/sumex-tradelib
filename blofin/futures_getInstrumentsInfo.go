package blofin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

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
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/market/instruments",
		SecType:  utils.SecTypeNone, // публичный метод, без подписи
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["instId"] = *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Data []futures_instrumentsInfo `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertInstrumentsInfo(answ.Data), nil
}

// структура под ответ Blofin /api/v1/market/instruments
// см. доку: Public Data → GET INSTRUMENTS
type futures_instrumentsInfo struct {
	InstId         string `json:"instId"`
	BaseCurrency   string `json:"baseCurrency"`
	QuoteCurrency  string `json:"quoteCurrency"`
	ContractValue  string `json:"contractValue"`
	ListTime       string `json:"listTime"`
	ExpireTime     string `json:"expireTime"`
	MaxLeverage    string `json:"maxLeverage"`
	MinSize        string `json:"minSize"`
	LotSize        string `json:"lotSize"`
	TickSize       string `json:"tickSize"`
	InstType       string `json:"instType"`
	ContractType   string `json:"contractType"`
	MaxLimitSize   string `json:"maxLimitSize"`
	MaxMarketSize  string `json:"maxMarketSize"`
	State          string `json:"state"`
	SettleCurrency string `json:"settleCurrency"` // есть в реальном ответе, см. твой лог
}

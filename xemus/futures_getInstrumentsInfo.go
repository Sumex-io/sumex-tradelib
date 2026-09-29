package xemus

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_getInstrumentsInfo struct {
	callAPI callAPIFunc
}

// Do maps the public symbol catalog (GET /v1/markets/symbols, a bare array, unsigned). perp-api
// caches it, so no client-side cache is kept. A row whose symbol is not PERP_<BASE>_<QUOTE> is
// skipped and logged rather than failing the whole catalog.
func (s *futures_getInstrumentsInfo) Do(ctx context.Context) (res []entity.Futures_InstrumentsInfo, err error) {
	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/markets/symbols"})
	if err != nil {
		return res, err
	}
	var rows []symbolRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return res, err
	}
	res = make([]entity.Futures_InstrumentsInfo, 0, len(rows))
	for _, row := range rows {
		info, err := toInstrumentInfo(row)
		if err != nil {
			log.Println(err)
			continue
		}
		res = append(res, info)
	}
	sortInstruments(res)
	return res, nil
}

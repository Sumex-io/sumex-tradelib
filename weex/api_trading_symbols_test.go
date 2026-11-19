package weex

import (
	"context"
	"net/http"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/utils"
)

func fakeCallAPI(t *testing.T, bodies map[string]string) func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) ([]byte, *http.Header, error) {
	return func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) ([]byte, *http.Header, error) {
		body, ok := bodies[r.Endpoint]
		if !ok {
			t.Fatalf("unexpected endpoint %s", r.Endpoint)
		}
		return []byte(body), nil, nil
	}
}

func TestFuturesInstrumentsKeepOnlyApiTradable(t *testing.T) {
	req := &futures_getInstrumentsInfo{callAPI: fakeCallAPI(t, map[string]string{
		"/capi/v3/market/exchangeInfo":      `{"symbols":[{"symbol":"BTCUSDT"},{"symbol":"NVDAUSDT"},{"symbol":"AAPLXUSDT"}]}`,
		"/capi/v3/market/apiTradingSymbols": `["BTCUSDT","NVDAUSDT"]`,
	})}

	got, err := req.Do(context.Background())
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if len(got) != 2 || got[0].Symbol != "BTCUSDT" || got[1].Symbol != "NVDAUSDT" {
		t.Fatalf("got %+v, want BTCUSDT and NVDAUSDT", got)
	}
}

func TestSpotInstrumentsFailOnEmptyApiTradingSymbols(t *testing.T) {
	req := &spot_getInstrumentsInfo{callAPI: fakeCallAPI(t, map[string]string{
		"/api/v3/exchangeInfo":      `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING"}]}`,
		"/api/v3/apiTradingSymbols": `[]`,
	})}

	if got, err := req.Do(context.Background()); err == nil {
		t.Fatalf("got %+v and no error, want an error", got)
	}
}

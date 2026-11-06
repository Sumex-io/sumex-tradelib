package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/utils"
)

func TestFuturesUserTradesParsesFillInBaseAsset(t *testing.T) {
	payload := []byte(`{"code":"200000","data":{"currentPage":1,"pageSize":50,"totalNum":1,"totalPage":1,"items":[{
		"symbol":"SOLUSDTM","tradeId":"1828954878212","orderId":"508147965953740800","side":"sell",
		"liquidity":"taker","price":"90.41","size":3,"value":"27.123","openFeePay":"0","closeFeePay":"0.0162738",
		"feeRate":"0.00060","feeCurrency":"USDT","marginMode":"CROSS","positionSide":"LONG","fee":"0.0162738",
		"settleCurrency":"USDT","orderType":"market","tradeType":"trade",
		"tradeTime":1793965118015000000,"createdAt":1793965118200
	}]}}`)

	var query url.Values
	s := &futures_userTrades{callAPI: func(_ context.Context, r *utils.Request, _ ...utils.RequestOption) ([]byte, *http.Header, error) {
		if r.Endpoint != "/api/v1/fills" {
			t.Fatalf("endpoint = %s", r.Endpoint)
		}
		query = r.Query
		return payload, nil, nil
	}}

	res, err := s.StartTime(1).EndTime(2).Limit(500).Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d trades", len(res))
	}

	got := res[0]
	if utils.StringToFloat(got.Qty) < 0.2999999 || utils.StringToFloat(got.Qty) > 0.3000001 {
		t.Errorf("qty = %s, want 0.3 base asset rather than 3 contracts", got.Qty)
	}
	if got.QuoteQty != "27.123" || got.Commission != "0.0162738" || got.CommissionAsset != "USDT" {
		t.Errorf("unexpected amounts: %+v", got)
	}
	if got.Side != "SELL" || got.PositionSide != "LONG" || got.Buyer || got.Maker {
		t.Errorf("unexpected flags: %+v", got)
	}
	if got.Time != 1793965118015 {
		t.Errorf("time = %d, want tradeTime in ms", got.Time)
	}
	if query.Get("startAt") != "1" || query.Get("endAt") != "2" || query.Get("pageSize") != "500" {
		t.Errorf("query = %v", query)
	}
}

func TestFuturesPositionsHistoryUsesEndpointParamNames(t *testing.T) {
	var query url.Values
	s := &futures_positionsHistory{callAPI: func(_ context.Context, r *utils.Request, _ ...utils.RequestOption) ([]byte, *http.Header, error) {
		query = r.Query
		return []byte(`{"code":"200000","data":{"items":[]}}`), nil, nil
	}}

	if _, err := s.StartTime(1).EndTime(2).Limit(200).Page(3).Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if query.Get("from") != "1" || query.Get("to") != "2" || query.Get("limit") != "200" || query.Get("pageId") != "3" {
		t.Errorf("query = %v", query)
	}
}

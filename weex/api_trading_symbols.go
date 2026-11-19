package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/utils"
)

// exchangeInfo also lists pairs that reject API orders (-1058); only apiTradingSymbols accept them.
func getApiTradingSymbols(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	endpoint string,
	opts ...utils.RequestOption,
) (map[string]struct{}, error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
		SecType:  utils.SecTypeNone,
	}

	data, _, err := callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var symbols []string
	if err := json.Unmarshal(data, &symbols); err != nil {
		return nil, err
	}
	if len(symbols) == 0 {
		return nil, errors.New("empty api trading symbols")
	}

	set := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		set[symbol] = struct{}{}
	}
	return set, nil
}

func keepApiTradable[T any](items []T, symbolOf func(T) string, tradable map[string]struct{}) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := tradable[symbolOf(item)]; ok {
			out = append(out, item)
		}
	}
	return out
}

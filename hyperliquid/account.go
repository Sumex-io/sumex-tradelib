package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

type cancelAction struct {
	Type    string `json:"type"`
	Cancels []struct {
		A int `json:"a"`
		O int `json:"o"`
	} `json:"cancels"`
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/exchange",
		SecType:  utils.SecTypeSigned,
	}

	var a cancelAction
	a.Type = "cancel"
	a.Cancels = append(a.Cancels, struct {
		A int `json:"a"`
		O int `json:"o"`
	}{A: 0, O: 0})

	b, _ := json.Marshal(a)
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	raw := strings.ToLower(string(data))

	if strings.Contains(raw, "l1 error") ||
		(strings.Contains(raw, "api wallet") && strings.Contains(raw, "does not exist")) ||
		strings.Contains(raw, "must deposit") {
		return res, &aPIError{Raw: data}
	}

	return s.convert.convertAccountInfo(hyperliquidAccountInfo{}), nil
}

type hyperliquidAccountInfo struct{}

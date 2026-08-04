package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===================GetAccountInfo==================
type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/account/config",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []accountInfo `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}
	return s.convert.convertAccountInfo(answ.Result[0]), nil
}

type accountInfo struct {
	UID                 string `json:"uid"`
	MainUID             string `json:"mainUid"`
	AcctLv              string `json:"acctLv"`
	AcctStpMode         string `json:"acctStpMode"`
	PosMode             string `json:"posMode"`
	AutoLoan            bool   `json:"autoLoan"`
	GreeksType          string `json:"greeksType"`
	Level               string `json:"level"`
	LevelTmp            string `json:"levelTmp"`
	CtIsoMode           string `json:"ctIsoMode"`
	MgnIsoMode          string `json:"mgnIsoMode"`
	RoleType            string `json:"roleType"`
	SpotRoleType        string `json:"spotRoleType"`
	OpAuth              string `json:"opAuth"`
	KycLv               string `json:"kycLv"`
	Label               string `json:"label"`
	Ip                  string `json:"ip"`
	Perm                string `json:"perm"`
	LiquidationGear     string `json:"liquidationGear"`
	EnableSpotBorrow    bool   `json:"enableSpotBorrow"`
	SpotBorrowAutoRepay bool   `json:"spotBorrowAutoRepay"`
	Type                string `json:"type"`
}

// ===================SignAuthStream==================

type signAuthStream struct {
	sec       string
	timeStamp *int64
}

func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = &timeStamp
	return s
}

func (s *signAuthStream) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.SignAuthStream, err error) {
	sf, err := utils.SignFunc(utils.KeyTypeHmacBase64)
	if err != nil {
		return res, err
	}

	t := int64(0)

	if s.timeStamp != nil {
		t = *s.timeStamp
	}

	raw := fmt.Sprintf("%dGET/users/self/verify", t)
	sign, err := sf(s.sec, raw)
	if err != nil {
		return res, err
	}
	res.Signature = *sign
	return res, nil
}

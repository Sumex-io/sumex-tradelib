package bingx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/v1/account/apiPermissions",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := accountInfo{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	res = s.convert.convertAccountInfo(answ)

	r.Endpoint = "/openApi/account/v1/uid"
	data, _, err = s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answUid := accountInfoUid{}
	err = json.Unmarshal(data, &answUid)
	if err != nil {
		return res, err
	}
	res.UID = utils.Int64ToString(answUid.Data.UID)
	return res, nil
}

type accountInfo struct {
	Note        string   `json:"note"`
	Permissions []int64  `json:"permissions"`
	IpAddresses []string `json:"ipAddresses"`
}

type accountInfoUid struct {
	Data struct {
		UID int64 `json:"uid"`
	} `json:"data"`
}

package hyperliquid

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Betarost/onetrades/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

type aPIError struct {
	Status   string          `json:"status,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
	Message  string          `json:"message,omitempty"`
	Raw      []byte          `json:"-"`
}

func (e aPIError) Error() string {
	if len(e.Raw) > 0 {
		return string(e.Raw)
	}
	if e.Message != "" {
		return e.Message
	}
	if len(e.Response) > 0 {
		return string(e.Response)
	}
	if e.Status != "" {
		return e.Status
	}
	return "hyperliquid api error"
}

func (c *SpotClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey

	opts = append(opts, createFullURL, createBody, createSign, createHeaders)
	if err := r.ParseRequest(opts...); err != nil {
		return []byte{}, &http.Header{}, err
	}

	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)
	c.debug("Headers %+v\n", r.Header)

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	req = req.WithContext(ctx)
	req.Header = r.Header
	c.debug("request: %#v\n", req)

	res, err := r.DoFunc(req)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	data, err = r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	defer func() {
		cerr := res.Body.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	c.debug("response status code: %d\n", res.StatusCode)
	c.debug("response body: %s\n", string(data))

	if res.StatusCode >= http.StatusBadRequest {
		return nil, &res.Header, aPIError{Raw: data}
	}

	var st struct {
		Status string `json:"status"`
	}
	if e := json.Unmarshal(data, &st); e == nil && strings.ToLower(st.Status) == "err" {
		return nil, &res.Header, aPIError{Raw: data}
	}

	return data, &res.Header, nil
}

func (c *FuturesClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey

	opts = append(opts, createFullURL, createBody, createSign, createHeaders)
	if err := r.ParseRequest(opts...); err != nil {
		return []byte{}, &http.Header{}, err
	}

	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)
	c.debug("Headers %+v\n", r.Header)

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	req = req.WithContext(ctx)
	req.Header = r.Header
	c.debug("request: %#v\n", req)

	res, err := r.DoFunc(req)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	data, err = r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	defer func() {
		cerr := res.Body.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	c.debug("response status code: %d\n", res.StatusCode)
	c.debug("response body: %s\n", string(data))

	if res.StatusCode >= http.StatusBadRequest {
		return nil, &res.Header, aPIError{Raw: data}
	}

	var st struct {
		Status string `json:"status"`
	}
	if e := json.Unmarshal(data, &st); e == nil && strings.ToLower(st.Status) == "err" {
		return nil, &res.Header, aPIError{Raw: data}
	}

	return data, &res.Header, nil
}

func createFullURL(r *utils.Request) error {
	r.FullURL = fmt.Sprintf("%s%s", r.BaseURL, r.Endpoint)
	return nil
}

func createBody(r *utils.Request) error {
	if r.BodyString != "" {
		r.Body = bytes.NewBufferString(r.BodyString)
		return nil
	}

	if r.Form != nil {
		bodyString := r.Form.Encode()
		if bodyString != "" {
			r.BodyString = bodyString
			r.Body = bytes.NewBufferString(bodyString)
			return nil
		}
	}

	r.Body = &bytes.Buffer{}
	return nil
}

func createHeaders(r *utils.Request) error {
	header := http.Header{}
	if r.Header != nil {
		header = r.Header.Clone()
	}

	if r.BodyString != "" && header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}

	r.Header = header
	return nil
}

func hlDebugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("HL_DEBUG_SIGN")))
	return v == "1" || v == "true" || v == "yes"
}

type hlExchangePayload struct {
	Action       json.RawMessage        `json:"action"`
	Nonce        int64                  `json:"nonce"`
	Signature    map[string]interface{} `json:"signature"`
	VaultAddress string                 `json:"vaultAddress,omitempty"`
}

func createSign(r *utils.Request) error {
	if r.SecType != utils.SecTypeSigned {
		r.TmpSig = ""
		r.TmpMemo = ""
		return nil
	}

	nonce := utils.CurrentTimestamp() - r.TimeOffset

	if strings.TrimSpace(r.BodyString) == "" {
		return fmt.Errorf("hyperliquid signed request requires r.BodyString to contain JSON action")
	}

	ordered, err := parseJSONOrdered([]byte(r.BodyString))
	if err != nil {
		return fmt.Errorf("hyperliquid: invalid action json: %v", err)
	}

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)

	if err := encodeOrderedValue(enc, ordered); err != nil {
		return fmt.Errorf("hyperliquid: msgpack encode action failed: %v", err)
	}

	actionMsgpack := buf.Bytes()
	actionMsgpack = convertStr16ToStr8(actionMsgpack)

	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))

	preimage := make([]byte, 0, len(actionMsgpack)+8+1+20)
	preimage = append(preimage, actionMsgpack...)
	preimage = append(preimage, nonceBytes...)

	vaultAddr := strings.TrimSpace(r.TmpMemo)
	if vaultAddr != "" {
		preimage = append(preimage, 0x01)
		vb, e := hexTo20Bytes(vaultAddr)
		if e != nil {
			return fmt.Errorf("hyperliquid: invalid vault address: %v", e)
		}
		preimage = append(preimage, vb...)
	} else {
		preimage = append(preimage, 0x00)
	}

	connectionID := crypto.Keccak256(preimage)

	priv, err := parseEcdsaPrivateKey(r.TmpSig)
	if err != nil {
		return fmt.Errorf("hyperliquid: invalid private key: %v", err)
	}

	source := "a"
	if strings.Contains(strings.ToLower(r.BaseURL), "testnet") {
		source = "b"
	}

	sig, eip712Hash, derivedAgent, recoveredAgent, err := signAgentTypedDataWithDebug(priv, source, connectionID)
	if err != nil {
		return fmt.Errorf("hyperliquid: sign typed data failed: %v", err)
	}

	if hlDebugEnabled() {
		log.Printf("[HL SIGN] actionJSON=%s", r.BodyString)
		log.Printf("[HL SIGN] nonce=%d", nonce)
		log.Printf("[HL SIGN] vaultAddr=%q", vaultAddr)
		log.Printf("[HL SIGN] actionMsgpack=%x", actionMsgpack)
		log.Printf("[HL SIGN] preimage=%x", preimage)
		log.Printf("[HL SIGN] connectionId=0x%x", connectionID)
		log.Printf("[HL SIGN] eip712Hash=0x%x", eip712Hash)
		log.Printf("[HL SIGN] derivedAgent=%s recoveredAgent=%s", derivedAgent, recoveredAgent)
		log.Printf("[HL SIGN] sig=%v", sig)
	}

	payload := hlExchangePayload{
		Action:    json.RawMessage(r.BodyString),
		Nonce:     nonce,
		Signature: sig,
	}
	if vaultAddr != "" {
		payload.VaultAddress = normalizeHexAddress(vaultAddr)
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("hyperliquid: marshal signed payload failed: %v", err)
	}

	r.BodyString = string(b)
	r.Body = bytes.NewBuffer(b)
	r.TmpApi = ""
	r.TmpSig = ""
	r.TmpMemo = ""
	return nil
}

func convertStr16ToStr8(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if data[i] == 0xDA && i+2 < len(data) {
			l := (int(data[i+1]) << 8) | int(data[i+2])
			if l < 256 {
				out = append(out, 0xD9, byte(l))
				i += 3
				if i+l <= len(data) {
					out = append(out, data[i:i+l]...)
					i += l
				}
				continue
			}
		}
		out = append(out, data[i])
		i++
	}
	return out
}

func signAgentTypedDataWithDebug(priv *ecdsa.PrivateKey, source string, connectionId32 []byte) (map[string]interface{}, []byte, string, string, error) {
	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Agent": []apitypes.Type{
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
		},
		PrimaryType: "Agent",
		Domain: apitypes.TypedDataDomain{
			Name:              "Exchange",
			Version:           "1",
			ChainId:           math.NewHexOrDecimal256(1337),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Message: apitypes.TypedDataMessage{
			"source":       source,
			"connectionId": "0x" + hex.EncodeToString(connectionId32),
		},
	}

	msgHash, err := td.HashStruct(td.PrimaryType, td.Message)
	if err != nil {
		return nil, nil, "", "", err
	}
	domainSep, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		return nil, nil, "", "", err
	}

	raw := make([]byte, 0, 2+32+32)
	raw = append(raw, 0x19, 0x01)
	raw = append(raw, domainSep...)
	raw = append(raw, msgHash...)
	finalHash := crypto.Keccak256(raw)

	sig65, err := crypto.Sign(finalHash, priv)
	if err != nil {
		return nil, nil, "", "", err
	}

	pub, err := crypto.SigToPub(finalHash, sig65)
	if err != nil {
		return nil, nil, "", "", err
	}
	recovered := crypto.PubkeyToAddress(*pub).Hex()
	derived := crypto.PubkeyToAddress(priv.PublicKey).Hex()

	rb := sig65[:32]
	sb := sig65[32:64]
	vb := sig65[64] + 27

	out := map[string]interface{}{
		"r": "0x" + hex.EncodeToString(rb),
		"s": "0x" + hex.EncodeToString(sb),
		"v": int(vb),
	}
	return out, finalHash, derived, recovered, nil
}

func parseEcdsaPrivateKey(pk string) (*ecdsa.PrivateKey, error) {
	pk = strings.TrimSpace(pk)
	pk = strings.TrimPrefix(pk, "0x")
	if len(pk) != 64 {
		return nil, fmt.Errorf("expected 32-byte hex private key (64 hex chars), got len=%d", len(pk))
	}
	b, err := hex.DecodeString(pk)
	if err != nil {
		return nil, err
	}
	return crypto.ToECDSA(b)
}

func normalizeHexAddress(addr string) string {
	a := strings.TrimSpace(strings.ToLower(addr))
	if a == "" {
		return ""
	}
	if !strings.HasPrefix(a, "0x") {
		a = "0x" + a
	}
	return a
}

func hexTo20Bytes(addr string) ([]byte, error) {
	a := normalizeHexAddress(addr)
	a = strings.TrimPrefix(a, "0x")
	if len(a) != 40 {
		return nil, fmt.Errorf("expected 20-byte address (40 hex chars), got len=%d", len(a))
	}
	return hex.DecodeString(a)
}

type orderedValue interface{}

type orderedMap struct {
	keys []string
	vals map[string]orderedValue
}

type orderedArray []orderedValue

func (om orderedMap) EncodeMsgpack(enc *msgpack.Encoder) error {
	if err := enc.EncodeMapLen(len(om.keys)); err != nil {
		return err
	}
	for _, k := range om.keys {
		if err := enc.EncodeString(k); err != nil {
			return err
		}
		if err := encodeOrderedValue(enc, om.vals[k]); err != nil {
			return err
		}
	}
	return nil
}

func (oa orderedArray) EncodeMsgpack(enc *msgpack.Encoder) error {
	if err := enc.EncodeArrayLen(len(oa)); err != nil {
		return err
	}
	for _, it := range oa {
		if err := encodeOrderedValue(enc, it); err != nil {
			return err
		}
	}
	return nil
}

func encodeOrderedValue(enc *msgpack.Encoder, v orderedValue) error {
	switch t := v.(type) {
	case orderedMap:
		return t.EncodeMsgpack(enc)
	case orderedArray:
		return t.EncodeMsgpack(enc)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			if i >= int64(int(^uint(0)>>1))*-1 && i <= int64(int(^uint(0)>>1)) {
				return enc.EncodeInt(int64(i))
			}
			return enc.EncodeInt64(i)
		}
		if f, err := t.Float64(); err == nil {
			return enc.EncodeFloat64(f)
		}
		return enc.EncodeString(t.String())
	case string:
		return enc.EncodeString(t)
	case bool:
		return enc.EncodeBool(t)
	case nil:
		return enc.EncodeNil()
	default:
		return enc.Encode(t)
	}
}

func parseJSONOrdered(b []byte) (orderedValue, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return parseToken(dec, tok)
}

func parseToken(dec *json.Decoder, tok json.Token) (orderedValue, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			om := orderedMap{keys: []string{}, vals: map[string]orderedValue{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				ks, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("expected string key, got %T", kt)
				}
				vt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				vv, err := parseToken(dec, vt)
				if err != nil {
					return nil, err
				}
				om.keys = append(om.keys, ks)
				om.vals[ks] = vv
			}
			_, err := dec.Token()
			if err != nil {
				return nil, err
			}
			return om, nil
		case '[':
			var arr orderedArray
			for dec.More() {
				vt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				vv, err := parseToken(dec, vt)
				if err != nil {
					return nil, err
				}
				arr = append(arr, vv)
			}
			_, err := dec.Token()
			if err != nil {
				return nil, err
			}
			return arr, nil
		default:
			return nil, fmt.Errorf("unexpected delim %q", t)
		}
	case string:
		return t, nil
	case bool:
		return t, nil
	case nil:
		return nil, nil
	case json.Number:
		return t, nil
	default:
		return nil, fmt.Errorf("unexpected token type %T", tok)
	}
}

func toChecksumAddress(addr string) string {
	return common.HexToAddress(normalizeHexAddress(addr)).Hex()
}

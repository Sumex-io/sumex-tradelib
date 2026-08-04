package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
)

func CurrentTimestamp() int64 {
	return int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
}

func CurrentSecondsTimestamp() int64 {
	return int64(time.Millisecond) * time.Now().UnixMilli() / int64(time.Second)
}

func StringToFloat(num string) float64 {
	newNum, _ := strconv.ParseFloat(strings.Replace(num, ",", ".", -1), 64)
	return newNum
}

func StringToInt64(num string) int64 {
	newNum, _ := strconv.ParseInt(num, 10, 64)
	return newNum
}

func StringToInt(num string) int {
	newNum, _ := strconv.ParseInt(num, 10, 0)
	return int(newNum)
}

func Int64ToString(num int64) string {
	return fmt.Sprintf("%d", num)
}

func IntToString(num int) string {
	return fmt.Sprintf("%d", num)
}

func TimestampMilliToDateFormat(stamp int64, format string) string {
	tHisory := time.UnixMilli(stamp).UTC()
	hisNow := time.Date(tHisory.Year(), tHisory.Month(), tHisory.Day(), tHisory.Hour(), tHisory.Minute(), tHisory.Second(), 0, tHisory.Location())
	return hisNow.Format(format)
}

func FloatToStringAll(num float64) string {
	return strconv.FormatFloat(num, 'f', -1, 64)
}

func GetPrecisionFromStr(s string) (pr string) {
	pr = "0"
	sp := strings.Split(s, ".")
	if len(sp) > 1 {
		pr = fmt.Sprintf("%d", len(sp[1]))
	}
	return pr
}

func PrecisionFormatString(ret string, scale string) string {
	sp := strings.Split(ret, ".")
	precision := StringToInt(scale)
	if precision == 0 {
		ret = sp[0]
	} else if len(sp) == 2 && len(sp[1]) > precision {
		ret = sp[0] + "." + sp[1][:precision]
	}
	return ret
}

func PrecisionFormatFloat64(num float64, scale string) string {
	ret := strconv.FormatFloat(num, 'f', -1, 64)
	precision := StringToInt(scale)
	sp := strings.Split(ret, ".")
	if precision == 0 {
		ret = sp[0]
	} else if len(sp) == 2 && len(sp[1]) > precision {
		ret = sp[0] + "." + sp[1][:precision]
	}
	return ret
}

func NewJSON(data []byte) (j *simplejson.Json, err error) {
	j, err = simplejson.NewJson(data)
	if err != nil {
		return nil, err
	}
	return j, nil
}

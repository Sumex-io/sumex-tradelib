package utils

import (
	"github.com/Betarost/onetrades/entity"
	"github.com/MicahParks/go-rsi/v2"
)

func CalculateRSI(candles []entity.Futures_MarketCandle, period int, limit int) (rsiArr []entity.RSI) {
	close := []float64{}
	times := []int64{}

	for _, k := range candles {
		if k.Complete {
			close = append(close, StringToFloat(k.ClosePrice))
			times = append(times, k.Time)
		}
	}

	if len(close) <= period {
		return rsiArr
	}

	const initialLength = rsi.DefaultPeriods + 1
	initial := close[:initialLength]
	r, result := rsi.New(initial)
	rsiArr = append(rsiArr, entity.RSI{
		Value:      result,
		ClosePrice: close[period],
		Time:       times[period],
	})

	remaining := close[initialLength:]
	for i, next := range remaining {
		result = r.Calculate(next)
		rsiArr = append(rsiArr, entity.RSI{
			Value:      result,
			ClosePrice: close[initialLength+i],
			Time:       times[initialLength+i],
		})
	}
	return rsiArr[len(rsiArr)-limit:]
}

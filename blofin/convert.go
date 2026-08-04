package blofin

import (
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===================converts==================

type account_converts struct{}

func (c *account_converts) convertAccountInfo(in accountInfo) (out entity.AccountInformation) {
	out.UID = in.UID
	out.Label = in.APIName

	// если несколько IP — склеиваем через запятую
	if len(in.IPs) > 0 {
		out.IP = strings.Join(in.IPs, ",")
	}

	// По документации:
	// readOnly = 1  -> только чтение
	// readOnly = 0  -> чтение + запись (trade)
	out.CanRead = true // наличие валидного API-ключа предполагает как минимум READ
	if in.ReadOnly == 0 {
		out.CanTrade = true
	} else {
		out.CanTrade = false
	}

	// Про TRANSFER в этом ответе информации нет, ставим по умолчанию false.
	out.CanTransfer = false

	// Blofin API (в onetrades v1) мы используем только для фьючерсов
	out.PermSpot = false
	out.PermFutures = true

	return out
}

// ==========Futures===============
type futures_converts struct{}

// Конвертация Blofin → унифицированный Futures_InstrumentsInfo
func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         item.InstId,
			Base:           item.BaseCurrency,  // базовая валюта (BTC в BTC-USDT)
			Quote:          item.QuoteCurrency, // котируемая (обычно USDT)
			MinQty:         item.MinSize,       // минимальный size в КОНТРАКТАХ
			MinNotional:    "",                 // Blofin не отдаёт minNotional — оставляем пустым
			PricePrecision: utils.GetPrecisionFromStr(item.TickSize),
			SizePrecision:  utils.GetPrecisionFromStr(item.MinSize), // шаг size в контрактах
			State:          strings.ToUpper(item.State),
			MaxLeverage:    item.MaxLeverage,
			Multiplier:     "1",
			ContractSize:   item.ContractValue,
			IsSizeContract: true,
		})
	}

	return out
}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, i := range in {
		for _, item := range i.Details {
			out = append(out, entity.FuturesBalance{
				Asset:            item.Currency,
				Balance:          item.Balance,
				Equity:           item.Equity,
				Available:        item.Available,             // можно заменить на AvailableEquity, если решишь, что логичнее
				UnrealizedProfit: item.IsolatedUnrealizedPnl, // PnL по деривативам
			})
		}
	}

	return out
}

func (c *futures_converts) convertMarginMode(in futures_marginMode) (out entity.Futures_MarginMode) {
	mode := strings.ToUpper(in.MarginMode)

	switch mode {
	case "CROSS":
		out.MarginMode = string(entity.MarginModeTypeCross)
	case "ISOLATED":
		out.MarginMode = string(entity.MarginModeTypeIsolated)
	default:
		// на всякий случай, если биржа когда-то добавит что-то ещё
		out.MarginMode = mode
	}

	return out
}

func (c *futures_converts) convertPositionMode(in futures_positionMode) (out entity.Futures_PositionsMode) {
	// По доке:
	// net_mode        -> one-way (hedgeMode = false)
	// long_short_mode -> hedge mode (hedgeMode = true)

	switch in.PositionMode {
	case "long_short_mode":
		out.HedgeMode = true
	case "net_mode":
		out.HedgeMode = false
	default:
		// на всякий случай — если биржа придумает новый режим, оставим false
		out.HedgeMode = false
	}

	return out
}

// Вверху файла не забудь:
// import "strings"

func (c *futures_converts) convertLeverage(in []futures_leverage) (out entity.Futures_Leverage) {
	if len(in) == 0 {
		return out
	}

	// Если отдали одну запись — просто дублируем плечо в long/short
	if len(in) == 1 {
		out.Symbol = in[0].InstId
		out.Leverage = in[0].Leverage
		out.LongLeverage = in[0].Leverage
		out.ShortLeverage = in[0].Leverage
		return out
	}

	// Хедж-режим: потенциально две записи (long / short)
	out.Symbol = in[0].InstId

	for _, item := range in {
		switch strings.ToLower(item.PositionSide) {
		case "long":
			out.LongLeverage = item.Leverage
		case "short":
			out.ShortLeverage = item.Leverage
		case "net":
			// В one-way режиме просто используем как общее плечо
			if out.Leverage == "" {
				out.Leverage = item.Leverage
			}
		}
	}

	// Если обе стороны заданы — выбираем максимальное плечо как "общее"
	if out.LongLeverage != "" || out.ShortLeverage != "" {
		longLev := utils.StringToInt64(out.LongLeverage)
		shortLev := utils.StringToInt64(out.ShortLeverage)

		if longLev >= shortLev {
			out.Leverage = out.LongLeverage
		} else {
			out.Leverage = out.ShortLeverage
		}
	}

	// На всякий случай fallback: если по каким-то причинам не выставили Leverage
	if out.Leverage == "" && len(in) > 0 {
		out.Leverage = in[0].Leverage
		if out.LongLeverage == "" {
			out.LongLeverage = out.Leverage
		}
		if out.ShortLeverage == "" {
			out.ShortLeverage = out.Leverage
		}
	}

	return out
}

func (c *futures_converts) convertPlaceOrder(in []placeOrder_Response) (out []entity.PlaceOrder) {
	if len(in) == 0 {
		return out
	}

	now := time.Now().UTC().UnixMilli()

	for _, item := range in {
		out = append(out, entity.PlaceOrder{
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			// позиционный ID Blofin в этом ответе не даёт, поэтому оставляем пустым
			Ts: now,
		})
	}
	return out
}

func (c *futures_converts) convertPositions(in []futures_Position) (out []entity.Futures_Positions) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		// Определяем сторону позиции
		positionSide := "LONG"
		posVal := utils.StringToFloat(item.Positions)

		if strings.ToLower(item.PositionSide) == "net" {
			if posVal < 0 {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.PositionSide)
		}

		// HedgeMode: true, если не net (т.е. по идее раздельные long/short)
		hedgeMode := strings.ToLower(item.PositionSide) != "net"

		out = append(out, entity.Futures_Positions{
			Symbol:           item.InstID,
			PositionSide:     positionSide,
			PositionSize:     item.Positions,
			Leverage:         item.Leverage,
			PositionID:       item.PositionID,
			EntryPrice:       item.AveragePrice,
			MarkPrice:        item.MarkPrice,
			UnRealizedProfit: item.UnrealizedPnl,
			RealizedProfit:   "", // в этом эндпоинте Blofin не отдает realizedPnl
			Notional:         "", // нет явного notional; при желании можно считать отдельно
			HedgeMode:        hedgeMode,
			MarginMode:       strings.ToUpper(item.MarginMode),
			CreateTime:       utils.StringToInt64(item.CreateTime),
			UpdateTime:       utils.StringToInt64(item.UpdateTime),
		})
	}

	return out
}

func (c *futures_converts) convertOrderList(answ []futures_orderList) (res []entity.Futures_OrdersList) {
	for _, item := range answ {
		// Определяем positionSide так же, как на OKX:
		// - если net, то смотрим по side (buy -> LONG, sell -> SHORT)
		// - иначе берём то, что отдала биржа
		positionSide := "LONG"
		if strings.ToLower(item.PositionSide) == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		} else if item.PositionSide != "" {
			positionSide = strings.ToUpper(item.PositionSide)
		}

		res = append(res, entity.Futures_OrdersList{
			Symbol:        item.InstId,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			PositionID:    item.PositionId,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  positionSide,
			PositionSize:  item.Size,
			ExecutedSize:  item.FilledSize,
			Price:         item.Price,
			Leverage:      item.Leverage,
			Type:          strings.ToUpper(item.OrderType),
			Status:        strings.ToUpper(item.State),
			MarginMode:    strings.ToUpper(item.MarginMode),
			CreateTime:    utils.StringToInt64(item.CreateTime),
			UpdateTime:    utils.StringToInt64(item.UpdateTime),
		})
	}
	return res
}

func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		status := strings.ToUpper(item.State)

		// Оставляем только FILLED, как ты уже сделал через state=filled + фильтр
		if status != "FILLED" {
			continue
		}

		hedgeMode := false
		posSide := item.PositionSide

		if strings.ToLower(posSide) == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				posSide = "SHORT"
			} else {
				posSide = "LONG"
			}
		} else {
			hedgeMode = true
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:         item.InstId,
			OrderID:        item.OrderId,
			ClientOrderID:  item.ClientOrderId,
			PositionID:     "",
			Side:           strings.ToUpper(item.Side),
			PositionSide:   strings.ToUpper(posSide),
			PositionSize:   item.Size,
			ExecutedSize:   item.FilledSize,
			Price:          item.Price,
			ExecutedPrice:  item.AveragePrice, // ✅ теперь берём корректное поле
			RealisedProfit: item.Pnl,          // ✅ pnl из JSON
			Fee:            item.Fee,
			Leverage:       item.Leverage,
			HedgeMode:      hedgeMode,
			MarginMode:     strings.ToUpper(item.MarginMode),
			Type:           strings.ToUpper(item.OrderType),
			Status:         status,
			CreateTime:     utils.StringToInt64(item.CreateTime),
			UpdateTime:     utils.StringToInt64(item.UpdateTime),
		})
	}

	return out
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		marginMode := strings.ToUpper(item.MarginMode)
		switch strings.ToLower(item.MarginMode) {
		case "cross":
			marginMode = string(entity.MarginModeTypeCross)
		case "isolated":
			marginMode = string(entity.MarginModeTypeIsolated)
		}

		positionSide := strings.ToUpper(item.PositionSide)

		// В one-way/net режиме Blofin отдаёт positionSide = net.
		// Для единого формата приводим к LONG/SHORT через side:
		// buy  -> LONG
		// sell -> SHORT
		if strings.ToLower(item.PositionSide) == "net" {
			switch strings.ToLower(item.Side) {
			case "buy":
				positionSide = "LONG"
			case "sell":
				positionSide = "SHORT"
			}
		}

		out = append(out, entity.Futures_PositionsHistory{
			Symbol:              item.InstId,
			PositionID:          item.PositionId,
			PositionSide:        positionSide,
			PositionAmt:         item.MaxPositions,
			ExecutedPositionAmt: item.ClosePositions,
			AvgPrice:            item.OpenAveragePrice,
			ExecutedAvgPrice:    item.CloseAveragePrice,
			RealisedProfit:      item.RealizedPnl,
			Fee:                 item.Fee,
			Leverage:            item.Leverage,
			Funding:             "", // Blofin positions-history funding не отдаёт
			MarginMode:          marginMode,
			CreateTime:          utils.StringToInt64(item.CreateTime),
			UpdateTime:          utils.StringToInt64(item.UpdateTime),
		})
	}

	return out
}

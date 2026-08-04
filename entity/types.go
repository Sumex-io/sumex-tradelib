package entity

type PositionModeType string
type PositionSideType string
type SideType string
type OrderType string
type TimeFrameType string
type PositionStatusType string
type MarginModeType string

const (
	PositionStatusTypeClosePartially       PositionStatusType = "CLOSE_PARTIALLY"
	PositionStatusTypeCloseAll             PositionStatusType = "CLOSE_ALL"
	PositionStatusTypeLiquidation          PositionStatusType = "LIQUIDATION"
	PositionStatusTypeLiquidationPartially PositionStatusType = "LIQUIDATION_PARTIALLY"
	PositionStatusTypeAdl                  PositionStatusType = "ADL"

	PositionModeTypeOneWay PositionModeType = "ONE_WAY"
	PositionModeTypeHedge  PositionModeType = "HEDGE"

	MarginModeTypeCross    MarginModeType = "CROSS"
	MarginModeTypeIsolated MarginModeType = "ISOLATED"

	PositionSideTypeLong  PositionSideType = "LONG"
	PositionSideTypeShort PositionSideType = "SHORT"
	PositionSideTypeBoth  PositionSideType = "BOTH"

	SideTypeBuy  SideType = "BUY"
	SideTypeSell SideType = "SELL"

	OrderTypeLimit      OrderType = "LIMIT"
	OrderTypeMarket     OrderType = "MARKET"
	OrderTypeTakeProfit OrderType = "TAKE_PROFIT"
	OrderTypeStop       OrderType = "STOP"

	TimeFrameType1m  TimeFrameType = "1m"
	TimeFrameType3m  TimeFrameType = "3m"
	TimeFrameType5m  TimeFrameType = "5m"
	TimeFrameType15m TimeFrameType = "15m"
	TimeFrameType30m TimeFrameType = "30m"
	TimeFrameType1H  TimeFrameType = "1H"
	TimeFrameType2H  TimeFrameType = "2H"
	TimeFrameType4H  TimeFrameType = "4H"
	TimeFrameType6H  TimeFrameType = "6H"
	TimeFrameType8H  TimeFrameType = "8H"
	TimeFrameType12H TimeFrameType = "12H"
	TimeFrameType1D  TimeFrameType = "1D"
)

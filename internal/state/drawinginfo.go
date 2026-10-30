package state

import (
	"math"

	"github.com/bluemir/monosketch/internal/geo"
)

// DrawingInfo converts between canvas pixels and board cells. JS sets it when the canvas
// size, the font or the scroll offset changes.
type DrawingInfo struct {
	OffsetPx     geo.Point `json:"offsetPx"`
	CellSizePx   geo.SizeF `json:"cellSizePx"`
	CanvasSizePx geo.Size  `json:"canvasSizePx"`
}

var defaultDrawingInfo = DrawingInfo{
	CellSizePx:   geo.SizeF{Width: 1, Height: 1},
	CanvasSizePx: geo.Size{Width: 1, Height: 1},
}

// BoardBound is the board area shown on the canvas.
func (d DrawingInfo) BoardBound() geo.Rect {
	// Truncated toward zero, same as the original.
	boardOffsetRow := int(float64(-d.OffsetPx.Top) / d.CellSizePx.Height)
	boardOffsetColumn := int(float64(-d.OffsetPx.Left) / d.CellSizePx.Width)
	rowCount := int(math.Ceil(float64(d.CanvasSizePx.Height) / d.CellSizePx.Height))
	columnCount := int(math.Ceil(float64(d.CanvasSizePx.Width) / d.CellSizePx.Width))
	return geo.RectByLTWH(boardOffsetColumn, boardOffsetRow, columnCount, rowCount)
}

func (d DrawingInfo) toXPx(column float64) float64 {
	return math.Floor(float64(d.OffsetPx.Left) + d.CellSizePx.Width*column)
}

func (d DrawingInfo) toYPx(row float64) float64 {
	return math.Floor(float64(d.OffsetPx.Top) + d.CellSizePx.Height*row)
}

func (d DrawingInfo) toWidthPx(width float64) float64 {
	return math.Floor(d.CellSizePx.Width * width)
}

func (d DrawingInfo) toHeightPx(height float64) float64 {
	return math.Floor(d.CellSizePx.Height * height)
}

func (d DrawingInfo) toBoardPoint(pointPx geo.Point) geo.Point {
	return geo.Point{
		Left: int(math.Floor(float64(pointPx.Left-d.OffsetPx.Left) / d.CellSizePx.Width)),
		Top:  int(math.Floor(float64(pointPx.Top-d.OffsetPx.Top) / d.CellSizePx.Height)),
	}
}

// toBoardPointF rounds to 1 decimal place.
func (d DrawingInfo) toBoardPointF(pointPx geo.Point) geo.PointF {
	return geo.PointF{
		Left: math.Round(float64(pointPx.Left-d.OffsetPx.Left)/d.CellSizePx.Width*10) / 10,
		Top:  math.Round(float64(pointPx.Top-d.OffsetPx.Top)/d.CellSizePx.Height*10) / 10,
	}
}

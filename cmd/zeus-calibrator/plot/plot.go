// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package plot

import (
	"fmt"
	"image"
	"log/slog"
	"math"

	. "github.com/gizak/termui/v3"
)

type PointF struct {
	X, Y float64
}

func PtF(x, y float64) PointF {
	return PointF{X: x, Y: y}
}

type RectangleF struct {
	Min, Max PointF
}

func RectF(x0, y0, x1, y1 float64) RectangleF {
	return RectangleF{Min: PointF{x0, y0}, Max: PointF{x1, y1}}
}

func (r RectangleF) Dx() float64 {
	return r.Max.X - r.Min.X
}

func (r RectangleF) Dy() float64 {
	return r.Max.Y - r.Min.Y
}

// Plot has two modes: line(default) and scatter.
// Plot also has two marker types: braille(default) and dot.
// A single braille character is a 2x4 grid of dots, so using braille
// gives 2x X resolution and 4x Y resolution over dot mode.
type Plot struct {
	Block

	XData      []float64
	YData      [][]float64
	DataLabels []string
	MaxVal     float64
	MinVal     float64

	LineColors []Color
	AxesColor  Color // TODO
	ShowAxes   bool

	DotMarkerRune rune

	axisLimits RectangleF
}

const (
	xAxisLabelsHeight = 1
	yAxisLabelsWidth  = 4
	xAxisLabelsGap    = 5
	yAxisLabelsGap    = 1
)

func NewPlot() *Plot {
	return &Plot{
		Block:      *NewBlock(),
		LineColors: Theme.Plot.Lines,
		AxesColor:  Theme.Plot.Axes,
		XData:      nil,
		YData:      [][]float64{},
		ShowAxes:   true,
	}
}

func (p *Plot) project(pt PointF, drawArea image.Rectangle) image.Point {
	slog.Info("point", "pt", pt, "drawArea", drawArea, "lims", p.axisLimits)
	return image.Pt(
		int((pt.X-p.axisLimits.Min.X)/p.axisLimits.Dx()*float64(drawArea.Dx()-1))+drawArea.Min.X,
		int((pt.Y-p.axisLimits.Min.Y)/p.axisLimits.Dy()*float64(drawArea.Dy()-1))+drawArea.Min.Y,
	)
}

func (p *Plot) foreachPoint(y []float64, drawArea image.Rectangle, do func(pt image.Point)) {
	inc := len(y)/drawArea.Dx() + 1
	stop := len(y)

	if len(p.XData) > 0 {
		stop = min(stop, len(p.XData))
	}

	for j := 0; j < stop; j += inc {
		x := float64(j)
		if j < len(p.XData) {
			x = p.XData[j]
		}

		do(p.project(PtF(x, y[j]), drawArea))
	}
}

func (self *Plot) renderBraille(buf *Buffer, drawArea image.Rectangle) {
	canvas := NewCanvas()
	canvas.Rectangle = drawArea

	for i, line := range self.YData {
		var previous *image.Point
		self.foreachPoint(line, drawArea, func(pt image.Point) {
			if previous != nil {
				slog.Info("draw line", "A", *previous, "B", pt)
				canvas.SetLine(
					image.Pt(previous.X*2, previous.Y*4),
					image.Pt(pt.X*2, pt.Y*4),
					SelectColor(self.LineColors, i),
				)
			}
			previous = &image.Point{X: pt.X, Y: pt.Y}
		})
		slog.Info("drawing done", "i", i)
	}

	canvas.Draw(buf)
}

func (self *Plot) plotAxes(buf *Buffer) {
	// draw origin cell
	buf.SetCell(
		NewCell(BOTTOM_LEFT, NewStyle(ColorWhite)),
		image.Pt(self.Inner.Min.X+yAxisLabelsWidth, self.Inner.Max.Y-xAxisLabelsHeight-1),
	)
	// draw x axis line
	for i := yAxisLabelsWidth + 1; i < self.Inner.Dx(); i++ {
		buf.SetCell(
			NewCell(HORIZONTAL_DASH, NewStyle(ColorWhite)),
			image.Pt(i+self.Inner.Min.X, self.Inner.Max.Y-xAxisLabelsHeight-1),
		)
	}
	// draw y axis line
	for i := 0; i < self.Inner.Dy()-xAxisLabelsHeight-1; i++ {
		buf.SetCell(
			NewCell(VERTICAL_DASH, NewStyle(ColorWhite)),
			image.Pt(self.Inner.Min.X+yAxisLabelsWidth, i+self.Inner.Min.Y),
		)
	}
	// draw x axis labels
	// draw rest
	maxXi := self.Inner.Dy() - yAxisLabelsWidth - 1

	minX, maxX := self.getXRange()
	for i := 0; i < maxXi; i += xAxisLabelsGap {
		x := float64(i)*(maxX-minX) + minX
		label := fmt.Sprintf(
			"%.1f",
			x,
		)
		buf.SetString(
			label,
			NewStyle(ColorWhite),
			image.Pt(i+yAxisLabelsWidth, self.Inner.Max.Y-1),
		)
	}
	// draw y axis labels
	verticalScale := self.axisLimits.Dy() / float64(self.Inner.Dy()-xAxisLabelsHeight-1)
	for i := 0; i*(yAxisLabelsGap+1) < self.Inner.Dy()-1; i++ {
		buf.SetString(
			fmt.Sprintf("%.2f", float64(i)*verticalScale*(yAxisLabelsGap+1)+self.axisLimits.Min.Y),
			NewStyle(ColorWhite),
			image.Pt(self.Inner.Min.X, self.Inner.Max.Y-(i*(yAxisLabelsGap+1))-2),
		)
	}
}

func (p *Plot) getXRange() (float64, float64) {
	if len(p.XData) == 0 {
		maxX := 1.0
		for _, yData := range p.YData {
			maxX = max(maxX, float64(len(yData)))
		}
		return 0.0, maxX
	}

	minX := math.Inf(1)
	maxX := math.Inf(-1)
	for _, v := range p.XData {
		minX = min(minX, v)
		maxX = max(maxX, v)
	}
	return minX, maxX

}

func (p *Plot) getYRange() (float64, float64) {
	if p.MinVal != 0.0 || p.MaxVal != 0.0 {
		return p.MinVal, p.MaxVal
	}
	if len(p.YData) == 0 || len(p.YData[0]) == 0 {
		return 0.0, 1.0
	}
	minY := math.Inf(1)
	maxY := math.Inf(-1)
	for _, d := range p.YData {
		for _, v := range d {
			minY = min(minY, v)
			maxY = max(maxY, v)
		}
	}
	return minY, maxY
}

func (p *Plot) updateAxisLimits() {
	p.axisLimits.Min.X, p.axisLimits.Max.X = p.getXRange()
	p.axisLimits.Min.Y, p.axisLimits.Max.Y = p.getYRange()
}

func (self *Plot) Draw(buf *Buffer) {
	self.Block.Draw(buf)

	self.updateAxisLimits()

	drawArea := self.Inner
	if self.ShowAxes {
		drawArea = image.Rect(
			self.Inner.Min.X+yAxisLabelsWidth+1, self.Inner.Min.Y,
			self.Inner.Max.X, self.Inner.Max.Y-xAxisLabelsHeight-1,
		)
		self.plotAxes(buf)
	}

	self.renderBraille(buf, drawArea)
}

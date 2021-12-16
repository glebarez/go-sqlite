// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// helpers for plotting benchmark results
package benchmark

import (
	"fmt"
	"os"
	"path"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	yAxisCeilStep = 200000
)

var transparentColor = drawing.ColorWhite.WithAlpha(0)

type GraphCompareOfNRows struct {
	// this fields should be set externally
	rowCountsE []int
	title      string
	palette    chart.ColorPalette

	// this fields are for private use
	seriesNameS   []string
	seriesValuesS [][]float64
}

func (g *GraphCompareOfNRows) AddSeries(name string, values []float64) {
	g.seriesNameS = append(g.seriesNameS, name)
	g.seriesValuesS = append(g.seriesValuesS, values)
}

func (g *GraphCompareOfNRows) Render(filename string) error {
	// new chart object
	graph := g.newGraph()

	// generate series objects for graph
	for i, seriesName := range g.seriesNameS {
		// get corresponding series values
		seriesValues := g.seriesValuesS[i]

		// create series object
		graph.Series = append(graph.Series, g.createSeries(seriesName, seriesValues))

		// adjust max for Y axis
		yMax := (int(max(seriesValues...)/yAxisCeilStep) + 1) * yAxisCeilStep // a special case of ceil()
		if graph.YAxis.Range.GetMax() < float64(yMax) {
			graph.YAxis.Range = &chart.ContinuousRange{
				Min: 0,
				Max: float64(yMax),
			}
		}

		// skip annotations for first series
		if i == 0 {
			continue
		}

		// for every series except first, we create a ratio annotation s[X]/s[0]
		annotations := &chart.AnnotationSeries{}
		for i, v := range seriesValues {
			annotations.Annotations = append(annotations.Annotations, g.newRatioAnnotation(
				float64(g.rowCountsE[i]),
				v,
				v/g.seriesValuesS[0][i],
			))

		}

		// append annotations to graph
		graph.Series = append(graph.Series, annotations)
	}

	// add legend
	graph.Elements = []chart.Renderable{
		chart.Legend(graph, chart.Style{
			FontSize:    12,
			StrokeColor: transparentColor,
			FillColor:   g.palette.CanvasColor(),
			FontColor:   g.palette.TextColor().WithAlpha(192),
		}),
	}

	// write into file
	if err := os.MkdirAll(path.Dir(filename), 0775); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := graph.Render(chart.PNG, f); err != nil {
		return err
	}
	return nil
}

func (g *GraphCompareOfNRows) createSeries(name string, values []float64) chart.Series {
	// convert E values of rowCount onto float64
	var xValues []float64
	for _, e := range g.rowCountsE {
		xValues = append(xValues, float64(e))
	}

	// create series
	series := &chart.ContinuousSeries{
		Name: name,
		Style: chart.Style{
			DotWidth:    2.5,
			Show:        true,
			StrokeWidth: 1.5,
		},
		XValues: xValues,
		YValues: values,
	}

	// save in series slice
	return series
}

func (g *GraphCompareOfNRows) newGraph() *chart.Chart {
	return &chart.Chart{
		ColorPalette: g.palette,
		Title:        g.title,
		TitleStyle: chart.Style{
			Show: true,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 20,
			},
		},
		Canvas: chart.Style{},
		XAxis: chart.XAxis{
			Style:     chart.StyleShow(),
			NameStyle: chart.StyleShow(),
			Name:      "rows",
			Ticks:     g.genXticks(),
		},
		YAxis: chart.YAxis{
			Range:          &chart.ContinuousRange{},
			Style:          chart.StyleShow(),
			Name:           "rows/sec",
			NameStyle:      chart.StyleShow(),
			ValueFormatter: func(v interface{}) string { return fmt.Sprintf("%.0f", v) },
		},
	}
}

func (g *GraphCompareOfNRows) newRatioAnnotation(x, y, ratio float64) chart.Value2 {
	return chart.Value2{
		XValue: x,
		YValue: y,
		Label:  fmt.Sprintf("%.2fx", ratio),
		Style: chart.Style{
			FontSize:            8,
			TextHorizontalAlign: chart.TextHorizontalAlignLeft,
			FillColor:           transparentColor,                     // full tranparency
			StrokeColor:         transparentColor,                     // full transparency
			FontColor:           g.palette.TextColor().WithAlpha(255), // no transaprency
			TextRotationDegrees: 45,
		},
	}
}

func (g *GraphCompareOfNRows) genXticks() []chart.Tick {
	var ticks []chart.Tick
	for i, e := range g.rowCountsE {
		ticks = append(ticks, chart.Tick{
			Value: float64(e),
			Label: fmt.Sprintf("1e%d", i+1),
		})
	}
	return ticks
}

func max(f ...float64) float64 {
	if len(f) == 0 {
		return 0
	}
	m := f[0]
	for i := 1; i < len(f); i++ {
		if m < f[i] {
			m = f[i]
		}
	}
	return m
}

type palette struct {
	bgColor           drawing.Color
	bgStrokeColor     drawing.Color
	canvasColor       drawing.Color
	canvasStrokeColor drawing.Color
	axisStrokeColor   drawing.Color
	textColor         drawing.Color
	seriesColor       []drawing.Color
}

func (p *palette) BackgroundColor() drawing.Color       { return p.bgColor }
func (p *palette) BackgroundStrokeColor() drawing.Color { return p.bgStrokeColor }
func (p *palette) CanvasColor() drawing.Color           { return p.canvasColor }
func (p *palette) CanvasStrokeColor() drawing.Color     { return p.canvasStrokeColor }
func (p *palette) AxisStrokeColor() drawing.Color       { return p.axisStrokeColor }
func (p *palette) TextColor() drawing.Color             { return p.textColor }
func (p *palette) GetSeriesColor(i int) drawing.Color   { return p.seriesColor[i%len(p.seriesColor)] }

var DarkPalette = &palette{
	bgColor:         drawing.ColorFromHex("252526"),
	canvasColor:     drawing.ColorFromHex("1e1e1e1"),
	textColor:       drawing.ColorFromHex("d4d4d4").WithAlpha(128),
	axisStrokeColor: drawing.ColorFromHex("d4d4d4").WithAlpha(128),
	seriesColor: []drawing.Color{
		drawing.ColorFromHex("d5d5a5"),
		drawing.ColorFromHex("569cd5"),
	},
}

var LightPalette = &palette{
	canvasColor:     drawing.ColorFromHex("f2f2f2"),
	bgColor:         drawing.ColorFromHex("f5f5f5"),
	textColor:       drawing.ColorFromHex("393939").WithAlpha(128),
	axisStrokeColor: drawing.ColorFromHex("393939").WithAlpha(128),
	seriesColor: []drawing.Color{
		drawing.ColorFromHex("aa3731"),
		drawing.ColorFromHex("5a77c7"),
	},
}

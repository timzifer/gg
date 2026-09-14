// Command repro-alpha-violin renders figure's violin example on the GPU and
// CPU. Run against each gg revision to compare the translucent interiors.
package main

import (
	"flag"
	"fmt"
	"math"

	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu"
	_ "github.com/gogpu/wgpu/hal/allbackends"
	"github.com/timzifer/figure"
	ggbackend "github.com/timzifer/figure/backend/gg"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/palette"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/theme"
)

func main() {
	out := flag.String("out", "violin", "output filename prefix")
	flag.Parse()
	if gg.Accelerator() == nil {
		panic("no GPU accelerator registered")
	}
	render(*out + "-gpu.png")
	gg.CloseAccelerator()
	gg.RegisterCoverageFiller(nil)
	render(*out + "-cpu.png")
	fmt.Println("Rendered", *out)
}

func render(out string) {
	p := figure.New(figure.Size(1000, 660), figure.Title("Latency by service"), figure.YTitle("milliseconds"), figure.Legend(true), figure.Theme(theme.Light))
	p.X(scale.Ordinal())
	p.Y(scale.Linear(scale.Nice(), scale.Zero()))
	p.Add(geom.Violin(serviceLatencies(), geom.X("service"), geom.Y("ms"), geom.GroupBy("region"), geom.Bandwidth(3), geom.ColorBy("region", scale.Qualitative(palette.OkabeIto))))
	if err := p.Render(ggbackend.PNG(out)); err != nil {
		panic(err)
	}
}

// Same deterministic data as figure's gallery: checkout has a slow path.
func serviceLatencies() figure.Source {
	names := []string{"auth", "search", "checkout"}
	regions := []string{"eu", "us"}
	var vals []float64
	var svc, region []string
	for i := range 900 {
		s, r := i%3, (i/3)%2
		v := math.Exp(3.2 + 0.35*float64(s) + 0.2*float64(r) + 0.4*noise(i))
		if s == 2 && i%7 == 0 {
			v *= 3
		}
		vals = append(vals, v)
		svc = append(svc, names[s])
		region = append(region, regions[r])
	}
	return figure.NewTable().Float64("ms", vals).String("service", svc).String("region", region)
}
func noise(i int) float64 {
	v := uint64(i)*2862933555777941757 + 3037000493
	sum := 0.0
	for range 12 {
		v = v*6364136223846793005 + 1442695040888963407
		sum += float64(v>>11) / float64(uint64(1)<<53)
	}
	return sum - 6
}

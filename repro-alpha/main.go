// Command repro-alpha compares opaque and translucent fills and strokes on
// the GPU and CPU. Run once against the original gg and once against the fix.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu"
	_ "github.com/gogpu/wgpu/hal/allbackends"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var alphas = []float64{1, 0.75, 0.5, 89.0 / 255, 0.125}
var rows = []string{"Rectangle (SDF)", "Convex fill", "Concave fill", "Two subpaths"}

func main() {
	out := flag.String("out", "alpha", "output prefix (without .png)")
	flag.Parse()
	if gg.Accelerator() == nil {
		panic("no GPU accelerator registered")
	}
	gpu := render()
	gg.CloseAccelerator()
	gg.RegisterCoverageFiller(nil)
	cpu := render()
	for r, name := range rows {
		x, y := 165+3*150+50, 70+r*110+40
		if r == 3 {
			x -= 25
		}
		fmt.Printf("%s at alpha=89/255: GPU %v; CPU %v\n", name, gpu.RGBAAt(x, y), cpu.RGBAAt(x, y))
	}
	save(*out+"-gpu.png", gpu)
	save(*out+"-cpu.png", cpu)
}

func render() *image.RGBA {
	c := gg.NewContext(940, 530)
	defer c.Close()
	c.ClearWithColor(gg.White)
	for row := range rows {
		for col, alpha := range alphas {
			x, y := float64(165+col*150), float64(70+row*110)
			paint := gg.RGBA{G: 114.0 / 255, B: 178.0 / 255, A: alpha}
			if row%2 == 1 {
				paint = gg.RGBA{R: 213.0 / 255, G: 94.0 / 255, A: alpha}
			}
			c.SetColor(paint)
			p := gg.NewPath()
			switch row {
			case 0:
				p.Rectangle(x+10, y+5, 90, 75)
			case 1:
				p.MoveTo(x+10, y+5)
				p.LineTo(x+100, y+15)
				p.LineTo(x+90, y+80)
				p.LineTo(x+20, y+70)
				p.Close()
			case 2:
				p.MoveTo(x+10, y+5)
				p.LineTo(x+100, y+5)
				p.LineTo(x+65, y+40)
				p.LineTo(x+100, y+80)
				p.LineTo(x+10, y+80)
				p.Close()
			case 3:
				p.Rectangle(x+5, y+5, 40, 75)
				p.Rectangle(x+60, y+5, 40, 75)
			}
			if err := c.FillPath(p); err != nil {
				panic(err)
			}
		}
	}
	if err := c.FlushGPU(); err != nil {
		panic(err)
	}
	img := c.Image()
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, image.Point{}, draw.Src)
	label := func(x, y int, s string) {
		d := font.Drawer{Dst: result, Src: image.NewUniform(color.RGBA{30, 30, 30, 255}), Face: basicfont.Face7x13, Dot: fixed.P(x, y)}
		d.DrawString(s)
	}
	label(18, 25, "Same colors and geometry; only alpha and rendering path vary")
	for col, alpha := range alphas {
		label(175+col*150, 50, fmt.Sprintf("alpha %.3f", alpha))
	}
	for row, name := range rows {
		label(15, 112+row*110, name)
	}
	label(18, 515, "Fully covered interior pixels are compared; edge antialiasing may differ.")
	return result
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, img); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
}

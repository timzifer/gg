// Command repro shows gg's GPU tier cancelling a stroke where it overlaps
// itself. It strokes a tight zigzag once with the GPU accelerator registered
// and once on the CPU, and compares the two colour by colour.
//
//	go run .            # against the gg in go.mod
//
// With gogpu/gg v0.52.5 about a fifth of the stroke's pixels come out wrong on
// the GPU: the outline is filled even-odd, so every turn where consecutive
// segments overlap is cancelled into a hole. With the fix, the GPU matches the
// CPU to anti-aliasing.
package main

import (
	"fmt"
	"image"
	"os"

	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu"             // registers the GPU accelerator
	_ "github.com/gogpu/wgpu/hal/allbackends" // wgpu's HAL backends, so an adapter is found
)

const size = 320

// zigzag strokes a line 22 px wide that turns back every 28 px, so each turn
// is covered twice by the expanded outline.
func zigzag(tag string) *image.RGBA {
	c := gg.NewContext(size, size)
	defer func() { _ = c.Close() }()
	c.SetColor(gg.RGBA{R: 1, G: 1, B: 1, A: 1})
	c.DrawRectangle(0, 0, size, size)
	_ = c.Fill()

	c.ClearPath()
	for k := 0; k <= 10; k++ {
		x := 20 + 28*float64(k)
		y := 40.0
		if k%2 == 1 {
			y = 280
		}
		if k == 0 {
			c.MoveTo(x, y)
		} else {
			c.LineTo(x, y)
		}
	}
	c.SetColor(gg.RGBA{R: 0.1, G: 0.25, B: 0.8, A: 1})
	c.SetLineWidth(22)
	c.SetLineJoin(gg.LineJoinMiter)
	c.SetLineCap(gg.LineCapButt)
	_ = c.Stroke()
	_ = c.FlushGPU()
	_ = c.SavePNG("zigzag-" + tag + ".png")

	src := c.Image()
	out := image.NewRGBA(src.Bounds())
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			out.Set(x, y, src.At(x, y))
		}
	}
	return out
}

func main() {
	if gg.Accelerator() == nil {
		fmt.Println("no GPU accelerator registered; nothing to compare")
		os.Exit(1)
	}
	onGPU := zigzag("gpu")
	gg.CloseAccelerator()
	onCPU := zigzag("cpu")

	ink, wrong := 0, 0
	for i := 0; i+3 < len(onCPU.Pix); i += 4 {
		if onCPU.Pix[i+2] > 150 && onCPU.Pix[i] < 120 {
			ink++
		}
		for c := 0; c < 3; c++ {
			d := int(onGPU.Pix[i+c]) - int(onCPU.Pix[i+c])
			if d > 48 || d < -48 {
				wrong++
				break
			}
		}
	}
	fmt.Printf("stroke pixels on the CPU: %d; wrong on the GPU: %d (%.1f%%)\n",
		ink, wrong, 100*float64(wrong)/float64(ink))
	fmt.Println("wrote zigzag-gpu.png and zigzag-cpu.png")
}

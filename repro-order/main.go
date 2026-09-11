// Command repro-order shows gg's GPU tier reordering draws under one clip.
//
// It paints three shapes back to front — a blue square, an orange concave
// arrow over it, a green square over the arrow — once with the GPU
// accelerator registered and once on the CPU, and compares the two.
//
//	go run .            # against the gg in go.mod
//
// The squares are convex and take the convex fast path; the arrow is not and
// takes stencil-then-cover. With gogpu/gg v0.52.5 the GPU draws every convex
// fill of a clip before every stencil fill, so the arrow lands on top of the
// green square that was painted after it. With the fix the GPU matches the CPU.
package main

import (
	"fmt"
	"image"
	"os"

	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu"             // registers the GPU accelerator
	_ "github.com/gogpu/wgpu/hal/allbackends" // wgpu's HAL backends, so an adapter is found
)

const size = 240

func paint(tag string) *image.RGBA {
	c := gg.NewContext(size, size)
	defer func() { _ = c.Close() }()
	c.SetColor(gg.RGBA{R: 1, G: 1, B: 1, A: 1})
	c.DrawRectangle(0, 0, size, size)
	_ = c.Fill()

	square := func(x, y float64, col gg.RGBA) {
		c.ClearPath()
		c.MoveTo(x, y)
		c.LineTo(x+100, y)
		c.LineTo(x+100, y+100)
		c.LineTo(x, y+100)
		c.ClosePath()
		c.SetColor(col)
		_ = c.Fill()
	}

	// Far: a blue square.
	square(20, 20, gg.RGBA{R: 0.15, G: 0.35, B: 0.85, A: 1})

	// Middle: an orange concave arrow across the whole picture.
	c.ClearPath()
	c.MoveTo(30, 60)
	c.LineTo(220, 120)
	c.LineTo(30, 180)
	c.LineTo(90, 120)
	c.ClosePath()
	c.SetColor(gg.RGBA{R: 0.95, G: 0.55, B: 0.1, A: 1})
	_ = c.Fill()

	// Near: a green square, which must cover the arrow's tip.
	square(120, 90, gg.RGBA{R: 0.1, G: 0.65, B: 0.3, A: 1})

	_ = c.FlushGPU()
	_ = c.SavePNG("order-" + tag + ".png")

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
	onGPU := paint("gpu")
	gg.CloseAccelerator()
	onCPU := paint("cpu")

	wrong := 0
	for i := 0; i+3 < len(onCPU.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			d := int(onGPU.Pix[i+c]) - int(onCPU.Pix[i+c])
			if d > 48 || d < -48 {
				wrong++
				break
			}
		}
	}
	fmt.Printf("pixels that differ between GPU and CPU: %d of %d (%.1f%%)\n",
		wrong, size*size, 100*float64(wrong)/float64(size*size))
	fmt.Println("wrote order-gpu.png and order-cpu.png")
}

package raster

import (
	"bytes"
	"image"
	"math"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/recording"
	"github.com/gogpu/gg/text"
	"golang.org/x/image/font/gofont/goregular"
)

func TestBackendRegistration(t *testing.T) {
	// Verify the backend is registered
	if !recording.IsRegistered("raster") {
		t.Fatal("raster backend not registered")
	}

	// Verify we can create a backend via registry
	backend, err := recording.NewBackend("raster")
	if err != nil {
		t.Fatalf("failed to create raster backend: %v", err)
	}
	if backend == nil {
		t.Fatal("backend is nil")
	}

	// Verify it's the correct type
	_, ok := backend.(*Backend)
	if !ok {
		t.Fatal("backend is not *raster.Backend")
	}
}

func TestBackendLifecycle(t *testing.T) {
	backend := NewBackend()

	// Begin
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Verify dimensions
	if backend.Width() != 100 {
		t.Errorf("Width = %d, want 100", backend.Width())
	}
	if backend.Height() != 100 {
		t.Errorf("Height = %d, want 100", backend.Height())
	}

	// End
	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Image should be available
	img := backend.Image()
	if img == nil {
		t.Fatal("Image() returned nil")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("Image bounds = %v, want 100x100", bounds)
	}
}

func TestBackendFillRect(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Fill a red rectangle
	rect := recording.NewRect(10, 10, 50, 50)
	brush := recording.NewSolidBrush(gg.Red)
	backend.FillRect(rect, brush)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Verify a pixel inside the rectangle is red
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Check center of rectangle (35, 35)
	pixel := rgba.RGBAAt(35, 35)
	// Red in 8-bit is R=255, G=0, B=0, A=255
	if pixel.R < 200 || pixel.G > 50 || pixel.B > 50 {
		t.Errorf("pixel at (35,35) = %v, expected red", pixel)
	}
}

func TestBackendFillPath(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a triangle path
	path := gg.NewPath()
	path.MoveTo(50, 10)
	path.LineTo(90, 90)
	path.LineTo(10, 90)
	path.Close()

	brush := recording.NewSolidBrush(gg.Blue)
	backend.FillPath(path, brush, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Verify a pixel inside the triangle is blue
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Check center of triangle (50, 60)
	pixel := rgba.RGBAAt(50, 60)
	if pixel.B < 200 || pixel.R > 50 || pixel.G > 50 {
		t.Errorf("pixel at (50,60) = %v, expected blue", pixel)
	}
}

func TestBackendStrokePath(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a line path
	path := gg.NewPath()
	path.MoveTo(10, 50)
	path.LineTo(90, 50)

	brush := recording.NewSolidBrush(gg.Green)
	stroke := recording.Stroke{
		Width:      5.0,
		Cap:        recording.LineCapRound,
		Join:       recording.LineJoinRound,
		MiterLimit: 4.0,
	}
	backend.StrokePath(path, brush, stroke)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Verify a pixel on the line is green
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Check center of line (50, 50)
	pixel := rgba.RGBAAt(50, 50)
	if pixel.G < 200 || pixel.R > 50 || pixel.B > 50 {
		t.Errorf("pixel at (50,50) = %v, expected green", pixel)
	}
}

func TestBackendSaveRestore(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Save/Restore should not panic
	backend.Save()
	backend.Restore()

	// Multiple saves and restores
	backend.Save()
	backend.Save()
	backend.Restore()
	backend.Restore()

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}
}

func TestBackendSetTransform(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Set a transform (translate by 10,10)
	m := recording.Translate(10, 10)
	backend.SetTransform(m)

	// This should not panic
	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}
}

func TestBackendWriteTo(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(50, 50)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Draw something
	rect := recording.NewRect(0, 0, 50, 50)
	brush := recording.NewSolidBrush(gg.Red)
	backend.FillRect(rect, brush)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Write to buffer
	var buf bytes.Buffer
	n, err := backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	if n == 0 {
		t.Error("WriteTo wrote 0 bytes")
	}

	// Verify PNG signature
	data := buf.Bytes()
	if len(data) < 8 {
		t.Fatal("PNG data too short")
	}
	pngSig := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i := 0; i < 8; i++ {
		if data[i] != pngSig[i] {
			t.Fatal("invalid PNG signature")
		}
	}
}

func TestBackendPixmap(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(50, 50)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Get pixmap
	pixmap := backend.Pixmap()
	if pixmap == nil {
		t.Fatal("Pixmap() returned nil")
	}
	if pixmap.Width() != 50 || pixmap.Height() != 50 {
		t.Errorf("Pixmap dimensions = %dx%d, want 50x50", pixmap.Width(), pixmap.Height())
	}
}

func TestBackendInterfaceCompliance(t *testing.T) {
	// Verify Backend implements all interfaces
	var _ recording.Backend = (*Backend)(nil)
	var _ recording.WriterBackend = (*Backend)(nil)
	var _ recording.FileBackend = (*Backend)(nil)
	var _ recording.PixmapBackend = (*Backend)(nil)
}

func TestBackendLinearGradient(t *testing.T) {
	// NOTE: Linear gradient rendering is limited by gg.SoftwareRenderer
	// which currently only supports solid colors in fillSupersampled.
	// The gradient brush is correctly created and set, but the renderer
	// falls back to black. This test verifies the backend doesn't crash.
	// Full gradient support requires gg library enhancement.
	t.Skip("gradient rendering not yet supported by gg.SoftwareRenderer")

	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a linear gradient brush
	grad := recording.NewLinearGradientBrush(0, 0, 100, 0).
		AddColorStop(0, gg.Red).
		AddColorStop(1, gg.Blue)

	rect := recording.NewRect(0, 0, 100, 100)
	backend.FillRect(rect, grad)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Verify gradient colors
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Left side should be red-ish
	leftPixel := rgba.RGBAAt(5, 50)
	if leftPixel.R < 150 {
		t.Errorf("left pixel = %v, expected more red", leftPixel)
	}

	// Right side should be blue-ish
	rightPixel := rgba.RGBAAt(95, 50)
	if rightPixel.B < 150 {
		t.Errorf("right pixel = %v, expected more blue", rightPixel)
	}
}

func TestBackendRadialGradient(t *testing.T) {
	// NOTE: Radial gradient rendering is limited by gg.SoftwareRenderer
	// which currently only supports solid colors in fillSupersampled.
	// The gradient brush is correctly created and set, but the renderer
	// falls back to black. This test verifies the backend doesn't crash.
	// Full gradient support requires gg library enhancement.
	t.Skip("gradient rendering not yet supported by gg.SoftwareRenderer")

	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a radial gradient brush
	grad := recording.NewRadialGradientBrush(50, 50, 0, 50).
		AddColorStop(0, gg.White).
		AddColorStop(1, gg.Black)

	rect := recording.NewRect(0, 0, 100, 100)
	backend.FillRect(rect, grad)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Verify gradient colors
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Center should be white
	centerPixel := rgba.RGBAAt(50, 50)
	if centerPixel.R < 200 || centerPixel.G < 200 || centerPixel.B < 200 {
		t.Errorf("center pixel = %v, expected white", centerPixel)
	}

	// Edge should be darker
	edgePixel := rgba.RGBAAt(5, 50)
	if edgePixel.R > 100 || edgePixel.G > 100 || edgePixel.B > 100 {
		t.Errorf("edge pixel = %v, expected dark", edgePixel)
	}
}

func TestRecordingPlayback(t *testing.T) {
	// Create a recording
	rec := recording.NewRecorder(100, 100)
	rec.SetRGB(1, 0, 0) // Red
	rec.DrawCircle(50, 50, 30)
	rec.Fill()
	r := rec.FinishRecording()

	// Playback to raster backend
	backend := NewBackend()
	err := r.Playback(backend)
	if err != nil {
		t.Fatalf("Playback failed: %v", err)
	}

	// Verify the circle was drawn
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Center of circle should be red
	centerPixel := rgba.RGBAAt(50, 50)
	if centerPixel.R < 200 || centerPixel.G > 50 || centerPixel.B > 50 {
		t.Errorf("center pixel = %v, expected red", centerPixel)
	}

	// Outside circle should be transparent/black
	outsidePixel := rgba.RGBAAt(5, 5)
	if outsidePixel.R > 50 || outsidePixel.G > 50 || outsidePixel.B > 50 {
		t.Errorf("outside pixel = %v, expected transparent/black", outsidePixel)
	}
}

func TestRecordingPlaybackViaRegistry(t *testing.T) {
	// Create a recording
	rec := recording.NewRecorder(100, 100)
	rec.SetRGB(0, 1, 0) // Green
	rec.FillRectangle(20, 20, 60, 60)
	r := rec.FinishRecording()

	// Get backend from registry
	backend, err := recording.NewBackend("raster")
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}

	// Playback
	err = r.Playback(backend)
	if err != nil {
		t.Fatalf("Playback failed: %v", err)
	}

	// Cast to raster backend to get image
	rb, ok := backend.(*Backend)
	if !ok {
		t.Fatal("backend is not *raster.Backend")
	}

	img := rb.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Center of rectangle should be green
	pixel := rgba.RGBAAt(50, 50)
	if pixel.G < 200 || pixel.R > 50 || pixel.B > 50 {
		t.Errorf("pixel = %v, expected green", pixel)
	}
}

func TestRecordingPlaybackStrokeRectangle(t *testing.T) {
	rec := recording.NewRecorder(100, 100)
	rec.SetStrokeRGB(1, 0, 0)
	rec.SetLineWidth(4)
	rec.StrokeRectangle(20, 20, 60, 60)

	backend := NewBackend()
	if err := rec.FinishRecording().Playback(backend); err != nil {
		t.Fatalf("Playback failed: %v", err)
	}

	rgba, ok := backend.Image().(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}
	if pixel := rgba.RGBAAt(20, 50); pixel.R < 200 || pixel.G > 50 || pixel.B > 50 || pixel.A == 0 {
		t.Fatalf("left rectangle edge pixel = %v, want opaque red", pixel)
	}
	if pixel := rgba.RGBAAt(50, 50); pixel.A != 0 {
		t.Fatalf("rectangle interior pixel = %v, want transparent", pixel)
	}
}

func TestRecordingPlaybackStrokeRectangleMatchesDirect(t *testing.T) {
	for _, tc := range []struct {
		name                string
		x, y, width, height float64
		transform           recording.Matrix
	}{
		{name: "translation", x: 20, y: 20, width: 60, height: 50, transform: recording.Translate(10, 15)},
		{name: "positive nonuniform scale", x: 20, y: 20, width: 50, height: 60, transform: recording.Matrix{A: 1.5, C: 5, E: 0.75, F: 10}},
		{name: "scale down", x: 20, y: 20, width: 80, height: 80, transform: recording.Matrix{A: 0.5, C: 10, E: 0.75, F: 10}},
		{name: "zero scale", x: 20, y: 20, width: 80, height: 80, transform: recording.Matrix{C: 90, F: 90}},
		{name: "rotation", x: -30, y: -20, width: 60, height: 40, transform: recording.Translate(90, 90).Multiply(recording.Rotate(math.Pi / 4))},
		{name: "shear", x: 10, y: 10, width: 60, height: 50, transform: recording.Translate(20, 20).Multiply(recording.Shear(0.4, 0.2))},
		{name: "reflection", x: 20, y: 20, width: 60, height: 50, transform: recording.Translate(130, 0).Multiply(recording.Scale(-1, 1))},
		{name: "negative width", x: 80, y: 20, width: -60, height: 60, transform: recording.Identity()},
		{name: "negative height", x: 20, y: 80, width: 60, height: -60, transform: recording.Identity()},
		{name: "negative width and height", x: 80, y: 80, width: -60, height: -60, transform: recording.Identity()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := gg.NewContext(180, 180)
			direct.SetRGB(0, 0, 0)
			direct.SetLineWidth(2)
			direct.SetDash(11, 7)
			direct.SetDashOffset(3)
			direct.SetTransform(gg.Matrix{
				A: tc.transform.A, B: tc.transform.B, C: tc.transform.C,
				D: tc.transform.D, E: tc.transform.E, F: tc.transform.F,
			})
			direct.DrawRectangle(tc.x, tc.y, tc.width, tc.height)
			if err := direct.Stroke(); err != nil {
				t.Fatalf("direct Stroke failed: %v", err)
			}

			rec := recording.NewRecorder(180, 180)
			rec.SetStrokeRGB(0, 0, 0)
			rec.SetLineWidth(2)
			rec.SetDash(11, 7)
			rec.SetDashOffset(3)
			rec.SetTransform(tc.transform)
			rec.StrokeRectangle(tc.x, tc.y, tc.width, tc.height)
			backend := NewBackend()
			if err := rec.FinishRecording().Playback(backend); err != nil {
				t.Fatalf("Playback failed: %v", err)
			}

			got, ok := backend.Image().(*image.RGBA)
			if !ok {
				t.Fatal("expected raster backend to return *image.RGBA")
			}
			want := direct.Image().(*image.RGBA)
			if !bytes.Equal(got.Pix, want.Pix) {
				t.Fatalf("recorded stroke differs from direct stroke for transform %#v and dimensions (%v,%v)", tc.transform, tc.width, tc.height)
			}
		})
	}
}

func TestBackendDashedStroke(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(100, 100)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a line path
	path := gg.NewPath()
	path.MoveTo(10, 50)
	path.LineTo(90, 50)

	brush := recording.NewSolidBrush(gg.Black)
	stroke := recording.Stroke{
		Width:       3.0,
		Cap:         recording.LineCapButt,
		Join:        recording.LineJoinMiter,
		MiterLimit:  4.0,
		DashPattern: []float64{10, 5}, // 10px dash, 5px gap
		DashOffset:  0,
	}
	backend.StrokePath(path, brush, stroke)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Verify the stroke was drawn (we can't easily verify dashing, just that something was drawn)
	img := backend.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA")
	}

	// Check somewhere on the line
	pixel := rgba.RGBAAt(15, 50)
	// Should have some alpha (not fully transparent)
	if pixel.A == 0 {
		t.Error("expected non-transparent pixel on dashed line")
	}
}

func loadGoRegular(t *testing.T) text.Face {
	t.Helper()
	src, err := text.NewFontSource(goregular.TTF)
	if err != nil {
		t.Fatalf("NewFontSource: %v", err)
	}
	t.Cleanup(func() { _ = src.Close() })
	return src.Face(24)
}

func TestDrawTextRendersInkPixels(t *testing.T) {
	face := loadGoRegular(t)

	rec := recording.NewRecorder(200, 60)
	rec.SetFont(face)
	rec.SetFontSize(24)
	rec.SetFillRGBA(0, 0, 0, 1)
	rec.DrawString("Hello", 10, 40)
	r := rec.FinishRecording()

	backend := NewBackend()
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	img := backend.Image()
	rgba := toRGBA(img)
	ink := countNonWhitePixels(rgba)
	if ink == 0 {
		t.Fatal("DrawText produced no ink pixels — text not rendered")
	}
	t.Logf("ink pixels: %d", ink)
}

func TestDrawTextWithNilFaceIsNoOp(t *testing.T) {
	rec := recording.NewRecorder(100, 40)
	rec.SetFillRGBA(0, 0, 0, 1)
	rec.DrawString("Ghost", 10, 30)
	r := rec.FinishRecording()

	backend := NewBackend()
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	img := backend.Image()
	rgba := toRGBA(img)
	ink := countNonWhitePixels(rgba)
	if ink != 0 {
		t.Errorf("nil face should produce no ink, got %d pixels", ink)
	}
}

func TestDrawTextScaledFontSize(t *testing.T) {
	face := loadGoRegular(t)

	render := func(scale float64) int {
		rec := recording.NewRecorder(200, 100)
		rec.SetFont(face)
		rec.SetFontSize(16)
		if scale != 1 {
			rec.Scale(scale, scale)
		}
		rec.SetFillRGBA(0, 0, 0, 1)
		rec.DrawString("Ag", 5, 20)
		r := rec.FinishRecording()

		backend := NewBackend()
		if err := r.Playback(backend); err != nil {
			t.Fatalf("Playback(scale=%.1f): %v", scale, err)
		}
		return countNonWhitePixels(toRGBA(backend.Image()))
	}

	ink1x := render(1)
	ink2x := render(2)

	if ink1x == 0 {
		t.Fatal("1x produced no ink")
	}
	if ink2x <= ink1x {
		t.Errorf("2x scale (%d ink) should produce more ink than 1x (%d)", ink2x, ink1x)
	}
	t.Logf("1x=%d 2x=%d ratio=%.1f", ink1x, ink2x, float64(ink2x)/float64(ink1x))
}

func TestNewBackendWithScale(t *testing.T) {
	rec := recording.NewRecorder(100, 50)
	rec.SetFillRGBA(1, 0, 0, 1)
	rec.DrawRectangle(10, 10, 80, 30)
	rec.Fill()
	r := rec.FinishRecording()

	backend := NewBackendWithScale(2)
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	img := backend.Image()
	bounds := img.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 100 {
		t.Errorf("2x backend image = %dx%d, want 200x100", bounds.Dx(), bounds.Dy())
	}

	rgba := toRGBA(img)
	red := countRedPixels(rgba)
	if red == 0 {
		t.Fatal("no red pixels — rectangle not rendered at 2x scale")
	}
	t.Logf("2x image: %dx%d, red pixels: %d", bounds.Dx(), bounds.Dy(), red)
}

func TestNewBackendWithScaleText(t *testing.T) {
	face := loadGoRegular(t)

	render := func(scale float64) int {
		rec := recording.NewRecorder(200, 60)
		rec.SetFont(face)
		rec.SetFontSize(20)
		rec.SetFillRGBA(0, 0, 0, 1)
		rec.DrawString("Test", 10, 40)
		r := rec.FinishRecording()

		backend := NewBackendWithScale(scale)
		if err := r.Playback(backend); err != nil {
			t.Fatalf("Playback(scale=%.1f): %v", scale, err)
		}
		return countNonWhitePixels(toRGBA(backend.Image()))
	}

	ink1x := render(1)
	ink2x := render(2)
	if ink1x == 0 {
		t.Fatal("1x text produced no ink")
	}
	if ink2x < ink1x*2 {
		t.Errorf("2x backend text (%d ink) should have roughly 4x more pixels than 1x (%d)", ink2x, ink1x)
	}
}

// TestDrawStringAnchoredMatchesContext verifies that Recorder.DrawStringAnchored
// computes the same anchor offset as Context.DrawStringAnchored. Pre-fix:
// Recorder ignored ax/ay entirely, text drawn at raw (x,y) position.
func TestDrawStringAnchoredMatchesContext(t *testing.T) {
	face := loadGoRegular(t)
	const (
		w, h = 400, 100
		s    = "Hello World"
		x    = 350.0
		y    = 50.0
		ax   = 1.0 // right-aligned
		ay   = 0.5 // vertically centered
	)

	// Direct Context rendering (reference)
	dc := gg.NewContext(w, h)
	dc.SetFont(face)
	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored(s, x, y, ax, ay)
	directImg := toRGBA(dc.Image())
	directInk := countNonWhitePixels(directImg)

	// Recording → raster backend rendering
	rec := recording.NewRecorder(w, h)
	rec.SetFont(face)
	rec.SetFontSize(24)
	rec.SetFillRGBA(0, 0, 0, 1)
	rec.DrawStringAnchored(s, x, y, ax, ay)
	r := rec.FinishRecording()

	backend := NewBackend()
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}
	recImg := toRGBA(backend.Image())
	recInk := countNonWhitePixels(recImg)

	if directInk == 0 {
		t.Fatal("direct Context produced no ink")
	}
	if recInk == 0 {
		t.Fatal("recording produced no ink")
	}

	// Find ink bounds for both
	directBounds := inkBounds(directImg)
	recBounds := inkBounds(recImg)

	// Right-aligned text at x=350 should end near x=350, not start there
	if directBounds.maxX > 360 {
		t.Errorf("direct: text extends past x=360 (maxX=%d) — anchor not applied", directBounds.maxX)
	}
	if abs(recBounds.minX-directBounds.minX) > 3 || abs(recBounds.maxX-directBounds.maxX) > 3 {
		t.Errorf("recording anchor mismatch: direct X=[%d,%d], recording X=[%d,%d]",
			directBounds.minX, directBounds.maxX, recBounds.minX, recBounds.maxX)
	}
	t.Logf("direct X=[%d,%d] ink=%d, recording X=[%d,%d] ink=%d",
		directBounds.minX, directBounds.maxX, directInk, recBounds.minX, recBounds.maxX, recInk)
}

// TestDrawStringAnchoredZeroAnchorNoOffset verifies ax=0,ay=0 produces same
// result as DrawString (no offset applied).
func TestDrawStringAnchoredZeroAnchorNoOffset(t *testing.T) {
	face := loadGoRegular(t)

	render := func(useAnchored bool) *image.RGBA {
		rec := recording.NewRecorder(200, 60)
		rec.SetFont(face)
		rec.SetFontSize(24)
		rec.SetFillRGBA(0, 0, 0, 1)
		if useAnchored {
			rec.DrawStringAnchored("Test", 10, 40, 0, 0)
		} else {
			rec.DrawString("Test", 10, 40)
		}
		r := rec.FinishRecording()
		backend := NewBackend()
		_ = r.Playback(backend)
		return toRGBA(backend.Image())
	}

	plain := render(false)
	anchored := render(true)

	plainB := inkBounds(plain)
	anchoredB := inkBounds(anchored)

	if plainB.count == 0 || anchoredB.count == 0 {
		t.Fatalf("no ink: plain=%d anchored=%d", plainB.count, anchoredB.count)
	}
	if abs(plainB.minX-anchoredB.minX) > 1 || abs(plainB.maxX-anchoredB.maxX) > 1 {
		t.Errorf("zero anchor changed position: plain X=[%d,%d], anchored X=[%d,%d]",
			plainB.minX, plainB.maxX, anchoredB.minX, anchoredB.maxX)
	}
}

type inkRect struct {
	minX, minY, maxX, maxY, count int
}

func inkBounds(img *image.RGBA) inkRect {
	bounds := img.Bounds()
	r := inkRect{minX: bounds.Max.X, minY: bounds.Max.Y, maxX: -1, maxY: -1}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A == 0 || (c.R >= 250 && c.G >= 250 && c.B >= 250) {
				continue
			}
			r.count++
			r.minX = min(r.minX, x)
			r.minY = min(r.minY, y)
			r.maxX = max(r.maxX, x)
			r.maxY = max(r.maxY, y)
		}
	}
	return r
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TestNewBackendWithScaleFillRect is the regression test for BUG-RASTER-BACKEND-SCALE-001:
// FillRect called ctx.Identity() which reset Scale(2,2) → rectangle rendered in
// top-left quadrant only (25% coverage instead of 100%).
func TestNewBackendWithScaleFillRect(t *testing.T) {
	rec := recording.NewRecorder(100, 50)
	rec.SetFillRGBA(1, 0, 0, 1)
	rec.FillRectangle(0, 0, 100, 50)
	r := rec.FinishRecording()

	backend := NewBackendWithScale(2)
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	img := backend.Image()
	bounds := img.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 100 {
		t.Fatalf("2x image = %dx%d, want 200x100", bounds.Dx(), bounds.Dy())
	}

	rgba := toRGBA(img)
	total := bounds.Dx() * bounds.Dy()
	red := countRedPixels(rgba)
	coverage := float64(red) / float64(total) * 100
	if coverage < 99 {
		t.Fatalf("FillRect at 2x scale: %.1f%% coverage (want 100%%) — device scale lost", coverage)
	}

	// Verify bottom-right corner is filled (pre-fix: transparent)
	c := rgba.RGBAAt(199, 99)
	if c.A == 0 {
		t.Fatal("bottom-right pixel transparent — device scale not applied to FillRect")
	}
}

func TestNewBackendWithScaleFillPath(t *testing.T) {
	rec := recording.NewRecorder(100, 50)
	rec.SetFillRGBA(0, 0, 1, 1)
	rec.DrawRectangle(0, 0, 100, 50)
	rec.Fill()
	r := rec.FinishRecording()

	backend := NewBackendWithScale(2)
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	rgba := toRGBA(backend.Image())
	bounds := backend.Image().Bounds()
	total := bounds.Dx() * bounds.Dy()
	filled := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rgba.RGBAAt(x, y)
			if c.B > 200 && c.A > 200 {
				filled++
			}
		}
	}
	coverage := float64(filled) / float64(total) * 100
	if coverage < 95 {
		t.Fatalf("FillPath at 2x scale: %.1f%% coverage (want ~100%%)", coverage)
	}
}

func TestNewBackendWithScaleClip(t *testing.T) {
	rec := recording.NewRecorder(100, 100)
	rec.DrawRectangle(10, 10, 80, 80)
	rec.Clip()
	rec.SetFillRGBA(1, 0, 0, 1)
	rec.FillRectangle(0, 0, 100, 100)
	r := rec.FinishRecording()

	backend := NewBackendWithScale(2)
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	rgba := toRGBA(backend.Image())
	// At 2x, clip region (10,10,80,80) → pixels (20,20)-(180,180)
	// Inside clip should be red
	c := rgba.RGBAAt(100, 100)
	if c.R < 200 || c.A < 200 {
		t.Fatalf("center pixel inside clip = R:%d A:%d, want red", c.R, c.A)
	}
	// Outside clip should be transparent
	c = rgba.RGBAAt(5, 5)
	if c.A > 0 {
		t.Fatalf("pixel outside clip has A=%d, want 0", c.A)
	}
}

func TestNewBackendWithScaleSetTransform(t *testing.T) {
	rec := recording.NewRecorder(100, 100)
	rec.Translate(50, 50)
	rec.SetFillRGBA(1, 0, 0, 1)
	rec.DrawRectangle(-10, -10, 20, 20)
	rec.Fill()
	r := rec.FinishRecording()

	backend := NewBackendWithScale(2)
	if err := r.Playback(backend); err != nil {
		t.Fatalf("Playback: %v", err)
	}

	rgba := toRGBA(backend.Image())
	// At 2x, center (50,50) → pixel (100,100), rect 20x20 → 40x40 pixels
	c := rgba.RGBAAt(100, 100)
	if c.R < 200 || c.A < 200 {
		t.Fatalf("center pixel at 2x = R:%d A:%d, want red", c.R, c.A)
	}
}

func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}

func countNonWhitePixels(img *image.RGBA) int {
	count := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A > 0 && (c.R < 250 || c.G < 250 || c.B < 250) {
				count++
			}
		}
	}
	return count
}

func countRedPixels(img *image.RGBA) int {
	count := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R > 200 && c.G < 50 && c.B < 50 && c.A > 200 {
				count++
			}
		}
	}
	return count
}

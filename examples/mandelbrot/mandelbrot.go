// mandelbrot example code adapted from Francesc Campoy's mandelbrot package.
// https://github.com/campoy/mandelbrot
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pkg/profile"
)

func main() {
	var (
		height  = flag.Int("h", 1024, "height of the output image in pixels")
		width   = flag.Int("w", 1024, "width of the output image in pixels")
		mode    = flag.String("mode", "seq", "mode: seq, px, row, workers")
		workers = flag.Int("workers", 1, "number of workers to use")
		trace   = flag.Bool("trace", false, "enable trace")
	)
	flag.Parse()

	if *trace {
		defer profile.Start(profile.TraceProfile).Stop()
	}

	const output = "mandelbrot.png"

	// open a new file
	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}

	// create the image
	c := make([][]color.RGBA, *height)
	for i := range c {
		c[i] = make([]color.RGBA, *width)
	}

	img := &img{
		h: *height,
		w: *width,
		m: c,
	}

	start := time.Now()
	switch *mode {
	case "seq":
		seqFillImg(img)
	case "px":
		oneToOneFillImg(img)
	case "row":
		onePerRowFillImg(img)
	case "workers":
		nWorkersFillImg(img, *workers)
	default:
		panic("unknown mode")
	}
	log.Printf("generation took: %v", time.Since(start))

	// and encoding it
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

type img struct {
	h, w int
	m    [][]color.RGBA
}

func (m *img) At(x, y int) color.Color { return m.m[x][y] }
func (m *img) ColorModel() color.Model { return color.RGBAModel }
func (m *img) Bounds() image.Rectangle { return image.Rect(0, 0, m.h, m.w) }

func seqFillImg(m *img) {
	for i, row := range m.m {
		for j := range row {
			fillPixel(m, i, j)
		}
	}
}

func oneToOneFillImg(m *img) {
	var wg sync.WaitGroup
	wg.Add(m.h * m.w)
	for i, row := range m.m {
		for j := range row {
			go func(i, j int) {
				fillPixel(m, i, j)
				wg.Done()
			}(i, j)
		}
	}
	wg.Wait()
}

func onePerRowFillImg(m *img) {
	var wg sync.WaitGroup
	wg.Add(m.h)
	for i := range m.m {
		go func(i int) {
			for j := range m.m[i] {
				fillPixel(m, i, j)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func nWorkersFillImg(m *img, workers int) {
	c := make(chan struct{ i, j int })
	for i := 0; i < workers; i++ {
		go func() {
			for t := range c {
				fillPixel(m, t.i, t.j)
			}
		}()
	}

	for i, row := range m.m {
		for j := range row {
			c <- struct{ i, j int }{i, j}
		}
	}
	close(c)
}

func fillPixel(m *img, i, j int) {
	// normalized from -2.5 to 1
	xi := 3.5*float64(i)/float64(m.w) - 2.5
	// normalized from -1 to 1
	yi := 2*float64(j)/float64(m.h) - 1

	const maxI = 1000
	var x, y float64
	for i := 0; (x*x+y*y < 4) && i < maxI; i++ {
		x, y = x*x-y*y+xi, 2*x*y+yi
	}

	paint(&m.m[i][j], x, y)
}

func paint(c *color.RGBA, x, y float64) {
	n := byte(x * y)
	c.R, c.G, c.B, c.A = n, n, n, 255
}

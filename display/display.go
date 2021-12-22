package display

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"time"

	rgbmatrix "github.com/aaron2198/go-rpi-rgb-led-matrix"
	"github.com/fogleman/gg"
)

type State int

const (
	Stopped State = iota
	Running
	Killed
)

type WindowInterface interface {
	Render() image.Image
	AddElement(e Element)
	GetElements() []Element
}

type PointAnimation interface {
	Render(*Point)
}

type Element interface {
	Render()
	Draw(*gg.Context)
	Debug(w io.Writer, msg string)
	SetAnimation(PointAnimation)
}

type Window struct {
	ctx      *gg.Context
	Elements []Element
}

type Notification struct {
	Title    string
	Body     string
	Icon     string
	Duration time.Duration
}

type Display struct {
	Canvas      *rgbmatrix.Canvas
	Windows     map[string]WindowInterface
	foreground  string
	framerate   time.Duration
	State       State
	ToState     chan State
	debugWriter io.Writer
	debugger    bool
}

func CreateDisplay(matrix rgbmatrix.Matrix, start WindowInterface, framerate time.Duration) *Display {
	windows := make(map[string]WindowInterface)
	windows["default"] = start
	display := &Display{
		Canvas:     rgbmatrix.NewCanvas(matrix),
		Windows:    windows,
		foreground: "default",
		framerate:  framerate,
		State:      Running,
		ToState:    make(chan State),
	}
	display.run()
	return display
}

func (d *Display) draw() {
	i := d.Windows[d.foreground].Render()
	draw.Draw(d.Canvas, d.Canvas.Bounds(), i, image.Point{}, draw.Over)
	d.Canvas.Render()
}

func (d *Display) run() {
	go func() {
		for {
			if d.debugger {
				d.Debug(d.debugWriter, "Core loop:")
			}
			select {
			case d.State = <-d.ToState:
			default:
				switch d.State {
				case Stopped:
					d.State = <-d.ToState
				case Running:
					d.draw()
					time.Sleep(d.framerate)
				case Killed:
					return
				}
			}
		}
	}()
}

func (d *Display) EnableDebugging(w io.Writer) {
	d.debugWriter = w
	d.debugger = true
}

func (d *Display) GetWindows() []string {
	var windows []string
	for name := range d.Windows {
		windows = append(windows, name)
	}
	return windows
}

func (d *Display) Foreground(windowname string) error {
	// Window must exist
	_, ok := d.Windows[windowname]

	if ok {
		// Present it
		d.foreground = windowname
	} else {
		// Error
		return errors.New("Window not found")
	}
	return nil
}

func (d *Display) Debug(w io.Writer, msg string) {
	fmt.Fprintf(w, "######################- %s -######################\n", msg)
	fmt.Fprintf(w, "Foreground: %s\n", d.foreground)
	fmt.Fprintf(w, "State: %d\n", d.State)
	fmt.Fprintf(w, "Windows:\n")
	for name, win := range d.Windows {
		fmt.Fprintf(w, "###########- %s: %v -###########\n", name, w)
		for _, e := range win.GetElements() {
			e.Debug(w, fmt.Sprintf("%T", e))
		}
	}
	fmt.Fprint(w, "\n")
}

func CreateWindow(width, height int) *Window {
	return &Window{
		ctx: gg.NewContext(width, height),
	}
}

func (d *Display) AddWindow(name string) {
	d.Windows[name] = CreateWindow(d.Canvas.Bounds().Max.X, d.Canvas.Bounds().Max.Y)
}

func (w *Window) AddElement(e Element) {
	w.Elements = append(w.Elements, e)
}

func (w *Window) GetElements() []Element {
	return w.Elements
}

func (w *Window) Render() image.Image {
	w.ctx.SetColor(color.Black)
	w.ctx.Clear()
	for _, e := range w.GetElements() {
		// call render to perform positional calculations
		e.Render()
		// call draw to modify pixels on canvas
		e.Draw(w.ctx)
	}

	return w.ctx.Image()
}

type Circle struct {
	point          Point
	pointanimation PointAnimation
	dirx           int
	diry           int
	s              int
	c              *color.RGBA
}

func CreateCircle(x, y, s int, c *color.RGBA) *Circle {
	return &Circle{
		point:          CreatePoint(x, y),
		pointanimation: nil,
		s:              s,
		c:              c,
		dirx:           1,
		diry:           1,
	}
}

func (c *Circle) Debug(w io.Writer, msg string) {
	fmt.Fprintf(w, "###- %s -###\n", msg)
	fmt.Fprintf(w, "x: %d\n", c.point.X)
	fmt.Fprintf(w, "y: %d\n", c.point.Y)
	fmt.Fprintf(w, "s: %d\n", c.s)
	fmt.Fprintf(w, "c: %v\n", c.c)
}

func (c *Circle) Render() {
	if c.pointanimation != nil {
		c.pointanimation.Render(&c.point)
	}
}

func (c *Circle) Draw(ctx *gg.Context) {

	ctx.DrawCircle(float64(c.point.X), float64(c.point.Y), float64(c.s))
	ctx.SetColor(c.c)
	ctx.Fill()
}

func (c *Circle) SetAnimation(a PointAnimation) {
	c.pointanimation = a
}

type BouncePoint struct {
	dirx    int
	diry    int
	boundx  int
	boundy  int
	padding int
}

func (b *BouncePoint) Render(p *Point) {
	p.X += 1 * b.dirx
	p.Y += 1 * b.diry
	if p.Y+b.padding > b.boundy {
		b.diry = -1
	} else if p.Y-b.padding < 0 {
		b.diry = 1
	}
	if p.X+b.padding > b.boundx {
		b.dirx = -1
	} else if p.X-b.padding < 0 {
		b.dirx = 1
	}
}

func CreateBouncePoint(boundx, boundy, padding int) *BouncePoint {
	return &BouncePoint{
		dirx:    1,
		diry:    1,
		boundx:  boundx,
		boundy:  boundy,
		padding: padding,
	}
}

type Point struct {
	X, Y int
}

func (point *Point) Coord() (int, int) {
	return point.X, point.Y
}

func CreatePoint(x, y int) Point {
	return Point{
		X: x,
		Y: y,
	}
}

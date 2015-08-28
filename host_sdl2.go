// Plaform-dependent routines

package nimble

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"os"
	"reflect"
	"runtime"
	"unsafe"
)

type renderClient interface {
	Init(width, height int32) // Inform client of window size
	Render(pm PixMap)
}

var renderClientList []renderClient

func AddRenderClient(r renderClient) {
	renderClientList = append(renderClientList, r)
}

var mouseX, mouseY int32

// Get position of mouse
func MouseWhere() (x, y int32) {
	x = int32(mouseX)
	y = int32(mouseY)
	return
}

// Get time in seconds.  Time zero is platform specific.
func Time() float64 {
	return float64(sdl.GetTicks()) * 0.001
}

// Creates a slice of Pixel from a raw pointer
func sliceFromPixelPtr(data unsafe.Pointer, length int) []Pixel {
	var pixels []Pixel
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&pixels))
	sliceHeader.Cap = int(length)
	sliceHeader.Len = int(length)
	sliceHeader.Data = uintptr(data)
	return pixels
}

func lockTexture(tex *sdl.Texture, width int, height int) (pixels []Pixel, pitch int) {
	var data unsafe.Pointer
	err := tex.Lock(nil, &data, &pitch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tex.Lock: %v", err)
		panic(err)
	}
	// Convert pitch units from byte to pixels
	pitch /= 4
	pixels = sliceFromPixelPtr(data, width*height)
	return
}

var winTitle string = "FIXME"
var winWidth, winHeight int = 800, 600

func Run() int {
	// All SDL calls must come from same thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}
	defer sdl.Quit()

	// Create window
	window, err := sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		winWidth, winHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %v\n", err)
		panic(err)
	}
	defer window.Destroy()

	// Create renderer
	width, height := window.GetSize()
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %v\n", err)
		panic(err)
	}
	defer renderer.Destroy()

	// Create texture
	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_STREAMING, width, height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "renderer.CreateTexture: %v\n", err)
		panic(err)
	}
	defer tex.Destroy()

	for _, r := range renderClientList {
		r.Init(int32(width), int32(height))
	}

	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				return 0
			case *sdl.MouseMotionEvent:
				mouseX = int32(t.X)
				mouseY = int32(t.Y)
			case *sdl.KeyUpEvent:
				return 0
			}
		}

		pixels, pitch := lockTexture(tex, width, height)
		pm := MakePixMap(int32(width), int32(height), pixels, int32(pitch))
		for _, r := range renderClientList {
			r.Render(pm)
		}
		tex.Unlock()

		err := renderer.Clear()
		if err != nil {
			fmt.Fprintf(os.Stderr, "renderer.Clear: %v", err)
			panic(err)
		}
		renderer.Copy(tex, nil, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "renderer.Copy: %v", err)
			panic(err)
		}
		renderer.Present()
	}
}

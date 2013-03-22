package team

import (
	"image/color"
)

var colorQueue chan color.Color = make(chan color.Color, 15)
var activeColors chan color.Color = make(chan color.Color, 15)

var initialized bool = false

func Init() {
	colorQueue <- color.RGBA{255,0,0,255}
	colorQueue <- color.RGBA{0,255,0,255}
	colorQueue <- color.RGBA{0,0,255,255}
	colorQueue <- color.RGBA{255,255,255,255}

	initialized = true
}

func GetNextColor() (col color.Color) {
	if initialized == false { Init() }

	select {
	case c := <-colorQueue:
		activeColors <- c
		col = c
	default:
		col = color.Black
	}

	return
}
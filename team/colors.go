package team

import (
	"image/color"
)

colorQueue  = make(chan color.Color, 4)
activeColors = make(chan color.Color, 4)

func Init() {
	colorQueue <- color.RGBA{255,0,0}
	colorQueue <- color.RGBA{0,255,0}
	colorQueue <- color.RGBA{0,0,255}
	colorQueue <- color.RGBA{255,255,255}
}

func GetNext() {
	color := <-colorQueue
	activeColors <- color
}
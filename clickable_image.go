package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type ClickableImage struct {
	widget.BaseWidget
	image    *canvas.Image
	OnTapped func()
}

func (img *ClickableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(img.image)
}

func NewClickableImage(image *canvas.Image, onTapped func()) *ClickableImage {
	clickableImage := &ClickableImage{
		image:    image,
		OnTapped: onTapped,
	}
	clickableImage.ExtendBaseWidget(clickableImage)
	return clickableImage
}

func (img *ClickableImage) Tapped(_ *fyne.PointEvent) {
	img.OnTapped()
}

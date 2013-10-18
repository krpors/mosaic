package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
)

// Calculates the average color used in the specified rectangle in the image.
// Will return a color.RGBA()
func calcAvg(img image.Image, rect image.Rectangle) color.Color {
	var r, g, b, iteration uint32
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			// get colors at this pixel. Docs say the values returned by RGBA are
			// alpha-premultiplied values, where each value ranges between [0-0xFFFF].
			// We'll remove the alpha part by byteshifting if 8 places to the right.
			cr, cg, cb, _ := img.At(x, y).RGBA()
			r += cr >> 8
			g += cg >> 8
			b += cb >> 8
			iteration++
		}
	}

	return color.RGBA{uint8(r / iteration), uint8(g / iteration), uint8(b / iteration), 0xff}
}

// Fills a rectangle in the specified rgba with the given color
func fillRect(rgba *image.RGBA, rect image.Rectangle, color color.Color) {
	for x := rect.Min.X; x <= rect.Max.X; x++ {
		for y := rect.Min.Y; y <= rect.Max.Y; y++ {
			rgba.Set(x, y, color)
		}
	}
}

func divide(img image.Image, rwidth, rheight int) {
	b := img.Bounds()


	// create image to write to:
	rgba := image.NewRGBA(b)

	for x := 0; x < b.Max.X; x += rwidth {
		for y := 0; y < b.Max.Y; y += rheight {
			x2 := x+rwidth
			y2 := y+rheight
			// stay within bounds of the original image here:
			if x2 >= b.Max.X {
				x2 = b.Max.X
			}
			if y2 >= b.Max.Y {
				y2 = b.Max.Y
			}
			bounds := image.Rect(x, y, x2, y2)
			avgcolor := calcAvg(img, bounds)
			fillRect(rgba, bounds, avgcolor)
		}
	}

	of, err := os.Create("/home/dump/LOOOOL.jpg")
	if err != nil {
		log.Fatal(err)
	}

	newImage := rgba.SubImage(b)
	err = jpeg.Encode(of, newImage, &jpeg.Options{80})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	f, err := os.Open("img.jpg")
	if err != nil {
		log.Fatal(err)
	}
	img, err := jpeg.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	divide(img, 610, 60)
}

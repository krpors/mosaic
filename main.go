package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"strings"
)

// Calculates the average color used in the specified rectangle in the image.
func calcAvg(img image.Image, rect image.Rectangle) color.Color {
	var r, g, b int64
	var iteration int64
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			r += int64(cr)
			g += int64(cg)
			b += int64(cb)
			iteration++
		}
	}

	ar := uint16(r / iteration)
	ag := uint16(g / iteration)
	ab := uint16(b / iteration)

	c := color.RGBA64{ar, ag, ab, 0xFFFF}
	return c
}

// Fills a rectangle in the specified rgba with the given color.
func fillRect(rgba *image.RGBA, rect image.Rectangle, color color.Color) {
	for x := rect.Min.X; x <= rect.Max.X; x++ {
		for y := rect.Min.Y; y <= rect.Max.Y; y++ {
			rgba.Set(x, y, color)
		}
	}
}

// This function downscales an image by the specified ratio, using area-averaging.
// For instance, specifying ratio of `2' will downscale the image to half the size
// of the image (width and height will both be divided by 2).
func downscaleRatio(src image.Image, ratio int) image.Image {
	newrect := image.Rect(0, 0, src.Bounds().Max.X/ratio, src.Bounds().Max.Y/ratio)
	rgba := image.NewRGBA(newrect)

	for x := 0; x <= src.Bounds().Max.X; x += ratio {
		for y := 0; y <= src.Bounds().Max.Y; y += ratio {
			sgm := image.Rect(x, y, x+ratio, y+ratio)
			color := calcAvg(src, sgm)
			rgba.Set(x/ratio, y/ratio, color)

		}
	}

	return rgba.SubImage(rgba.Bounds())
}

// Downscales an image to the specified width, maintaining aspect ratio. This will
// call downscaleRatio(), where the specified ratio will be the source's image width
// divided by the specified target width. For example, if the source image width is
// 4000 pixels, the target width is 2000, the aspect ratio will be 4000/2000 = 2.
func downscaleWidth(src image.Image, width int) image.Image {
	return downscaleRatio(src, src.Bounds().Max.X/width)
}

// 'Pixelizes' the given image, with each 'pixel' the width and height of the given
// parameters. Returns a new image as a result.
func pixelize(img image.Image, rwidth, rheight int) image.Image {
	b := img.Bounds()

	// create image to write to:
	rgba := image.NewRGBA(b)

	for x := 0; x < b.Max.X; x += rwidth {
		for y := 0; y < b.Max.Y; y += rheight {
			x2 := x + rwidth
			y2 := y + rheight
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

	newImage := rgba.SubImage(b)
	return newImage
}

// Handy dandy open file.
func openImage(ff string) image.Image {
	f, err := os.Open(ff)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	var img image.Image
	if strings.HasSuffix(ff, ".png") {
		img, err = png.Decode(f)
	} else if strings.HasSuffix(ff, ".jpg") {
		img, err = jpeg.Decode(f)
	} else {
		log.Fatal("unrecognized image format")
	}

	if err != nil {
		log.Fatal(err)
	}

	return img
}

// Handy dandy write image.
func writeImage(ff string, img image.Image) {
	of, err := os.Create(ff)
	if err != nil {
		log.Fatal(err)
	}

	if strings.HasSuffix(ff, ".png") {
		err = png.Encode(of, img)
	} else if strings.HasSuffix(ff, ".jpg") {
		jpeg.Encode(of, img, &jpeg.Options{100})
	} else {
		log.Fatal("unrecognized image format")
	}
}

func main() {
	img := openImage("text.jpg")
	simg := downscaleRatio(img, 3) 
	writeImage("/home/dump/lol2.jpg", simg)
}

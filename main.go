package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
)

func calcAvg(img image.Image) color.Color {
	fmt.Println(img.Bounds())
	var r, g, b, iteration uint32
	for x := 0; x < img.Bounds().Max.X; x++ {
		for y := 0; y < img.Bounds().Max.Y; y++ {
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

func main() {
	of, err := os.Create("lol.jpg")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	_ = color.White
	myimg := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			myimg.Set(x, y, color.RGBA{0, 55, 210, 255})
		}
	}

	subimg := myimg.SubImage(image.Rect(0, 0, 100, 100))
	avgcolor := calcAvg(subimg)

	err = jpeg.Encode(of, myimg, &jpeg.Options{90})
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	fmt.Println("Average color: ", avgcolor)
	os.Exit(0)
}

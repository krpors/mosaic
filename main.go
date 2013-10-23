package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Worker pools:
// http://play.golang.org/p/zfn5t52w4p (laz`)
// http://play.golang.org/p/ssMGqjQw4q (e-dard)

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

// Opens a file, try to decode it as png or jpg, and return the image instance. Or
// obviously an error when the shit hit the fan.
func openImage(ff string) (image.Image, error) {
	f, err := os.Open(ff)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	var img image.Image
	if strings.HasSuffix(ff, ".png") {
		img, err = png.Decode(f)
	} else if strings.HasSuffix(ff, ".jpg") {
		img, err = jpeg.Decode(f)
	} else {
		return nil, fmt.Errorf("unrecognized image format for file '%s'", ff)
	}

	if err != nil {
		return nil, err
	}

	return img, nil
}

// Handy dandy write image. Returns an error when the file cannot be created, or
// encoding the image failed.
func writeImage(ff string, img image.Image) error {
	of, err := os.Create(ff)
	if err != nil {
		return err
	}

	if strings.HasSuffix(ff, ".png") {
		err = png.Encode(of, img)
	} else if strings.HasSuffix(ff, ".jpg") {
		err = jpeg.Encode(of, img, &jpeg.Options{100})
	} else {
		err = fmt.Errorf("unrecognized image format '%s'", ff)
	}

	return err
}

// Struct containing RGBA values for an image. If err is not nil, the particular
// instance of this struct should be ignored by analyzeFiles()
type imageInfo struct {
	Path  string
	Red   uint32
	Green uint32
	Blue  uint32
	Alpha uint32

	// If a file failed to be read, this will be filled and must be discarded.
	// Since this struct will be sent over a channel, and we cannot send nil
	// values over this channel, we'll be sending an error in an imageInfo
	// struct instance instead.
	err error
}

// Worker to analyze image files, by calculating the average color for that image.
// c is channel where files are received on , and result is the channel where the
// results are sent to.
func worker(c chan string, result chan imageInfo) {
	for path := range c {
		var info imageInfo
		info.Path = path

		img, err := openImage(path)
		if err != nil {
			info.err = err
		} else {
			avg := calcAvg(img, img.Bounds())
			r, g, b, _ := color.NRGBAModel.Convert(avg).RGBA()

			info.Red = r
			info.Green = g
			info.Blue = b
		}

		result <- info
	}
}

// Iterates over the 'files' slice, and finds per file the average color used.
// The analyzing happens in parallel, over separate goroutines ('workers'):
//
// 1. An x amount of worker goroutines are spawned (the worker() function), using
//    two channels: a fileChan to 'send' the files to, so the worker 'picks them up',
//    and a resultChan, where the results of the worker are sent to.
// 2. A goroutine is created which reads from the resultChan, which receives results
//	  once they are ready.
// 3. The files are sent to the fileChan
// 4. The fileChan is closed to prevent deadlock
// 5. Once all results are read in the goroutine from step 2, the cumulative results
//    are sent to the doneChan channel.
//
// TODO: make the amount of workers configurable?
func analyzeFiles(files []string) {
	// temporary container struct to serialize images to JSON
	type container struct {
		Info []imageInfo
	}

	fileChan := make(chan string)      // used to send the files to the workers
	resultChan := make(chan imageInfo) // used to receive individual results
	doneChan := make(chan container)   // used to post the cumulative result

	// Create the workers here, and spawn them, wait for work to do.
	maxjobs := 3
	for i := 0; i < maxjobs; i++ {
		go worker(fileChan, resultChan)
	}

	// Seperate goroutine to receive results from worker, thanks laz` and e-dard
	go func() {
		cont := container{}
		for _ = range files {
			info := <-resultChan
			if info.err != nil {
				fmt.Println(info.err)
			} else {
				fmt.Printf("Processed file '%v'\n", info.Path)
				cont.Info = append(cont.Info, info)
			}
		}
		// send the cumulative results to the done channel, so the function
		// can finish up.
		doneChan <- cont
	}()

	// Send each path to the worker:
	for _, path := range files {
		fileChan <- path
	}

	// close the file channel; nothing is to be sent to this channel anymore.
	// Without this, the runtime will report a deadlock.
	close(fileChan)

	// Finally, wait until everything is done ...
	allImageInfo := <-doneChan

	// ... and write it to JSON here.
	bytes, err := json.MarshalIndent(allImageInfo, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	of, err := os.Create("output.json")
	defer of.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	of.Write(bytes)
}

func main() {
	files := make([]string, 0)
	wf := func(path string, fi os.FileInfo, err error) error {
		if !fi.IsDir() {
			files = append(files, path)
		}

		return nil
	}

	filepath.Walk(".", wf)

	analyzeFiles(files)
}

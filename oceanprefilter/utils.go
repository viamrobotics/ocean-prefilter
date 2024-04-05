package oceanprefilter

import (
	"fmt"
	"image"
	"image/draw"

	"gocv.io/x/gocv"
)

func findHorizonLine(pic image.Image) ([]image.Point, error) {
	// make gray
	gray := toGray(pic)
	grayMat, err := gocv.ImageGrayToMatGray(gray)
	if err != nil {
		return nil, err
	}
	defer grayMat.Close()
	gocv.GaussianBlur(grayMat, &grayMat, image.Pt(3, 3), 0, 0, gocv.BorderDefault)
	// threshold to turn it into a binary image according to Otsu's method
	_ = gocv.Threshold(grayMat, &grayMat, 0, 255, gocv.ThresholdOtsu)
	oneScalar := gocv.NewScalar(255, 0, 0, 0)
	kernelMorph := gocv.NewMatWithSize(9, 9, gocv.MatTypeCV8U)
	defer kernelMorph.Close()
	kernelMorph.SetTo(oneScalar)
	// smooth out holes with morphology transform
	gocv.MorphologyEx(grayMat, &grayMat, gocv.MorphClose, kernelMorph)

	horizonX1 := 0
	horizonX2 := pic.Bounds().Max.X - 1
	horizonY1 := 0
	horizonY2 := 0
	for y := 0; y < grayMat.Rows(); y++ { // For Mat, y is in the first dimension, the rows
		// Check the pixel at [y, horizonX1]
		pixelVal1 := grayMat.GetUCharAt(y, horizonX1)
		if pixelVal1 == 255 {
			horizonY1 = y
		}
		// Check the pixel at [y, horizonX2]
		pixelVal2 := grayMat.GetUCharAt(y, horizonX2)
		if pixelVal2 == 255 {
			horizonY2 = y
		}
	}
	return []image.Point{{horizonX1, horizonY1}, {horizonX2, horizonY2}}, nil
}

func toGray(pic image.Image) *image.Gray {
	if g, ok := pic.(*image.Gray); ok {
		return g
	}
	gray := image.NewGray(pic.Bounds())
	draw.Draw(gray, gray.Bounds(), pic, pic.Bounds().Min, draw.Src)
	return gray
}

// crop the image from yValue -> img.Bounds().Max.Y
// and then split the cropped image into n horizontal bands of equal height
func splitUpImage(img image.Image, yValue, n int) ([]image.Image, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be greater than 0")
	}

	// Crop the image from yValue to img.Bounds().Max.Y
	bounds := img.Bounds()
	croppedHeight := bounds.Max.Y - yValue
	if croppedHeight <= 0 {
		return nil, fmt.Errorf("yValue must be within the image bounds")
	}
	croppedRect := image.Rect(bounds.Min.X, yValue, bounds.Max.X, bounds.Max.Y)
	croppedImg := image.NewRGBA(croppedRect)
	draw.Draw(croppedImg, croppedImg.Bounds(), img, image.Point{bounds.Min.X, yValue}, draw.Src)

	// Split the cropped image into n horizontal bands
	imageWidth := croppedImg.Bounds().Dx()
	bandHeight := croppedImg.Bounds().Dy() / n
	images := make([]image.Image, n)
	for i := 0; i < n; i++ {
		bandRect := image.Rect(0, i*bandHeight, imageWidth, (i+1)*bandHeight)
		bandImg := image.NewRGBA(bandRect)
		draw.Draw(bandImg, bandImg.Bounds(), croppedImg, bandRect.Min, draw.Src)
		images[i] = bandImg
	}

	return images, nil
}

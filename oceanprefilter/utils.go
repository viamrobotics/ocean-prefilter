package oceanprefilter

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/pkg/errors"
	"go.viam.com/rdk/rimage"
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
// and then split the cropped image into nh horizontal and nv vertical bands of equal height and width
func splitUpImage(img image.Image, exZone image.Rectangle, yValue, nh, nv int) ([]image.Image, error) {
	if img == nil {
		return nil, errors.New("input image to split up is nil")
	}
	if nv <= 0 {
		return nil, fmt.Errorf("number of vertical lines must be greater than 0")
	}
	if nh <= 0 {
		return nil, fmt.Errorf("number of horizontal lines must be greater than 0")
	}

	// Crop the image from yValue to img.Bounds().Max.Y
	bounds := img.Bounds()
	croppedHeight := bounds.Max.Y - yValue
	if croppedHeight <= 0 {
		return nil, fmt.Errorf("yValue must be within the image bounds")
	}
	// edit exluded zone to take the crop into account
	exZone.Min.Y = exZone.Min.Y - yValue
	exZone.Max.Y = exZone.Max.Y - yValue
	croppedRect := image.Rect(bounds.Min.X, yValue, bounds.Max.X, bounds.Max.Y)
	croppedImg := image.NewRGBA(croppedRect)
	draw.Draw(croppedImg, croppedImg.Bounds(), img, croppedRect.Min, draw.Src)
	rimage.SaveImage(croppedImg, "/tmp/croppedImg.jpg")

	// Split the cropped image into n horizontal bands
	bandHeight := croppedImg.Bounds().Dy() / nh
	bandWidth := croppedImg.Bounds().Dx() / nv
	images := make([]image.Image, 0, nv*nh)
	for i := 0; i < nh; i++ {
		for j := 0; j < nv; j++ {
			bandRect := image.Rect(j*bandWidth, i*bandHeight, (j+1)*bandWidth, (i+1)*bandHeight)
			// if rect in excluded zone, skip it
			if bandRect.Overlaps(exZone) {
				continue
			}
			bandImg := image.NewRGBA(bandRect)
			draw.Draw(bandImg, bandImg.Bounds(), croppedImg, image.Point{bandRect.Min.X, bandRect.Min.Y + yValue}, draw.Src)
			images = append(images, bandImg)
			rimage.SaveImage(bandImg, fmt.Sprintf("/tmp/%v_%v.jpg", i, j))
		}
	}
	return images, nil
}

package oceanprefilter

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
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
// and then split the cropped image into nh horizontal and nv vertical bands of equal height and width (dimensions given)
func splitUpImageConst(img image.Image, exZone *image.Rectangle, yValue, h, w int) ([]image.Image, error) {
	if img == nil {
		return nil, errors.New("input image to split up is nil")
	}
	if h <= 0 {
		return nil, fmt.Errorf("height must be greater than 0, got %v", h)
	}
	if w <= 0 {
		return nil, fmt.Errorf("width must be greater than 0, got %v", w)
	}

	// Crop the image from yValue to img.Bounds().Max.Y
	bounds := img.Bounds()
	croppedHeight := bounds.Max.Y - yValue
	if croppedHeight <= 0 {
		return nil, errors.New("yValue must be within the image bounds")
	}
	// edit exluded zone to take the crop into account
	excludedBox := image.Rectangle{}
	if exZone != nil {
		excludedBox.Min.X = exZone.Min.X
		excludedBox.Min.Y = exZone.Min.Y - yValue
		excludedBox.Max.X = exZone.Max.X
		excludedBox.Max.Y = exZone.Max.Y - yValue
	}
	croppedRect := image.Rect(0, yValue, bounds.Max.X, bounds.Max.Y)
	croppedImg := image.NewRGBA(image.Rect(0, 0, croppedRect.Dx(), croppedRect.Dy()))
	draw.Draw(croppedImg, croppedImg.Bounds(), img, croppedRect.Min, draw.Src)

	// Split the cropped image into n horizontal bands
	nv := int(math.Ceil(float64(croppedImg.Bounds().Dy()) / float64(h)))
	nh := int(math.Ceil(float64(croppedImg.Bounds().Dx()) / float64(w)))
	images := make([]image.Image, 0, w*h)
	edgeX := croppedImg.Bounds().Max.X
	edgeY := croppedImg.Bounds().Max.Y

	for i := 0; i < nv; i++ {
		for j := 0; j < nh; j++ {
			flag := false
			xEnd := (j+1)*w
			yEnd := (i+1)*h
			//bounds checking
			if xEnd >= edgeX {
				xEnd = edgeX - 1
				flag = true
			}
			if yEnd >= edgeY {
				yEnd = edgeY - 1
				flag = true
			}
			bandRect := image.Rect(j*w, i*h, xEnd, yEnd)
			// if rect in excluded zone, skip it
			if exZone != nil && bandRect.Overlaps(excludedBox) {
				continue
			}
			bandImg := image.NewRGBA(bandRect)
			draw.Draw(bandImg, bandImg.Bounds(), croppedImg, image.Point{bandRect.Min.X, bandRect.Min.Y}, draw.Src)

			if flag {
				resized := imaging.Resize(bandImg, w, h, imaging.Lanczos)
				images = append(images, resized)
			} else {
				images = append(images, bandImg)
			}
		}
	}
	return images, nil
}

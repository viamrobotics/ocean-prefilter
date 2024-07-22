package oceanprefilter

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/pkg/errors"
)

const NVSplit = 4
const NHSplit = 4

type Bucket struct {
	Count int
}

type Histogram struct {
	Count   int
	Buckets []Bucket
	spacing float64
}

func Hist(buckets int, minVal, maxVal float64, input []float64) Histogram {
	// make evenly spaced buckets between min and max
	// values outside of min and max are discarded
	space := (maxVal - minVal) / float64(buckets)
	bkts := make([]Bucket, buckets)
	hist := Histogram{
		Count:   0,
		Buckets: bkts,
		spacing: space,
	}
	for _, val := range input {
		if val >= maxVal || val < minVal { // skip over out of range values
			continue
		}
		bucketIndex := int((val - minVal) / space)
		if bucketIndex < 0 || bucketIndex > len(hist.Buckets)-1 {
			panic(fmt.Sprintf("value of %v out of range between (%v, %v)", val, maxVal, minVal))
		}
		hist.Count++
		hist.Buckets[bucketIndex].Count++
	}
	return hist
}

func histogramChangeFilter(
	oldHists []Histogram, newImg image.Image, rc RunConfig,
) (bool, []Histogram, error) {
	firstHist := false
	thresh := rc.Threshold
	if len(oldHists) == 0 {
		firstHist = true // get the data for the first histogram
	}
	// find the horizon, take the average y value
	linePoints, err := findHorizonLine(newImg)
	if err != nil {
		return false, nil, err
	}
	if len(linePoints) < 2 {
		return false, nil, errors.New("function to find the horizon line returned less than 2 points")
	}
	cropY := int(math.Max(float64(linePoints[0].Y), float64(linePoints[1].Y)))
	if rc.debug {
		rc.logger.Debugf("found horizon at y = %v", cropY)
	}
	if cropY >= (newImg.Bounds().Max.Y-1) || cropY <= 1 {
		return false, nil, errors.Errorf("could not find horizon in image. Got a horizon value of y = %v", cropY)
	}
	imgs, err := splitUpImage(newImg, rc.ExcludedZone, cropY, NHSplit, NVSplit)
	if err != nil {
		return false, nil, err
	}
	newHists := []Histogram{} // a histogram for each RGB channel
	trigger := false
	if !firstHist && len(imgs) != len(oldHists) {
		return false, nil, errors.New("image changed drastically, cannot evaluate histogram difference. Can be caused by large amounts of motion")
	}
	for i, img := range imgs {
		resultHists := createGrayHistograms(img)
		for j, h := range resultHists {
			newHist := h
			splitTrigger := false
			if !firstHist {
				oldHist := oldHists[i*len(resultHists)+j]
				if len(oldHist.Buckets) != len(newHist.Buckets) {
					return false, nil, errors.Errorf("hists should have same number of buckets, old hist: %v, new hist: %v", len(oldHist.Buckets), len(newHist.Buckets))
				}
				splitTrigger = histogramTrigger(oldHist, newHist, thresh)
			}
			newHists = append(newHists, newHist)
			if splitTrigger {
				trigger = true
			}
		}
	}
	return trigger, newHists, nil
}

// take in an old image histogram, a new image, and return a trigger bool and the new image histogram
// returns a bool based on the thresh value
func histogramTrigger(oldHist, newHist Histogram, thresh float64) bool {
	//result := basicCompare(oldHist, newHist)
	ecdf1 := histogramToECDF(oldHist)
	ecdf2 := histogramToECDF(newHist)
	result := kolmogorovSmirnovTest(ecdf1, ecdf2)
	trigger := false
	if result >= thresh {
		trigger = true
	}
	return trigger
}

func convertToFloat64(uint8Slice []uint8) []float64 {
	float64Slice := make([]float64, len(uint8Slice))

	for i, v := range uint8Slice {
		float64Slice[i] = float64(v)
	}
	return float64Slice
}

func createGrayHistograms(pic image.Image) []Histogram {
	hists := make([]Histogram, 0, 1)
	img := toGray(pic)
	pix := []float64{}
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			c := img.GrayAt(x, y)
			pix = append(pix, float64(c.Y))
		}
	}
	hists = append(hists, Hist(32, 0, 256, pix)) // 32 bin, 8 values in each bin in 255 total
	return hists
}

func createColorHistograms(pic image.Image) []Histogram {
	hists := make([]Histogram, 0, 3)
	img := image.NewRGBA(pic.Bounds())
	if rgbaImg, ok := pic.(*image.RGBA); ok {
		img = rgbaImg
	} else {
		draw.Draw(img, pic.Bounds(), pic, pic.Bounds().Min, draw.Src)
	}
	rPix := []float64{}
	gPix := []float64{}
	bPix := []float64{}
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			c := img.RGBAAt(x, y)
			rPix = append(rPix, float64(c.R))
			gPix = append(gPix, float64(c.G))
			bPix = append(bPix, float64(c.B))
		}
	}
	hists = append(hists, Hist(32, 0, 256, rPix)) // 32 bin, 8 values in each bin in 255 total
	hists = append(hists, Hist(32, 0, 256, gPix)) // 32 bin, 8 values in each bin in 255 total
	hists = append(hists, Hist(32, 0, 256, bPix)) // 32 bin, 8 values in each bin in 255 total
	return hists
}

// basicCompare compares two histograms and returns a measure of their difference
func basicCompare(hist1, hist2 Histogram) float64 {
	var sum float64
	for i := range hist1.Buckets {
		diff := hist1.Buckets[i].Count - hist2.Buckets[i].Count
		if diff < 0 {
			diff = -diff
		}
		sum += float64(diff)
	}
	// Normalize the difference based on the number of pixels
	totalPixels := float64(hist1.Count + hist2.Count) // hist2 has the same number of pixels
	score := sum / totalPixels
	return score
}

// histogramToECDF converts a histogram to an empirical cumulative distribution function (ECDF)
// Assumes histogram bins are evenly distributed over the data range
func histogramToECDF(hist Histogram) []float64 {
	total := float64(hist.Count)
	ecdf := make([]float64, len(hist.Buckets))
	cumulativeCount := 0.0
	if total == 0 {
		return ecdf
	}
	for i, bkt := range hist.Buckets {
		cumulativeCount += float64(bkt.Count)
		ecdf[i] = cumulativeCount / total
	}
	return ecdf
}

// kolmogorovSmirnovTest computes the Kolmogorov-Smirnov statistic for two ECDFs
func kolmogorovSmirnovTest(ecdf1, ecdf2 []float64) float64 {
	maxDiff := 0.0
	for i := range ecdf1 {
		diff := math.Abs(ecdf1[i] - ecdf2[i])
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

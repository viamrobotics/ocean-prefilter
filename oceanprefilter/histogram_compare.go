package oceanprefilter

import (
	"image"
	"math"

	"github.com/pkg/errors"
)

const NSplit = 3

func histogramChangeFilter(
	oldHists [][]float64, newImg image.Image, thresh float64,
) (bool, [][]float64, error) {
	if len(oldHists) != NSplit {
		return false, nil, errors.Errorf("only have %v old histograms, expected %v", len(oldHists), NSplit)
	}
	// find the horizon, take the average y value
	linePoints, err := findHorizonLine(newImg)
	if err != nil {
		return false, nil, err
	}
	if len(linePoints) < 2 {
		return false, nil, errors.New("function to find the horizon line returned less than 2 points")
	}
	avgY := int((float64(linePoints[0].Y) + float64(linePoints[1].Y)) / 2.0)
	imgs, err := splitUpImage(newImg, avgY, NSplit)
	if err != nil {
		return false, nil, err
	}
	newHists := make([][]float64, len(imgs))
	trigger := false
	for i, img := range imgs {
		splitTrigger, newHist := histogramTrigger(oldHists[i], img, thresh)
		newHists[i] = newHist
		if splitTrigger {
			trigger = true
		}
	}
	return trigger, newHists, nil
}

// take in an old image histogram, a new image, and return a trigger bool and the new image histogram
// returns a bool based on the thresh value
func histogramTrigger(oldHist []float64, newImg image.Image, thresh float64) (bool, []float64) {
	newHist := createHistogram(newImg)
	result := basicCompare(oldHist, newHist)
	trigger := false
	if result >= thresh {
		trigger = true
	}
	return trigger, newHist
}

func createHistogram(pic image.Image) []float64 {
	histogram := make([]float64, 256) // 256 bins for grayscale image
	img := toGray(pic)
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayColor := img.GrayAt(x, y)
			histogram[grayColor.Y] += 1.0
		}
	}
	return histogram
}

// basicCompare compares two histograms and returns a measure of their difference
func basicCompare(hist1, hist2 []float64) float64 {
	var sum float64
	for i := range hist1 {
		diff := hist1[i] - hist2[i]
		if diff < 0 {
			diff = -diff
		}
		sum += diff
	}
	// Normalize the difference based on the number of pixels
	totalPixels := sumOf(hist1) + sumOf(hist2)
	return sum / totalPixels
}

func sumOf(slice []float64) float64 {
	sum := 0
	for _, value := range slice {
		sum += value
	}
	return sum
}

// histogramToECDF converts a histogram to an empirical cumulative distribution function (ECDF)
// Assumes histogram bins are evenly distributed over the data range
func histogramToECDF(histogram []float64) []float64 {
	total := 0.0
	for _, count := range histogram {
		total += count
	}

	ecdf := make([]float64, len(histogram))
	cumulativeCount := 0.0
	for i, count := range histogram {
		cumulativeCount += count
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

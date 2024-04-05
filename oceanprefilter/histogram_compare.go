package oceanprefilter

import (
	"image"
	"math"
)

func createHistogram(pic image.Image) []int {
	histogram := make([]int, 256) // 256 bins for grayscale image
	img := toGray(pic)
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayColor := img.GrayAt(x, y)
			histogram[grayColor.Y]++
		}
	}
	return histogram
}

// basicCompare compares two histograms and returns a measure of their difference
func basicCompare(hist1, hist2 []int) float64 {
	var sum int
	for i := range hist1 {
		diff := hist1[i] - hist2[i]
		if diff < 0 {
			diff = -diff
		}
		sum += diff
	}
	// Normalize the difference based on the number of pixels
	totalPixels := sumOf(hist1) + sumOf(hist2)
	return float64(sum) / float64(totalPixels)
}

func sumOf(slice []int) int {
	sum := 0
	for _, value := range slice {
		sum += value
	}
	return sum
}

// histogramToECDF converts a histogram to an empirical cumulative distribution function (ECDF)
// Assumes histogram bins are evenly distributed over the data range
func histogramToECDF(histogram []int) []float64 {
	total := 0
	for _, count := range histogram {
		total += count
	}

	ecdf := make([]float64, len(histogram))
	cumulativeCount := 0
	for i, count := range histogram {
		cumulativeCount += count
		ecdf[i] = float64(cumulativeCount) / float64(total)
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

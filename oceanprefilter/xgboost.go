package oceanprefilter

import (
	"fmt"
	"image"
	"github.com/Elvenson/xgboost-go/activation"
	"github.com/Elvenson/xgboost-go/mat"
	// "github.com/Elvenson/xgboost-go/models"
)

// Vector represents a dense vector.
type Vector []float32

// SparseVector is a map with index as a key and value as a value at that index.
type SparseVector map[int]float32

// SparseMatrix is a list of sparse vectors.
type SparseMatrix struct {
	Vectors []SparseVector
}


// ConvertImageToSparseMatrix converts an image to a SparseMatrix.
func ConvertImageToSparseMatrix(img image.Image) SparseMatrix {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	matrix := SparseMatrix{
		Vectors: make([]SparseVector, height),
	}

	for y := 0; y < height; y++ {
		sparseVector := make(SparseVector)
		for x := 0; x < width; x++ {
			// Extract RGB values from the image
			r, g, b, _ := img.At(x, y).RGBA()

			// Compute average of RGB values
			avg := (float32(r/257) + float32(g/257) + float32(b/257)) / 3.0 // 257 is the divisor to scale down from 16-bit to 8-bit range

			if avg > 0 { // Store non-zero pixels
				sparseVector[x] = avg
			}
		}
		matrix.Vectors[y] = sparseVector
	}

	return matrix
}

func inference(input image.Image, rc runConfig) (bool, error) {
	ensemble, err := models.LoadXGBoostFromJSON("xg_boost_dump.json",
		"", 1, 4, &activation.Logistic{})
	if err != nil {
		panic(err)
	}

	// *******
	thresh := rc.threshold
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
	imgs, err := splitUpImage(newImg, rc.excludedZone, cropY, NHSplit, NVSplit)
	if err != nil {
		return false, nil, err
	}

	// new code -> checks if any square is interesting
	trigger := false
	for i, img := range imgs {
		// change image format
		img = ConvertImageToSparseMatrix(img)
		result, err := ensemble.PredictProba(img)
		if err != nil {
			panic(err)
		}
		
		if result {
			return true
		}
	}

// ******************

	// fmt.Printf("%+v\n", triggers)
	return false, nil
}
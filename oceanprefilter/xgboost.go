package oceanprefilter

import (
	"image"
	"math"

	"github.com/Elvenson/xgboost-go/mat"
	"github.com/pkg/errors"
)

func flatten(matrix mat.SparseMatrix) mat.SparseMatrix {
	flatVector := mat.SparseVector{}
	offset := 0

	for _, vector := range matrix.Vectors {
		for idx, value := range vector {
			flatIndex := offset + idx
			flatVector[flatIndex] = value
		}
		offset += len(vector)
	}

	flatMatrix := mat.SparseMatrix{
		Vectors: []mat.SparseVector{flatVector},
	}

	return flatMatrix
}

func avgPoolFull(img image.Image, patchSize image.Point) mat.SparseMatrix {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	patchWidth, patchHeight := patchSize.X, patchSize.Y

	h := height / patchHeight
	w := width / patchWidth

	downsize := make([]mat.SparseVector, h)

	for i := 0; i < h; i++ {
		row := make(mat.SparseVector)
		for j := 0; j < w; j++ {
			sum := 0.0
			count := 0
			for x := 0; x < patchWidth; x++ {
				for y := 0; y < patchHeight; y++ {
					px := bounds.Min.X + j*patchWidth + x
					py := bounds.Min.Y + i*patchHeight + y
					r, g, b, a := img.At(px, py).RGBA()
					avg := float64(r+g+b) / 3.0 / 256.0
					if a > 0 {
						sum += avg
						count++
					}
				}
			}
			if count > 0 {
				row[j] = float32(sum / float64(count))
			}
		}
		downsize[i] = row
	}

	return mat.SparseMatrix{Vectors: downsize}

}
func MakeInference(input image.Image, rc RunConfig) (bool, error) {
	// find the horizon, take the average y value
	linePoints, err := findHorizonLine(input)
	if err != nil {
		return false, err
	}
	if len(linePoints) < 2 {
		return false, errors.New("function to find the horizon line returned less than 2 points")
	}
	cropY := int(math.Max(float64(linePoints[0].Y), float64(linePoints[1].Y)))
	if rc.debug {
		rc.logger.Debugf("found horizon at y = %v", cropY)
	}
	if cropY >= (input.Bounds().Max.Y-1) || cropY <= 1 {
		return false, errors.Errorf("could not find horizon in image. Got a horizon value of y = %v", cropY)
	}
	imgs, err := splitUpImageConst(input, rc.ExcludedZone, cropY, 80, 200)
	if err != nil {
		return false, err
	}

	// new code -> checks if any square is interesting
	for _, img := range imgs {
		in_mat := avgPoolFull(img, image.Point{10, 2})
		in_mat = flatten(in_mat)

		result, err := rc.Model.Predict(in_mat)
		if err != nil {
			panic(err)
		}

		if (*result.Vectors[0])[0] == 1 {
			return true, nil
		}
	}

	return false, nil
}

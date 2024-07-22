package oceanprefilter

import (
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"

	xgb "github.com/Elvenson/xgboost-go"
	"github.com/Elvenson/xgboost-go/activation"
	"go.viam.com/test"
)

func TestXGBoostInference(t *testing.T) {
	f, err := os.Open("test_data/2288.jpg")
	defer f.Close()
	img, _, err := image.Decode(f)
	test.That(t, err, test.ShouldBeNil)

	rc := RunConfig{}
	ensemble, err := xgb.LoadXGBoostFromJSONBytes(modelbytes,
		"", 2, 8, &activation.Softmax{})
	test.That(t, err, test.ShouldBeNil)

	rc.Model = ensemble
	rc.Threshold = 0.25
	rect := image.Rectangle{
		Min: image.Point{X: 250, Y: 350},
		Max: image.Point{X: 580, Y: 480},
	}
	rc.ExcludedZone = &rect

	res, err := MakeInference(img, rc)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, res, test.ShouldBeTrue)
}

func TestSplitData(t *testing.T) {
	dir := "test_data/2288.jpg/"
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if file.Name() == ".DS_Store" {
			continue
		}
		fp := filepath.Join(dir, file.Name())
		f, err := os.Open(fp)
		defer f.Close()
		img, _, err := image.Decode(f)
		test.That(t, err, test.ShouldBeNil)

		rc := RunConfig{}
		rc.Threshold = 0.25
		rect := image.Rectangle{
			Min: image.Point{X: 250, Y: 350},
			Max: image.Point{X: 580, Y: 480},
		}
		rc.ExcludedZone = &rect
		linePoints, err := findHorizonLine(img)
		cropY := int(math.Max(float64(linePoints[0].Y), float64(linePoints[1].Y)))

		_, err = splitUpImageConst(img, rc.ExcludedZone, cropY, 80, 200)
		test.That(t, err, test.ShouldBeNil)

	}
}

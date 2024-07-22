package oceanprefilter

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"
	"image/jpeg"
	xgb "github.com/Elvenson/xgboost-go"
	"github.com/Elvenson/xgboost-go/activation"
)


func TestGeneralModel(t * testing.T) {

	f, err := os.Open("numbered_data/1000.jpg")
	fmt.Print(err)
	defer f.Close()
	img, _, err := image.Decode(f)

	rc := runConfig{}
	ensemble, err := xgb.LoadXGBoostFromJSONBytes(modelbytes,
		"", 2, 8, &activation.Softmax{}) // wrong activation logistic is 1 class only
	if err != nil {
		panic(err)
	}
	rc.model = ensemble
	rc.threshold = 0.25
	rect := image.Rectangle{
		Min: image.Point{X: 250, Y: 350},
		Max: image.Point{X: 580, Y: 480},
	}
	rc.excludedZone = &rect
	
	res, err := make_inference(img, rc)
	if err != nil {
		fmt.Printf("err")
	}

	fmt.Printf("output result is %+v\n", res)
}

func TestAcc(t * testing.T) {
	dir := "numbered_data/"
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if file.Name() == ".DS_Store" {
			continue
		}
		fp := filepath.Join(dir, file.Name())
		fmt.Println(fp)
		f, err := os.Open(fp)
		defer f.Close()
		img, _, err := image.Decode(f)

		rc := runConfig{}
		ensemble, err := xgb.LoadXGBoostFromJSONBytes(modelbytes,
			"", 2, 8, &activation.Softmax{}) // wrong activation logistic is 1 class only
		if err != nil {
			panic(err)
		}

		rc.model = ensemble
		rc.threshold = 0.25
		rect := image.Rectangle{
			Min: image.Point{X: 250, Y: 350},
			Max: image.Point{X: 580, Y: 480},
		}
		rc.excludedZone = &rect
		
		res, err := make_inference(img, rc)
		if res {

		} else {
			
		}
	}
}

func TestSplitData(t * testing.T) {
	dir := "triggers/2/"
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if file.Name() == ".DS_Store" {
			continue
		}
		fp := filepath.Join(dir, file.Name())
		fmt.Println(fp)
		f, err := os.Open(fp)
		defer f.Close()
		img, _, err := image.Decode(f)

		rc := runConfig{}
		rc.threshold = 0.25
		rect := image.Rectangle{
			Min: image.Point{X: 250, Y: 350},
			Max: image.Point{X: 580, Y: 480},
		}
		rc.excludedZone = &rect
		linePoints, err := findHorizonLine(img)
		cropY := int(math.Max(float64(linePoints[0].Y), float64(linePoints[1].Y)))

		imgs, err := splitUpImageConst(img, rc.excludedZone, cropY, 80, 200)
		if err != nil {
			fmt.Printf("err")
		}
		
		for idx, img := range imgs {
			////
			filename := fmt.Sprintf("temp/%d_%s", idx, file.Name())
			file, _ := os.Create(filename)
			defer file.Close()
			jpeg.Encode(file, img, nil)
			////
		}
	}
}

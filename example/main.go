package main

import (
	_ "embed"
	"fmt"
	"image"
	"os"
	"path/filepath"

	xgb "github.com/Elvenson/xgboost-go"
	"github.com/Elvenson/xgboost-go/activation"
	"github.com/viamrobotics/ocean-prefilter/oceanprefilter"
)

//go:embed xg_boost_dump.json
var modelbytes []byte

func main() {
	dir := os.Args[1]
	fmt.Println(dir)
	files, _ := os.ReadDir(dir)

	rc := oceanprefilter.RunConfig{}
	ensemble, err := xgb.LoadXGBoostFromJSONBytes(modelbytes,
		"", 2, 8, &activation.Softmax{})

	if err != nil {
		fmt.Println(err)
	}

	rc.Model = ensemble
	rc.Threshold = 0.25
	rect := image.Rectangle{
		Min: image.Point{X: 250, Y: 350},
		Max: image.Point{X: 580, Y: 480},
	}

	rc.ExcludedZone = &rect
	for _, file := range files {
		fp := filepath.Join(dir, file.Name())
		f, _ := os.Open(fp)

		defer f.Close()
		img, _, _ := image.Decode(f)

		res, err := oceanprefilter.MakeInference(img, rc)
		if err != nil {
			fmt.Println("error making inference: ", err)
		} else {
			fmt.Printf("output is %t: ", res)
		}
	}
}

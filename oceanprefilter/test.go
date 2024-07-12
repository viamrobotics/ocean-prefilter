package oceanprefilter


import (
	"fmt"
	"image"
	"os"
)

f, err := os.Open("")

defer f.Close()
image, _, err := image.Decode(f)

rc := runConfig{}
rc.threshold = 0.25

res, err = inference(f, rc)
if err != nil {
	fmt.Printf("err")
}

fmt.Printf("%+v\n", res)
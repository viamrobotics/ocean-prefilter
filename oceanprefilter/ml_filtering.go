package oceanprefilter

import (
	"context"
	"image"
)

func mlFilter(ctx context.Context, img image.Image, rc RunConfig) (bool, error) {
	// first try if the optional detection is present
	if rc.detector == nil {
		return false, nil
	}
	dets, _ := rc.detector.Detections(ctx, img, nil)
	if dets == nil {
		return false, nil
	}
	if rc.chosenLabels == nil {
		if len(dets) != 0 {
			return true, nil
		}
		return false, nil
	}
	for _, d := range dets {
		if conf, ok := rc.chosenLabels[d.Label()]; ok {
			if d.Score() >= conf {
				return true, nil
			}
		}
	}
	return false, nil
}

package oceanprefilter

import (
	"context"
	"image"
	"image/color"
	"sync/atomic"
	"testing"
	"unsafe"
    "os"
	"go.viam.com/rdk/services/vision"
	"go.viam.com/rdk/vision/classification"
	"go.viam.com/rdk/vision/viscapture"
	"go.viam.com/test"

    xgb "github.com/Elvenson/xgboost-go"
	"github.com/Elvenson/xgboost-go/activation"
)

// MockImage creates a mock RGBA image for testing purposes.
func MockImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill the image with a solid color (e.g., white)
	drawColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, drawColor)
		}
	}
	return img
}

func TestConfigValidate(t *testing.T) {
	// Test case where camera name is empty
	cfg := &Config{
		CameraName: "",
	}
	path := "test_path"
	dependencies, err := cfg.Validate(path)
	test.That(t, dependencies, test.ShouldBeNil)
    test.That(t, err, test.ShouldBeNil)

	// Test case where detector name is empty
	cfg = &Config{
		CameraName:   "camera1",
		DetectorName: "",
	}
	path = "test_path"
	dependencies, err = cfg.Validate(path)
	test.That(t, dependencies, test.ShouldResemble, []string{"camera1"})
	test.That(t, err, test.ShouldBeNil)
}

func TestClassificationsFromCamera(t *testing.T) {
	pf := &prefilter{
		camName:      "configuredCamera",
		triggerFlag:  &atomic.Bool{},
		cancelContext: context.Background(),
	}
	ctx := context.Background()
	cameraName := "testCamera"

	// Test case where camera name does not match
	classifications, err := pf.ClassificationsFromCamera(ctx, cameraName, 1, nil)
	test.That(t, classifications, test.ShouldBeNil)
	test.That(t, err.Error(), test.ShouldEqual, "camera name given to method, testCamera is not the same as configured camera configuredCamera")

	// Test case where context is done
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()
	classifications, err = pf.ClassificationsFromCamera(cancelledCtx, "configuredCamera", 1, nil)
	test.That(t, classifications, test.ShouldBeNil)
	test.That(t, err.Error(), test.ShouldEqual, "module might be configuring: context canceled")

	// Test case where internal context is done
	cancelledInternalCtx, internalCancel := context.WithCancel(ctx)
	pf.cancelContext = cancelledInternalCtx
	internalCancel()
	classifications, err = pf.ClassificationsFromCamera(ctx, "configuredCamera", 1, nil)
	test.That(t, classifications, test.ShouldBeNil)
	test.That(t, err.Error(), test.ShouldEqual, "lost connection with background camera stream loop: context canceled")

	// Test case where trigger flag is not set
	pf.cancelContext = context.Background()
	classifications, err = pf.ClassificationsFromCamera(ctx, "configuredCamera", 1, nil)
	test.That(t, classifications, test.ShouldBeEmpty)
	test.That(t, err, test.ShouldBeNil)

	// Test case where trigger flag is set
	pf.triggerFlag.Store(true)
	classifications, err = pf.ClassificationsFromCamera(ctx, "configuredCamera", 1, nil)
	expectedClassifications := classification.Classifications{
		classification.NewClassification(1.0, "TRIGGER"),
	}
	test.That(t, classifications, test.ShouldResemble, expectedClassifications)
	test.That(t, err, test.ShouldBeNil)
}

func TestClassifications(t *testing.T) {
    // only tests that get_classifications works with a given image
    // context no longer needed for this function
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
    pf := &prefilter{
        triggerFlag:   &atomic.Bool{},
        cancelContext: context.Background(),
        rc: rc,
    }

    ctx := context.Background()
	f, err := os.Open("test_data/2288.jpg")
	test.That(t, err, test.ShouldBeNil)
	defer f.Close()
	img, _, err := image.Decode(f)
	test.That(t, err, test.ShouldBeNil)

    classifications, err := pf.Classifications(ctx, img, 1, nil)
    test.That(t, err, test.ShouldBeNil)
    test.That(t, classifications, test.ShouldNotBeNil)

    // Test case where context is canceled
    cancelledCtx, cancel := context.WithCancel(ctx)
    cancel()
    classifications, err = pf.Classifications(cancelledCtx, img, 1, nil)
    test.That(t, err, test.ShouldBeNil)
    test.That(t, classifications, test.ShouldNotBeNil)
}

func TestGetProperties(t *testing.T) {
    // Create a mock prefilter instance
    pf := &prefilter{
        properties: vision.Properties{
            ClassificationSupported: true,
            DetectionSupported:      false,
            ObjectPCDsSupported:     false,
        },
    }

    // Mock context and extra parameters
    ctx := context.Background()
    extra := map[string]interface{}{}

    // Call the GetProperties method
    properties, err := pf.GetProperties(ctx, extra)

    // Assert the returned properties
    test.That(t, properties, test.ShouldNotBeNil)
    test.That(t, *properties, test.ShouldResemble, pf.properties)

    // Assert no error returned
    test.That(t, err, test.ShouldBeNil)
}


func TestCaptureAllFromCamera(t *testing.T) {
    stubImage := image.NewRGBA(image.Rect(0, 0, 100, 100))
    imgInterface := image.Image(stubImage) // Convert *image.RGBA to image.Image interface
    imgPtr := unsafe.Pointer(&imgInterface)

    // Mock image dimensions
    mockWidth := 100
    mockHeight := 100

    // Create a mock image
    mockImg := image.NewRGBA(image.Rect(0, 0, mockWidth, mockHeight))

    pf := &prefilter{
        camName:       "configuredCamera",
        triggerFlag:   &atomic.Bool{},
        cancelContext: context.Background(),
    }

    ctx := context.Background()

    atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&pf.currImg)), unsafe.Pointer(&mockImg))

    // Test case where context is canceled
    cancelledCtx, cancel := context.WithCancel(ctx)
    cancel()
    capture, err := pf.CaptureAllFromCamera(cancelledCtx, "configuredCamera", viscapture.CaptureOptions{ReturnImage: true, ReturnClassifications: true}, nil)
    test.That(t, capture.Image, test.ShouldBeEmpty)
    test.That(t, capture.Classifications, test.ShouldBeEmpty)
    test.That(t, err.Error(), test.ShouldEqual, "context canceled")

    // Test case where internal context is canceled
    cancelledInternalCtx, internalCancel := context.WithCancel(ctx)
    pf.cancelContext = cancelledInternalCtx
    internalCancel()
    capture, err = pf.CaptureAllFromCamera(ctx, "configuredCamera", viscapture.CaptureOptions{ReturnImage: true, ReturnClassifications: true}, nil)
    test.That(t, capture.Image, test.ShouldBeEmpty)
    test.That(t, capture.Classifications, test.ShouldBeEmpty)
    test.That(t, err.Error(), test.ShouldEqual, "context canceled")

    // Test case where camera name does not match
    pf.cancelContext = context.Background()
    capture, err = pf.CaptureAllFromCamera(ctx, "incorrectCamera", viscapture.CaptureOptions{ReturnImage: true, ReturnClassifications: true}, nil)
    test.That(t, capture.Image, test.ShouldBeEmpty)
    test.That(t, capture.Classifications, test.ShouldBeEmpty)
    test.That(t, err.Error(), test.ShouldEqual, "Camera name \"incorrectCamera\" given to CaptureAllFromCamera is not the same as configured camera \"configuredCamera\"")

    // Test case where no image or classifications are requested
    capture, err = pf.CaptureAllFromCamera(ctx, "configuredCamera", viscapture.CaptureOptions{}, nil)
    test.That(t, capture.Image, test.ShouldBeEmpty)
    test.That(t, capture.Classifications, test.ShouldBeEmpty)
    test.That(t, err, test.ShouldBeNil)

    // Test case where only image is requested
    pf.triggerFlag.Store(true)
    atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&pf.currImg)), imgPtr)
    capture, err = pf.CaptureAllFromCamera(ctx, "configuredCamera", viscapture.CaptureOptions{ReturnImage: true}, nil)
    test.That(t, capture.Image, test.ShouldResemble, stubImage)
    test.That(t, capture.Classifications, test.ShouldBeEmpty)
    test.That(t, err, test.ShouldBeNil)

    // Test case where only classifications are requested
    capture, err = pf.CaptureAllFromCamera(ctx, "configuredCamera", viscapture.CaptureOptions{ReturnClassifications: true}, nil)
    expectedClassifications := viscapture.VisCapture{
        Classifications: classification.Classifications{
            classification.NewClassification(1.0, triggerClassName),
        },
    }
    test.That(t, capture.Image, test.ShouldBeNil)
    test.That(t, capture.Classifications, test.ShouldResemble, expectedClassifications.Classifications)
    test.That(t, err, test.ShouldBeNil)

    // Test case where both image and classifications are requested
    capture, err = pf.CaptureAllFromCamera(ctx, "configuredCamera", viscapture.CaptureOptions{ReturnImage: true, ReturnClassifications: true}, nil)
    test.That(t, capture.Image, test.ShouldResemble, stubImage)
    test.That(t, capture.Classifications, test.ShouldResemble, expectedClassifications.Classifications)
    test.That(t, err, test.ShouldBeNil)
}

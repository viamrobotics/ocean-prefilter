// Package oceanprefilter implements a classifier for interesting events on the open waters, as a Viam vision service
package oceanprefilter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"image"

	"github.com/pkg/errors"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
	vis "go.viam.com/rdk/vision"
	"go.viam.com/rdk/vision/classification"
	objdet "go.viam.com/rdk/vision/objectdetection"
	viamutils "go.viam.com/utils"
)

const (
	// ModelName is the name of the model
	ModelName = "ocean-prefilter"
	// DefaulMaxFrequency is how often the vision service will poll the camera for a new image
	DefaultMaxFrequency = 10.0
	triggerClassName    = "TRIGGER"
)

var (
	// Model is the resource
	Model            = resource.NewModel("viam", "vision", ModelName)
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(vision.API, Model, resource.Registration[vision.Service, *Config]{
		Constructor: newPrefilter,
	})
}

// Config contains names for necessary resources (camera and vision service)
type Config struct {
	CameraName   string             `json:"camera_name"`
	DetectorName string             `json:"detector_name"`
	ChosenLabels map[string]float64 `json:"chosen_labels"`
	MaxFrequency float64            `json:"max_frequency_hz"`
}

// Validate validates the config and returns implicit dependencies,
// this Validate checks if the camera and detector(optional) exist for the module's vision model.
func (cfg *Config) Validate(path string) ([]string, error) {
	// this makes them required for the model to successfully build
	if cfg.CameraName == "" {
		return nil, fmt.Errorf(`expected "camera_name" attribute for object tracker %q`, path)
	}
	if cfg.DetectorName == "" {
		return []string{cfg.CameraName}, nil
	}
	return []string{cfg.CameraName, cfg.DetectorName}, nil
}

// prefilter is the main struct for this module. It is a vision service classifier that will return a "TRIGGER" class
// if it sees something interesting
type prefilter struct {
	resource.Named
	logger                  logging.Logger
	cancelFunc              context.CancelFunc
	cancelContext           context.Context
	activeBackgroundWorkers sync.WaitGroup
	triggerFlag             *atomic.Bool // will be a shared variable
	camName                 string
}

// runConfig are the settings that will be fed to the background thread that will constantly be evaluating images for events
type runConfig struct {
	cam           camera.Camera
	camName       string
	detector      vision.Service
	chosenLabels  map[string]float64
	frequency     float64
	minConfidence float64
}

// newPrefilter creates the vision service classifier
func newPrefilter(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (vision.Service, error) {
	pf := &prefilter{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}
	pf.triggerFlag.Store(false)

	if err := pf.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}
	return pf, nil
}

// Reconfigure reconfigures prefilter with new settings from the config. It stops the old stream and starts a new one.
func (pf *prefilter) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// first check if there is a stream controled by the context already running that needs to be closed
	if pf.cancelFunc != nil {
		pf.cancelFunc()
		pf.activeBackgroundWorkers.Wait()
	}
	pf.triggerFlag.Store(false)
	cancelableCtx, cancel := context.WithCancel(context.Background())
	pf.cancelFunc = cancel
	pf.cancelContext = cancelableCtx
	// This takes the generic resource.Config passed down from the parent and converts it to the
	// model-specific (aka "native") Config structure defined, above making it easier to directly access attributes.
	prefilterConfig, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return errors.Errorf("Could not assert proper config for %s", ModelName)
	}

	// the run config will store the relevant variables from the prefilterConfig for running
	rc := runConfig{}
	// now load the relevant info into the runConfig
	if prefilterConfig.MaxFrequency < 0 {
		return errors.New("frequency(Hz) must be a non-negative number")
	}
	if prefilterConfig.MaxFrequency == 0 {
		rc.frequency = DefaultMaxFrequency
	} else {
		rc.frequency = prefilterConfig.MaxFrequency
	}

	rc.camName = prefilterConfig.CameraName
	pf.camName = prefilterConfig.CameraName
	rc.cam, err = camera.FromDependencies(deps, prefilterConfig.CameraName)
	if err != nil {
		return errors.Wrapf(err, "unable to get camera %v for ocean prefilter", prefilterConfig.CameraName)
	}
	if prefilterConfig.DetectorName != "" {
		rc.detector, err = vision.FromDependencies(deps, prefilterConfig.DetectorName)
		if err != nil {
			return errors.Wrapf(err, "unable to get camera %v for ocean prefilter", prefilterConfig.DetectorName)
		}
	}
	rc.chosenLabels = prefilterConfig.ChosenLabels // if you configred an optional detector, this determines the labels and confidences to use

	// now start the background thread
	pf.activeBackgroundWorkers.Add(1)
	viamutils.ManagedGo(func() {
		// if you get an error while running just keep trying forever
		for {
			runErr := run(pf.cancelContext, rc, pf.triggerFlag)
			if runErr != nil {
				pf.logger.Errorw("background camera stream exited with error", "error", runErr)
				continue // keep trying to run, forever
			}
			return
		}
	}, func() {
		pf.activeBackgroundWorkers.Done()
	})
	return nil
}

// run sets up a camera stream and then takes new pictures and processes them for anomalies
// at the desired frequency.
func run(ctx context.Context, rc runConfig, trigger *atomic.Bool) error {
	if rc.cam == nil {
		return errors.Errorf("underlying camera %q is nil, cannot start background stream", rc.camName)
	}
	stream, err := rc.cam.Stream(ctx, nil)
	if err != nil {
		return err
	}
	defer stream.Close(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			start := time.Now()
			img, release, err := stream.Next(ctx)
			if err != nil {
				trigger.Store(false)
				release()
				return err
			}
			isTriggered, err := theFilter(img) // bogus stand in function for now
			if isTriggered {
				trigger.Store(true)
			} else {
				trigger.Store(false)
			}
			release()

			took := time.Since(start)
			waitFor := time.Duration((1/rc.frequency)*float64(time.Second)) - took // only poll according to set freq
			if waitFor > time.Microsecond {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(waitFor):
				}
			}
		}
	}
}

// theFilter is a stupid fillter function just for testing purposes
func theFilter(img image.Image) (bool, error) {
	// if the minute is even, return true
	currentTime := time.Now()
	minute := currentTime.Minute()
	isEven := minute%2 == 0
	return isEven, nil
}

func (pf *prefilter) DetectionsFromCamera(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]objdet.Detection, error) {
	return nil, errUnimplemented
}

func (pf *prefilter) Detections(ctx context.Context, img image.Image, extra map[string]interface{}) ([]objdet.Detection, error) {
	return nil, errUnimplemented
}

func (pf *prefilter) ClassificationsFromCamera(
	ctx context.Context,
	cameraName string,
	n int,
	extra map[string]interface{},
) (classification.Classifications, error) {
	if cameraName != pf.camName {
		return nil, errors.Errorf("camera name given to method, %v is not the same as configured camera %v", cameraName, pf.camName)
	}
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "module might be configuring")
	case <-pf.cancelContext.Done():
		return nil, errors.Wrap(pf.cancelContext.Err(), "lost connection with background camera stream loop")
	default:
		cls := []classification.Classification{}
		if pf.triggerFlag.Load() {
			c := classification.NewClassification(1.0, triggerClassName)
			cls = append(cls, c)
		}
		return classification.Classifications(cls), nil
	}
}

func (pf *prefilter) Classifications(ctx context.Context, img image.Image,
	n int, extra map[string]interface{},
) (classification.Classifications, error) {
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "module might be configuring")
	case <-pf.cancelContext.Done():
		return nil, errors.Wrap(pf.cancelContext.Err(), "lost connection with background camera stream loop")
	default:
		cls := []classification.Classification{}
		if pf.triggerFlag.Load() {
			c := classification.NewClassification(1.0, triggerClassName)
			cls = append(cls, c)
		}
		return classification.Classifications(cls), nil
	}
}

func (pf *prefilter) GetObjectPointClouds(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]*vis.Object, error) {
	return nil, errUnimplemented
}

func (pf *prefilter) Close(ctx context.Context) error {
	pf.cancelFunc()
	pf.activeBackgroundWorkers.Wait()
	return nil
}

// DoCommand will return the slowest, fastest, and average time of the tracking module
func (pf *prefilter) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

# ocean-prefilter

Viam provides an `ocean-prefilter` model of the [vision service](https://docs.viam.com//services/vision) to find objects of interest in large bodies of water.

It works by finding the horizon, cropping the image to only see the water, and then divind the water into patches. These patches are turned into histograms, which are interpreted as probability distributions for their color/intensity values. A Kolmogorov-Smirnov test is then done on the before and after image, to determine if they are "close enough" to being the same scene, or if something new has entered the scene. 

Strong motion of the waves or bobbing up-and-down of the boat can trigger the pre-filter.

## Config

You can download it from the viam Registry at [ocean-prefilter](https://app.viam.com/module/viam-labs/ocean-prefilter)

```
    {
      "name": "vision-ocean",
      "namespace": "rdk",
      "type": "vision",
      "model": "viam-labs:vision:ocean-prefilter",
      "attributes": {
        "camera_name": "my_cam",
        "threshold": 0.25,
        "max_frequency_hz": 5
      }
    }
```

- `camera_name`: this is a necessary parameter that links the prefilter to a specific camera. It calls the camera stream in a loop in the background in order to always be monitoring the scene for triggers. As such, calls to `get_classifications` simply returns what the result from `get_classifications_from_camera` would return, as the vision service is strongly linked to this camera.
- `threshold` : default is 0.25. this is a number between 0 and 1. It determines how sensitive the trigger for the pre-filter is. The prefilter will pick up on both strong motion of the boat/waves, as well as objects like other boats, buoys, and anything that looks different from the standard pattern of the water.
- `max_frequency_hz`: default is 10. Determines how often the vision service should poll the background camera stream for changes in the scene. It is not recommended to set this lower than 1, unless the scene is changing very slowly. 

## statically build in Linux

```
wget -O opencv.zip https://github.com/opencv/opencv/archive/4.9.0.zip
unzip opencv.zip
mkdir -p build && cd build

cmake -DBUILD_SHARED_LIBS=OFF -DOPENCV_GENERATE_PKGCONFIG=ON ../opencv-4.9.0
sudo make
sudo make install

CGO_ENABLED=1 CGO_CFLAGS="-I/usr/local/include -I/usr/local/include/opencv4" CGO_LDFLAGS="-I/usr/local/include/opencv4 -L/usr/local/lib" go build -tags static -ldflags="-extldflags=-static" main.go
```

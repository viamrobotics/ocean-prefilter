# ocean-prefilter

Viam provides an `ocean-prefilter` model of the [vision service](https://docs.viam.com//services/vision) to find objects of interest in large bodies of water.

It works by finding the horizon, cropping the image to only see the water, and then divides the water into patches. These patches are turned into histograms, which are interpreted as probability distributions for their color/intensity values. A Kolmogorov-Smirnov test is then done on the before and after image, to determine if they are "close enough" to being the same scene, or if something new has entered the scene. 

Strong motion of the waves or bobbing up-and-down of the boat can trigger the pre-filter.

This vision service only returns one label classification called _TRIGGER_ with a confidence of 1.0.

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

- _`camera_name`_: this is a necessary parameter that links the prefilter to a specific camera. It calls the camera stream in a loop in the background in order to always be monitoring the scene for triggers. As such, calls to `get_classifications` simply returns what the result from `get_classifications_from_camera` would return, as the vision service is strongly linked to this camera.
- _`threshold`_ : default is 0.25. this is a number between 0 and 1. It determines how sensitive the trigger for the pre-filter is. The prefilter will pick up on both strong motion of the boat/waves, as well as objects like other boats, buoys, and anything that looks different from the standard pattern of the water.
- _`max_frequency_hz`_: default is 10. Determines how often the vision service should poll the background camera stream for changes in the scene. It is not recommended to set this lower than 1, unless the scene is changing very slowly. 

## Download dependencies for the module

openCV is a requirement for this module, if you want to compile this module yourself
log into your raspberry pi and install the following necessary libraries 
```
sudo apt-get install libjpeg-dev

git clone https://github.com/hybridgroup/gocv.git
cd gocv
sudo make install_raspi # or just sudo make install

git clone https://github.com/viam-labs/ocean-prefilter
make ocean-prefilter
```


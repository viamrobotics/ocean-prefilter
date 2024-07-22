# [`ocean-prefilter` module](https://app.viam.com/module/viam-labs/ocean-prefilter)

This module implements the [`rdk:service:vision` API](https://docs.viam.com/ml/vision/#api) in an `ocean-prefilter` model for your machine to find objects of interest in large bodies of water.

When you configure a machine with this module, the module:
- Locates the horizon, crops the image to only include the water, and then divides the water into patches.
- Performs feature extraction on the resulting patches, average pooling the the patches with a window size of (10, 2) and taking the RGB mean of each sub-patch
- Classifies the frame using XGBoost - this will trigger if any patch in the given image is found interesting

Strong motion of the waves or bobbing up-and-down of the boat can trigger the pre-filter.
This vision service only returns one label classification called `TRIGGER` with a confidence of `1.0`.

## Requirements

To compile this module yourself, follow these steps on your Raspberry Pi:

1. Download and install [`openCV`](https://opencv.org/)
2. Install the following necessary libraries:
```
sudo apt-get install libjpeg-dev

git clone https://github.com/hybridgroup/gocv.git
cd gocv
sudo make install_raspi # or just sudo make install

git clone https://github.com/viamrobotics/ocean-prefilter
make ocean-prefilter
```

## Configure your `ocean-prefilter` vision service

> [!NOTE]
> Before configuring your vision service, you must [create a machine](https://docs.viam.com/fleet/machines/#add-a-new-machine).

Navigate to the **CONFIGURE** tab of your machine's page in [the Viam app](https://app.viam.com).
Click the **+** icon next to your machine part in the left-hand menu and select **Service**.
Select the `vision` type, then search for and select the `ocean-prefilter` model.
Click **Add module**, then enter a name or use the suggested name for your service and click **Create**.

Click the **{}** (Switch to Advanced) button in the top right of the service panel to edit the service's attributes directly with JSON.
Copy and paste the following attribute template into your vision service's attributes field:

```json
  {
      "camera_name": "my_cam",
      "threshold": 0.25,
      "max_frequency_hz": 5,
      "excluded_region": [xmin, ymin, xmax, ymax]
  }
```

> [!NOTE]
> For more information, see [Configure a Machine](https://docs.viam.com/build/configure/).

### Attributes

| Name  | Type  | Inclusion | Description | Value |
|-------|-------|-----------|-------------| ------|
| `camera_name` | string | Optional | Links the pre-filter to a specific camera and continuously monitors the camera stream for changes or triggers in the background. | The name of your camera component. | If the camera name is not provided, you can input your own image from the CLI |
| `threshold`  | int | Optional | Determines the sensitivity of the pre-filter trigger. This enables the pre-filter to detect significant motion such as boat or wave movements, and identifies objects like other boats, buoys, or any deviations from typical water patterns. | 0 to 1<br/> Default: `0.25` |
| `max_frequency_hz`| int | Optional  | Determines the frequency that the vision service monitors the background camera stream for changes. If your scene changes very slowly set this below 1. | 1 to 10<br/> Default: `10` |
| `excluded_region` | object   | Optional  | Specifies areas within the cameras view to ignore. This is useful for excluding static parts of the camera stream, like parts of the boat. | A list of coordinates in frame. |

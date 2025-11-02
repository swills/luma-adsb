# luma-adsb

Designed to run on [ADSB.im](https://adsb.im) and [GeeekPi](https://mygeeekpi.com/) [Mini Tower Kit for Raspberry Pi 5](https://www.amazon.com/dp/B0CQYTN94R)

Displays the ADSB data on the SSD1306 display

![Photo of the software in action](images/luma-adsb-photo.png "Luma ADSB in action")

# Usage

Set `dtparam=i2c_arm=on,i2c_arm_baudrate=400000` in `/boot/firmware/config.txt` and reboot.

Set the following environment variables:

`LUMAADSB_HOST`: hostname or IP of adsb.im host

`LUMAADSB_LAT`: Latitude of the host

`LUMAADSB_LON`: Longitude of the host

`LUMAADSB_MAX_ALT`: Max altitude to consider "close"

`LUMAADSB_MAX_DISTANCE`: Max distance to consider "close"

`LUMAADSB_MIN_ALT`: Min altitude to consider "close"

The "close" parameters can also be customized per category by suffixing the category, for example:

`LUMAADSB_MAX_DISTANCE_CATEGORY_A1`: Max distance for A1

`LUMAADSB_MAX_ALT_CATEGORY_A3`: Max altitude for A3

Then run `./luma-adsb`

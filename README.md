# luma-adsb

Designed to run on [ADSB.im](https://adsb.im) and [GeeekPi](https://mygeeekpi.com/) [Mini Tower Kit for Raspberry Pi 5](https://www.amazon.com/dp/B0CQYTN94R)

Displays the ADSB data on the SSD1306 display

![Photo of the software in action](images/luma-adsb-photo.png "Luma ADSB in action")

# Usage

Set `dtparam=i2c_arm=on,i2c_arm_baudrate=400000` in `/boot/firmware/config.txt` and reboot.

Set the following environment variables:

`ADSBFEED_HOST`: hostname or IP of adsb.im host

`ADSBFEED_LAT`: Latitude of the host

`ADSBFEED_LON`: Longitude of the host

Then run `./luma-adsb`

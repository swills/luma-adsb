# luma-adsb

[![Build and Release](https://github.com/swills/luma-adsb/actions/workflows/build.yml/badge.svg)](https://github.com/swills/luma-adsb/actions/workflows/build.yml)
[![Lint](https://github.com/swills/luma-adsb/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/swills/luma-adsb/actions/workflows/golangci-lint.yml)

Designed to run on [ADSB.im](https://adsb.im) and [GeeekPi](https://mygeeekpi.com/) [Mini Tower Kit for Raspberry Pi 5](https://www.amazon.com/dp/B0CQYTN94R)

Displays the ADSB data on the SSD1306 display

![Photo of the software in action](images/luma-adsb-photo.png "Luma ADSB in action")

# Usage

Set `dtparam=i2c_arm=on,i2c_arm_baudrate=400000` in `/boot/firmware/config.txt` and reboot.

Set the following environment variables:

`LUMAADSB_HOST`: hostname or IP of adsb.im host

`LUMAADSB_LAT`: Latitude of the host

`LUMAADSB_LON`: Longitude of the host

`LUMAADSB_ALT`: Altitude of the host

Then run `./luma-adsb`

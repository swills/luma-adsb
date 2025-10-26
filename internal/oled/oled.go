package oled

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	goi2coled "github.com/waxdred/go-i2c-oled"
	"github.com/waxdred/go-i2c-oled/ssd1306"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func InitDisplay() *goi2coled.I2c {
	// Initialize the OLED display with the provided parameters
	oled, err := goi2coled.NewI2c(ssd1306.SSD1306_SWITCHCAPVCC, 64, 128, 0x3C, 1)
	if err != nil {
		panic(err)
	}

	black := color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}

	// Set the entire OLED image to black
	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: black}, image.Point{}, draw.Src)

	return oled
}

func ClearDisplay(oled *goi2coled.I2c) {
	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
	oled.Clear()
	oled.Draw()
	_ = oled.Display()
}

func UpdateDisplayLines(dispLines []string, oled *goi2coled.I2c) {
	fontHeight := basicfont.Face7x13.Metrics().Height

	var err error

	var drawer *font.Drawer

	point := fixed.Point26_6{
		X: fixed.Int26_6(0 * 64),
		Y: fixed.Int26_6(15 * 64),
	} // x = 0, y = 15

	drawer = &font.Drawer{
		Dst:  oled.Img,
		Src:  &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	draw.Draw(oled.Img, oled.Img.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)

	for i, line := range dispLines {
		if line == "" || i > 5 {
			break
		}

		drawer.DrawString(line)
		drawer.Dot.X = fixed.Int26_6(0)
		drawer.Dot.Y += fontHeight
	}

	oled.Draw()

	err = oled.Display()
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}

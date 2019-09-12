package main

import (
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func convert(f io.ReadCloser, chars []string, subWidth, subHeight int, imageSwitch bool, bgColor, penColor color.RGBA) (string, *image.NRGBA, error) {
	defer f.Close()
	var charsLength int = len(chars)
	if charsLength == 0 {
		return "", nil, fmt.Errorf("No chars provided")
	}
	if subWidth == 0 || subHeight == 0 {
		return "", nil, fmt.Errorf("subWidth and subHeight params is required")
	}
	m, _, err := image.Decode(f)
	if err != nil {
		return "", nil, err
	}
	imageWidth, imageHeight := m.Bounds().Max.X, m.Bounds().Max.Y
	var img *image.NRGBA
	if imageSwitch {
		img = initImage(imageWidth, imageHeight, bgColor)
	}
	piecesX, piecesY := imageWidth/subWidth, imageHeight/subHeight
	var buff bytes.Buffer
	for y := 0; y < piecesY; y++ {
		offsetY := y * subHeight
		for x := 0; x < piecesX; x++ {
			offsetX := x * subWidth
			averageBrightness := calculateAverageBrightness(m, image.Rect(offsetX, offsetY, offsetX+subWidth, offsetY+subHeight))
			char := getCharByBrightness(chars, averageBrightness)
			buff.WriteString(char)
			if img != nil {
				addCharToImage(img, char, x*subWidth, y*subHeight, penColor)
			}
		}
		buff.WriteString("\n")
	}
	return buff.String(), img, nil
}

func initImage(width, height int, bgColor color.RGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, bgColor)
		}
	}
	return img
}

func calculateAverageBrightness(img image.Image, rect image.Rectangle) float64 {
	var averageBrightness float64
	width, height := rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y
	var brightness float64
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			brightness = float64(r>>8+g>>8+b>>8) / 3
			averageBrightness += brightness
		}
	}
	averageBrightness /= float64(width * height)
	return averageBrightness
}

func getCharByBrightness(chars []string, brightness float64) string {
	index := int(brightness*float64(len(chars))) >> 8
	if index == len(chars) {
		index--
	}
	return chars[len(chars)-index-1]
}

func addCharToImage(img *image.NRGBA, char string, x, y int, penColor color.RGBA) {
	face := basicfont.Face7x13
	point := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(penColor),
		Face: face,
		Dot:  point,
	}
	d.DrawString(char)
}

func getDataFromUrl(imageUrl string) (io.ReadCloser, error) {
	var respImgData *http.Response
	respImgData, err := http.Get(imageUrl)
	if err != nil {
		return nil, err
	}
	return respImgData.Body, nil
}

var colors map[string]color.RGBA = map[string]color.RGBA{"black": {0, 0, 0, 255},
	"gray":  {140, 140, 140, 255},
	"red":   {255, 0, 0, 255},
	"green": {0, 128, 0, 255},
	"blue":  {0, 0, 255, 255}}

func main() {
	server := flag.Bool("server", false, "Set true to enable web ui server")
	bind := flag.String("ip", "127.0.0.1:8080", "The address to bind.")
	imagePath := flag.String("image-path", "", "The path of the picture file(jpg/png).")
	imageUrl := flag.String("image-url", "", "The url of the picture file(jpg/png).")
	characters := flag.String("characters", "M80V1i:*|, ", "The chars in the value will be used to generate ascii art.")
	subWidth := flag.Int("sub-width", 10, "The width of the piece of small rectangle")
	subHeight := flag.Int("sub-height", 10, "The height of the piece of small rectangle")
	imageOut := flag.Bool("image-out", false, "If set to true, will also output the ascii art to an image file")
	imageOutPath := flag.String("image-out-path", "", "If imageOut is true, add this option to print the picture to file.")
	bgColorType := flag.String("bg", "black", "The background of the ascii art image file.(Only useful when image-out is true, red|gray|green|blue|black are available).")
	penColorType := flag.String("color", "gray", "The color of the ascii font in image file.(Only useful when image-out is true, red|gray|green|blue|black are available).")
	flag.Parse()

	if  os.Getenv("PORT") != "" {
		temp := fmt.Sprintf(":%s", os.Getenv("PORT"))
		bind = &temp
	}

	if *server {
		startServer(*bind)
	}

	var file io.ReadCloser
	var err error
	if *imagePath != "" {
		file, err = os.Open(*imagePath)
	} else {
		file, err = getDataFromUrl(*imageUrl)
	}
	if err != nil {
		log.Fatal(err)
	}
	//chars := []string{"M", "8", "0", "V", "1", "i", ":", "*", "|", ".", " "}
	chars := strings.Split(*characters, "")
	bgColor, penColor := colors[*bgColorType], colors[*penColorType]
	result, img, err := convert(file, chars, *subWidth, *subHeight, *imageOut, bgColor, penColor)
	if err != nil {
		log.Fatal(err)
	}
	if img != nil {
		imgFile, _ := os.Create(*imageOutPath)
		defer imgFile.Close()
		//img = imaging.Fit(img, 800, 600, imaging.Lanczos)
		png.Encode(imgFile, img)
	}
	fmt.Print(result)
}

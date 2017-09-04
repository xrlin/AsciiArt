package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func index(w http.ResponseWriter, r *http.Request) {
	data, _ := Asset("index.html")
	w.Write(data)
}

type config struct {
	ImageLink string `json:"image_link"`
}

type serverError struct {
	Error string `json:"error"`
}

type responseData struct {
	Ascii string `json:"ascii"`
	Image string `json:"image"`
}

func ascii(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			errorMessage, _ := json.Marshal(serverError{Error: err.(error).Error()})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(errorMessage)
		}
	}()
	r.ParseMultipartForm(0)
	image_link := r.FormValue("image_link")

	imageFile, _, err := r.FormFile("image_file")
	defer func() {
		if imageFile != nil {
			imageFile.Close()
		}
	}()

	characters := strings.Split(r.FormValue("characters"), "")
	subWidth, err := strconv.Atoi(r.FormValue("sub_width"))
	if err != nil {
		panic(err)
	}
	subHeight, err := strconv.Atoi(r.FormValue("sub_height"))
	if err != nil {
		panic(err)
	}
	bgColor := r.FormValue("bg")
	penColor := r.FormValue("pen")
	var imgData io.ReadCloser
	if err == nil && imageFile != nil {
		imgData = io.ReadCloser(imageFile)
	} else {
		var respImgData *http.Response
		fmt.Println(image_link)
		respImgData, err = http.Get(image_link)
		if err != nil {
			panic(err)
		}
		imgData = respImgData.Body
	}
	if err != nil {
		panic(err)
	}
	result, img, err := convert(imgData, characters, subWidth, subHeight, true, colors[bgColor], colors[penColor])
	if err != nil {
		errorMessage, _ := json.Marshal(serverError{Error: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorMessage)
		return
	}
	imgBase64, err := imageToBase64(img)
	if err != nil {
		panic(err)
	}
	resp, _ := json.Marshal(responseData{Ascii: result, Image: imgBase64})
	w.Write(resp)
}

func imageToBase64(img *image.NRGBA) (string, error) {
	buffer := bytes.NewBuffer(nil)
	err := png.Encode(buffer, img)
	if err != nil {
		return "", err
	}
	result := base64.StdEncoding.EncodeToString(buffer.Bytes())
	result = "data:image/png;base64, " + result
	return result, nil
}

func startServer(addr string) {
	http.HandleFunc("/", index)
	http.HandleFunc("/ascii", ascii)
	http.ListenAndServe(addr, nil)
}

package main

import (
	"bytes"
	"errors"
	"flag"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unsafe"

	"github.com/nfnt/resize"
)

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L. -lm

#include <stdio.h>
#define NANOSVG_IMPLEMENTATION
#include "nanosvg.h"
#define NANOSVGRAST_IMPLEMENTATION
#include "nanosvgrast.h"
*/
import "C"

const defaultMaxWidth = 200
const defaultMaxHeight = 200
const defaultInterp = "NearestNeighbor"

const dpi = 96

var httpAddr = flag.String("http", ":5000", "HTTP service address (e.g., ':5000')")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			httpErr(w, r, http.StatusMethodNotAllowed, "Hint: use POST method", nil)
			return
		}

		file, header, err := r.FormFile("file")
		defer file.Close()

		width := uint(parseInt(r, "w", defaultMaxWidth))
		height := uint(parseInt(r, "h", defaultMaxHeight))

		if err != nil {
			httpErr(w, r, http.StatusBadRequest, "Err: ", err)
			return
		}

		var m image.Image
		switch path.Ext(header.Filename) {
		case ".webm":
			var buf bytes.Buffer
			err = ffmpeg(file, &buf)
			if err != nil {
				break
			}
			m, _, err = image.Decode(&buf)
		case ".png":
			m, err = png.Decode(file)
		case ".gif":
			m, err = gif.Decode(file)
		case ".jpg", ".jpeg":
			m, err = jpeg.Decode(file)
		case ".svg":
			m, err = rasterizeSVG(file)
		default:
			httpErr(w, r, http.StatusBadRequest, "Bad file type: "+header.Filename, nil)
			return
		}
		if err != nil {
			httpErr(w, r, http.StatusInternalServerError, "Convert Err: ", err)
			return
		}

		interp := r.FormValue("interp")
		if interp == "" {
			interp = defaultInterp
		}

		srcSize := m.Bounds().Size()
		w.Header().Add("SrcImage-Width", strconv.Itoa(srcSize.X))
		w.Header().Add("SrcImage-Height", strconv.Itoa(srcSize.Y))

		switch strings.ToLower(interp) {
		case "nearestneighbor": // XXX default
			m = resize.Thumbnail(width, height, m, resize.NearestNeighbor)
		case "bilinear":
			m = resize.Thumbnail(width, height, m, resize.Bilinear)
		case "bicubic":
			m = resize.Thumbnail(width, height, m, resize.Bicubic)
		case "mitchellnetravali":
			m = resize.Thumbnail(width, height, m, resize.MitchellNetravali)
		case "lanczos2":
			m = resize.Thumbnail(width, height, m, resize.Lanczos2)
		case "lanczos3":
			m = resize.Thumbnail(width, height, m, resize.Lanczos3)
		}

		dstSize := m.Bounds().Size()
		w.Header().Add("DstImage-Width", strconv.Itoa(dstSize.X))
		w.Header().Add("DstImage-Height", strconv.Itoa(dstSize.Y))

		quality := parseInt(r, "jpeg", 90)
		if quality < 0 || quality > 100 {
			err = png.Encode(w, m)
		} else {
			err = jpeg.Encode(w, m, &jpeg.Options{Quality: quality})
		}
		if err != nil {
			httpErr(w, r, http.StatusInternalServerError, "Err: ", err)
		}
	})
	http.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add("Accept", "text/html")
		w.Write([]byte(form))
	})

	log.Println("start thumbnail service at", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

func httpErr(w http.ResponseWriter, r *http.Request, code int, msg string, err error) {
	status := http.StatusText(code)
	http.Error(w, status, code)
	w.Write([]byte("\n\n"))
	w.Write([]byte(msg))
	if err != nil {
		w.Write([]byte(err.Error()))
	}
	log.Println(r.Method, code, r.RequestURI, status, err)
}

func parseInt(r *http.Request, name string, def int) int {
	n, err := strconv.ParseUint(r.FormValue("w"), 10, 32)
	if err != nil {
		return def
	}
	return int(n)
}

var failOpenSVG = errors.New("Could not open SVG image.")
var failRasterSVG = errors.New("Could not init rasterizer.")

func rasterizeSVG(file io.Reader) (m *image.RGBA, err error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	var svg *C.NSVGimage
	svg = C.nsvgParse(C.CString(string(data)), C.CString("px"), dpi)
	defer C.nsvgDelete(svg)
	if svg == nil {
		err = failOpenSVG
		return
	}

	w := C.int(svg.width)
	h := C.int(svg.height)

	var rast *C.NSVGrasterizer
	rast = C.nsvgCreateRasterizer()
	defer C.nsvgDeleteRasterizer(rast)
	if rast == nil {
		err = failRasterSVG
		return
	}

	m = image.NewRGBA(image.Rect(0, 0, int(w), int(h)))

	C.nsvgRasterize(
		rast, svg, 0, 0, 1,
		(*C.uchar)(unsafe.Pointer(&m.Pix[0])),
		w, h, w*4)

	return
}

func ffmpeg(r io.Reader, w io.Writer) error {
	cmd := exec.Command("ffmpeg",
		"-f", "webm", "-i", "pipe:0",
		"-f", "image2pipe", "-c", "png", "-vframes", "1", "pipe:1")
	cmd.Stdin = r
	cmd.Stdout = w
	var buf bytes.Buffer
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil && buf.Len() != 0 {
		err = errors.New(buf.String())
	}
	return err
}

const form = `
<!DOCTYPE html>
<html>
<body>
<form action="/" method="post" enctype="multipart/form-data">
	file: <input type="file" name="file">
	<button type="submit">Submit</button>
	<br>

	w/h (size):
	<input type="text" name="w" value="200">
	<input type="text" name="h" value="200">
	<br>
	jpeg: <input type="text" name="jpeg" value="90">
	<br>

	interp: <select name="interp">
		<option value="">Default</option>
		<option value="None">No resize</option>
		<option value="NearestNeighbor">NearestNeighbor</option>
		<option value="Bilinear">Bilinear</option>
		<option value="Bicubic">Bicubic</option>
		<option value="MitchellNetravali">MitchellNetravali</option>
		<option value="Lanczos2">Lanczos2</option>
		<option value="Lanczos3">Lanczos3</option>
	</select>

	<br>
	<br>
	Just POST with
	[file (required), w(=200), h(=200), interp(=NearestNeighbor), jpeg(=90)]
	options.
	<br>
	interp option case-insensitive
	<br>
	if jpeg not valid, output is png-file.

</form>
</body>
</html>
`

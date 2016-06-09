package main

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/nfnt/resize"
)

const defaultMaxWidth = 200
const defaultMaxHeight = 200
const defaultInterp = "NearestNeighbor"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			httpErr(w, r, http.StatusMethodNotAllowed, "Hint: use POST method", nil)
			return
		}
		file, header, err := r.FormFile("file")
		defer file.Close()

		width := parseUint(r, "w", defaultMaxWidth)
		height := parseUint(r, "h", defaultMaxHeight)

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
				httpErr(w, r, http.StatusInternalServerError, "ffmpeg Err: ", err)
				return
			}
			m, _, err = image.Decode(&buf)
			if err != nil {
				httpErr(w, r, http.StatusInternalServerError, "Err: ", err)
				return
			}
		case ".png":
			m, err = png.Decode(file)
			if err != nil {
				httpErr(w, r, http.StatusInternalServerError, "Err: ", err)
				return
			}
		case ".gif":
			m, err = gif.Decode(file)
			if err != nil {
				httpErr(w, r, http.StatusInternalServerError, "Err: ", err)
				return
			}
		case ".jpg", ".jpeg":
			m, err = jpeg.Decode(file)
			if err != nil {
				httpErr(w, r, http.StatusInternalServerError, "Err: ", err)
				return
			}
		default:
			httpErr(w, r, http.StatusBadRequest, "Bad file type: "+header.Filename, nil)
			return
		}

		interp := r.FormValue("interp")
		if interp == "" {
			interp = defaultInterp
		}

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

		err = png.Encode(w, m)
		if err != nil {
			httpErr(w, r, http.StatusInternalServerError, "Err: ", err)
			return
		}
	})
	http.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add("Accept", "text/html")
		w.Write([]byte(form))
	})
	log.Fatal(http.ListenAndServe(":5000", nil))
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

func parseUint(r *http.Request, name string, def uint) uint {
	n, err := strconv.ParseUint(r.FormValue("w"), 10, 32)
	if err != nil {
		return def
	}
	return uint(n)
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
	Just POST with [file (required), w(=200), h(=200), interp(=NearestNeighbor)] options.
	<br>
	interp option case-insensitive

</form>
</body>
</html>
`

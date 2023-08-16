package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestRandomString(t *testing.T) {
	var tools Tools
	s := tools.RandomString(10)
	if len(s) != 10 {
		t.Error("RandomString should return a string of length 10")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{"allowed no raname", []string{"image/jpeg", "image/png"}, false, false},
	{"allowed raname", []string{"image/jpeg", "image/png"}, true, false},
	{"not allowed", []string{"image/jpeg"}, true, true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer writer.Close()

			// create the form data field "file"
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			fileName := fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}
			// clean up
			_ = os.Remove(fileName)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: expected no error but received: %s", e.name, err.Error())
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// create the form data field "file"
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	fileName := fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean up
	_ = os.Remove(fileName)
}

func TestTools_CreateDirIfNotExists(t *testing.T) {
	var testTools Tools
	err := testTools.CreateDirIfNotExists("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExists("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	// clean up
	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{"valid string", "now is the time", "now-is-the-time", false},
	{"empty string", "", "", true},
	{"complex string", "Now is the time for all GOOD men! + fish & such &^123", "now-is-the-time-for-all-good-men-fish-such-123", false},
	{name: "japanese string", s: "こんにちは世界", expected: "", errorExpected: true},
	{name: "japanese string and roman characters", s: "hello world こんにちは世界", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugTests {
		slug, err := testTools.Slugify(e.s)

		if err != nil && !e.errorExpected {
			t.Errorf("%s: did not expect an error but got one: %s", e.name, err)
		}
		if err == nil && e.errorExpected {
			t.Errorf("%s: expected an error but did not get one", e.name)
		}

		if !e.errorExpected && slug != e.expected {
			t.Log(slug)
			t.Log(e.expected)
			t.Errorf("%s: expected %s but received %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, req, "./testdata", "img.png", "new_img.png")
	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "534283" {
		t.Errorf("wrong content lenght of %s", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"new_img.png\"" {
		t.Errorf("wrong content disposition: %s", res.Header["Content-Disposition"][0])
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var readJSONTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{"valid json", `{"foo": "bar"}`, false, 1024, false},
	{"badly formatted json", `{"foo":}`, true, 1024, false},
	{"incorrect type", `{"foo": 1}`, true, 1024, false},
	{"two json files", `{"foo":"1"}{"alpha":"beta"}`, true, 1024, false},
	{"empty body", ``, true, 1024, false},
	{"syntax error in json", `{"foo": 1"}`, true, 1024, false},
	{"unknown field in json", `{"BAZ":"1"}`, true, 1024, false},
	{"missing field name", `{jack:"1"}`, true, 1024, true},
	{"file too large", `{"foo":"bar"}`, true, 5, true},
	{"not json", `Hello, World!`, true, 1024, true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools
	for _, e := range readJSONTests {
		testTools.MaxJSONSize = e.maxSize
		testTools.AllowUnknownFields = e.allowUnknown

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error: ", err)
		}
		rr := httptest.NewRecorder()

		err = testTools.ReadJSON(rr, req, &decodedJSON)
		if e.errorExpected && err == nil {
			t.Errorf("%s: expected an error but did not get one", e.name)
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: did not expect an error but got one: %s", e.name, err.Error())
		}
		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}
	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write json, %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON", err)
	}

	if !payload.Error {
		t.Error("expected payload.Error to be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status code %d but received %d", http.StatusServiceUnavailable, rr.Code)
	}
}

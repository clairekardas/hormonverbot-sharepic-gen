// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This file contains the web handler to parse the user input, start creating
// the sharepic and sending it back.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// sharepicResult is the internal data type for a SharePic generation request.
type sharepicResult struct {
	Error string
	Jpeg  []byte
}

// Status code for HTTP headers.
func (result sharepicResult) Status() (code int) {
	code = http.StatusOK
	if result.Error != "" {
		code = http.StatusInternalServerError
	}
	return
}

// detectMime is a specialized http.DetectContentType function with additional
// support for HEIF and HEIC.
func detectMime(imgData []byte) (mime string) {
	defer func() {
		if mime == "" {
			mime = http.DetectContentType(imgData)
		}
	}()

	// https://github.com/strukturag/libheif/blob/v1.13.0/libheif/heif.cc#L358
	heifMagic := map[string]string{
		"ftypheic": "image/heic",
		"ftypheix": "image/heic",
		"ftypheim": "image/heic",
		"ftypheis": "image/heic",

		"ftypmif1": "image/heif",
	}

	if len(imgData) < 12 {
		return
	}
	if heifMime, ok := heifMagic[string(imgData[4:12])]; ok {
		mime = heifMime
	}
	return
}

// extractImageFromRequest returns the POSTed image.
func extractImageFromRequest(r *http.Request) (data []byte, mime string, err error) {
	img, imgHeader, err := r.FormFile("img")
	if err != nil {
		return nil, "", fmt.Errorf("cannot fetch `img` form, %v", err)
	}

	defer img.Close()

	if size := imgHeader.Size; size > maxImageSize {
		return nil, "", fmt.Errorf("`img`'s size %d is greater maximum size %d", size, maxImageSize)
	}

	imgData, err := io.ReadAll(img)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read `img`, %v", err)
	}

	mimeType := detectMime(imgData)
	if _, ok := allowedMimeTypes[mimeType]; !ok {
		return nil, "", fmt.Errorf("unsupported MIME type %q", mimeType)
	}

	return imgData, mimeType, nil
}

// extractStringFromRequest returns the POSTed string for key, or falls back to
// a default if the key either does not exist or the value is faulty.
func extractStringFromRequest(r *http.Request, key, fallback string) string {
	rawInput := r.FormValue(key)
	if rawInput == "" {
		return fallback
	}

	rawCleaned := strings.TrimSpace(rawInput)
	rawCleaned = regexp.MustCompile(`\s+`).ReplaceAllString(rawCleaned, " ")
	return rawCleaned
}

// sharepicRawResponse writes back the result either as a JPEG or text.
func sharepicRawResponse(result sharepicResult, w http.ResponseWriter, _ *http.Request) {
	if result.Error != "" {
		http.Error(w, result.Error, result.Status())
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(result.Status())

	if _, err := w.Write(result.Jpeg); err != nil {
		log.Printf("cannot write sharepic, %v", err)
	}
}

// sharepicJSONResponse writes back the result as a JSON object.
func sharepicJSONResponse(result sharepicResult, w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(result.Status())

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(result); err != nil {
		log.Printf("cannot encode result struct to JSON, %v", err)
	}
}

// sharepicHandler creates sharepics based on the POSTed data.
func sharepicHandler(w http.ResponseWriter, r *http.Request) {
	var (
		result         = sharepicResult{}
		startTime      = time.Now()
		responseFormat = "raw"
	)

	defer func() {
		stopTime := time.Now()

		log.Printf("request took %v to complete", stopTime.Sub(startTime))
		if result.Error != "" {
			log.Printf("request finished with an error, %v", result.Error)
		}
	}()

	defer func() {
		switch responseFormat {
		case "json":
			sharepicJSONResponse(result, w, r)

		case "raw":
			fallthrough
		default:
			sharepicRawResponse(result, w, r)
		}
	}()

	if method := r.Method; method != "POST" {
		result.Error = "HTTP POST only"
		return
	}

	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		result.Error = fmt.Sprintf("cannot parse multipart form, %v", err)
		return
	}

	responseFormat = extractStringFromRequest(r, "responseFormat", "raw")

	imgData, _, err := extractImageFromRequest(r)
	if err != nil {
		result.Error = fmt.Sprintf("cannot extract image, %v", err)
		return
	}

	sharepicData, err := MakeSharepic(r.Context(), sharepicCustomization{
		Name:    extractStringFromRequest(r, "template", "ilovefs"),
		Message: extractStringFromRequest(r, "message", ""),

		AuthorName: extractStringFromRequest(r, "authorName", "Jane Doe"),
		AuthorDesc: extractStringFromRequest(r, "authorDesc", ""),
	}, imgData)
	if err != nil {
		result.Error = fmt.Sprintf("cannot create sharepic, %v", err)
		return
	}
	result.Jpeg = sharepicData
}

// healthHandler will be queried by Docker to check if the service is still up.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "I'm Still Standing")
}

// LaunchWebserver and blocks until it finishes.
func LaunchWebserver() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/sharepic", sharepicHandler)

	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("web server finished, %v", server.ListenAndServe())
}

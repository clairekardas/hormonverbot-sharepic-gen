// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This file contains the backend program's entry point, initializing the
// modules and eventually starting the web server.

package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/gographics/imagick.v3/imagick"
	"gopkg.in/yaml.v3"
)

// apacheLogger formats Go's logger similar to Apache's output.
type apacheLogger struct{}

func (logger apacheLogger) Write(msg []byte) (int, error) {
	ts := time.Now().Format("02/Jan/2006:15:04:05 -0700")
	return fmt.Printf("[%s] %s", ts, msg)
}

// InitLogger to unify the loggers format with Apache's.
func InitLogger() {
	log.SetFlags(0)
	log.SetOutput(new(apacheLogger))
	log.SetPrefix("")
}

// InitLeastPrivilege drops privileges by a platform specific method.
func InitLeastPrivilege() {
	if err := ToLeastPrivilege(); err != nil {
		log.Fatalf("cannot drop privileges, %v", err)
	} else {
		log.Printf("dropped privileges successfully")
	}
}

var (
	// allowedMimeTypes is an allow list for MIME types of uploaded images.
	allowedMimeTypes map[string]struct{}

	// maxImageSize for uploads in bytes.
	maxImageSize int64
)

//go:embed inc/backend.yml
var backendConfig []byte

// InitEnvironmentConfig initializes configurations from inc/backend.yml.
func InitConfig() {
	type cfg struct {
		AllowedMimes []string `yaml:"allowed_mimes"`
		MaxImgSize   int64    `yaml:"max_img_size"`
	}
	var c cfg

	if err := yaml.Unmarshal(backendConfig, &c); err != nil {
		log.Fatalf("cannot unmarshal backend.yml, %v", err)
	}

	// allowedMimeTypes
	allowedMimeTypes = make(map[string]struct{})
	for _, mimeType := range c.AllowedMimes {
		allowedMimeTypes[mimeType] = struct{}{}
		log.Printf("allowed MIME %q", mimeType)
	}

	// maxImageSize
	maxImageSize = c.MaxImgSize
	log.Printf("set maximum image size to %dB", maxImageSize)
}

// IsTestRun if the tool was called with "--test-run".
func IsTestRun() bool {
	return len(os.Args[1:]) > 0 && os.Args[1] == "--test-run"
}

func main() {
	InitLogger()
	InitLeastPrivilege()
	InitConfig()
	InitSharepicConfig()

	imagick.Initialize()
	defer imagick.Terminate()

	if IsTestRun() {
		log.Print("test run succeeded")
		return
	}

	LaunchWebserver()
}

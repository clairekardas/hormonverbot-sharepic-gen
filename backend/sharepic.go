// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This file contains entry points for the sharepic generation. A bunch of logic
// is added during compile time via embed and loaded within the init function.
//
// For usage, the MakeSharepic function is the most relevant one.

package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// sharepicTemplate to be customized for each sharepic.Will be populated in this
// file's init function for all SVG templates.
var sharepicTemplate *template.Template

// sharepicCustomization contains all template fields from the sharepicTemplate
// and a name to identify the template, without the file extension.
type sharepicCustomization struct {
	Name    string
	Message string

	ImageData  string
	AuthorName string
	AuthorDesc string
}

// Template name to be used in sharepicTemplate.
func (cus sharepicCustomization) Template() string {
	return cus.Name + ".svg"
}

// sharepicConf describes the YAML configuration in an identically named .yml
// file for each .svg file in the templates directory. It contains the further
// configuration regarding the message overlay.
type sharepicConf struct {
	Sharepic struct {
		Width  int
		Height int
	}

	PictureBox struct {
		Width     int
		Height    int
		Grayscale bool
	} `yaml:"picture_box"`

	Font struct {
		Name      string
		Color     string
		Uppercase bool
		Sizes     []int
	}

	MaxLength struct {
		Message     int
		Author      int
		Description int
	} `yaml:"max_length"`

	MessageBox struct {
		Disable bool

		Width  int
		Height int

		MarginWidth  int `yaml:"margin_width"`
		MarginHeight int `yaml:"margin_height"`
	} `yaml:"message_box"`
}

// sharepicConfs maps the name of a template without a file extension to a
// sharepicConf, able to derive a generator instance.
var sharepicConfs map[string]sharepicConf

//go:embed inc/templates/*
var templatesFs embed.FS

// InitSharepicConfig from the embedded template files.
func InitSharepicConfig() {
	// Populate template with all SVG files.
	var sharepicTemplateErr error
	sharepicTemplate, sharepicTemplateErr = template.New("svg").ParseFS(templatesFs, "inc/templates/*.svg")
	if sharepicTemplateErr != nil {
		log.Fatalf("cannot parse templates, %v", sharepicTemplateErr)
	}

	// Create all template configurations from the YAML files.
	entries, entriesErr := templatesFs.ReadDir("inc/templates")
	if entriesErr != nil {
		log.Fatalf("cannot read embedded templates directory, %v", entriesErr)
	}

	sharepicConfs = make(map[string]sharepicConf)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		ymlConf, ymlConfErr := templatesFs.Open(filepath.Join("inc/templates", entry.Name()))
		if ymlConfErr != nil {
			log.Fatalf("cannot open template configuration %v, %v", entry, ymlConfErr)
		}

		var conf sharepicConf
		decodeErr := yaml.NewDecoder(ymlConf).Decode(&conf)
		if decodeErr != nil {
			log.Fatalf("failed to decode YAML, %v", decodeErr)
		}

		// Strip file extension, ".yml".
		key := entry.Name()[:len(entry.Name())-4]
		sharepicConfs[key] = conf

		log.Printf("loaded configuration for template %s", key)
	}
}

// MakeSharepic creates the sharepic from the passed user input.
func MakeSharepic(ctx context.Context, input sharepicCustomization, imageData []byte) ([]byte, error) {
	conf, ok := sharepicConfs[input.Name]
	if !ok {
		return nil, fmt.Errorf("no template %q available", input.Name)
	}

	fieldLengths := []struct {
		name   string
		max    int
		length int
	}{
		{"message", conf.MaxLength.Message, utf8.RuneCountInString(input.Message)},
		{"author", conf.MaxLength.Author, utf8.RuneCountInString(input.AuthorName)},
		{"description", conf.MaxLength.Description, utf8.RuneCountInString(input.AuthorDesc)},
	}
	for _, fieldLength := range fieldLengths {
		if fieldLength.max != 0 && fieldLength.length > fieldLength.max {
			return nil, fmt.Errorf("length of %s, %d, exceeds maximum %d",
				fieldLength.name, fieldLength.length, fieldLength.max)
		}
	}

	gen := generator{
		sharepicTempl: conf,
		customization: input,
		imageData:     imageData,
	}

	return gen.GenSharepic(ctx)
}

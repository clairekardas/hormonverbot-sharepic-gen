// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This file contains code to generate a "perfect" sharepic under the specified
// constraints from user data.
//
// For usage, one wants to create an instance of a generator and call the
// GenSharepic method on it afterwards.

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/gographics/imagick.v3/imagick"
)

// generator is used internally to pass state for sharepic generation.
//
// For usage, only the GenSharepic method is relevant.
type generator struct {
	sharepicTempl sharepicConf
	customization sharepicCustomization
	imageData     []byte

	// The values below MUST NOT be set as they are populated during execution.

	fontSize   int
	lineHeight int

	tmpfileSvg *bytes.Buffer
	lines      []string
}

// prepareInputText to mitigate SVG injections and perform uppercase conversions.
func (gen *generator) prepareInputText() error {
	fields := []struct {
		input     *string
		xmlEscape bool
	}{
		{&gen.customization.Message, false},
		{&gen.customization.AuthorName, true},
		{&gen.customization.AuthorDesc, true},
	}
	for _, field := range fields {
		if gen.sharepicTempl.Font.Uppercase {
			*field.input = strings.ToUpper(*field.input)
		}

		if field.xmlEscape {
			var buff bytes.Buffer
			if err := xml.EscapeText(&buff, []byte(*field.input)); err != nil {
				return fmt.Errorf("cannot escape XML string, %w", err)
			}
			*field.input = buff.String()
		}
	}

	return nil
}

// prepareInputImage rotates and crops the user submitted picture, the returned
// byte slice contains a JPEG image.
func (gen *generator) prepareInputImage() (picData []byte, err error) {
	mw := imagick.NewMagickWand()

	if err = mw.ReadImageBlob(gen.imageData); err != nil {
		return
	}

	if err = mw.SetImageFormat("JPEG"); err != nil {
		return
	}

	if gen.sharepicTempl.PictureBox.Grayscale {
		if err = mw.TransformImageColorspace(imagick.COLORSPACE_GRAY); err != nil {
			return
		}
	}

	// Calculate two possible dimensions of the original image, normalized to a
	// multiple of the desirable value. One side has to fit.
	wishWidth, wishHeight := 4*float64(gen.sharepicTempl.PictureBox.Width), 4*float64(gen.sharepicTempl.PictureBox.Height)
	baseWidth, baseHeight := float64(mw.GetImageWidth()), float64(mw.GetImageHeight())

	wOptX, wOptY := wishWidth, (wishWidth/baseWidth)*baseHeight
	hOptX, hOptY := (wishHeight/baseHeight)*baseWidth, wishHeight

	// Continue with the greater resolution. The image will be resized to this
	// value. However, most likely one side will be too long.
	wSize, hSize := wOptX*wOptY, hOptX*hOptY

	var x, y uint
	if wSize >= hSize {
		x, y = uint(wOptX), uint(wOptY)
	} else {
		x, y = uint(hOptX), uint(hOptY)
	}

	if err = mw.ResizeImage(x, y, imagick.FILTER_GAUSSIAN); err != nil {
		return
	}

	// Finally, crop the image to the desired resolution to the center.
	cropX, cropY := int(x/2-uint(wishWidth/2)), int(y/2-uint(wishHeight/2))

	if err = mw.CropImage(uint(wishWidth), uint(wishHeight), cropX, cropY); err != nil {
		return
	}

	if err = mw.AutoOrientImage(); err != nil {
		return
	}

	picData = mw.GetImageBlob()
	return
}

// createTemplate from the inc/templates/*.svg file.
func (gen *generator) createTemplate(ctx context.Context) error {
	picData, err := gen.prepareInputImage()
	if err != nil {
		return err
	}

	encChan := make(chan error, 1)
	go func() {
		gen.customization.ImageData = base64.StdEncoding.EncodeToString(picData)

		gen.tmpfileSvg = new(bytes.Buffer)
		encChan <- sharepicTemplate.ExecuteTemplate(gen.tmpfileSvg, gen.customization.Template(), gen.customization)
	}()

	select {
	case encErr := <-encChan:
		return encErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// findOptParams for both separation of the message into line and font size.
func (gen *generator) findOptParams(ctx context.Context) error {
	// Can be skipped iff there is no message to be shown.
	if gen.sharepicTempl.MessageBox.Disable {
		return nil
	}

	sentences, size, lineHeight, err := ConjureBox(
		ctx,
		gen.sharepicTempl.Font.Name, gen.sharepicTempl.Font.Sizes,
		strings.Split(gen.customization.Message, " "),
		gen.sharepicTempl.MessageBox.Width, gen.sharepicTempl.MessageBox.Height)
	if err != nil {
		return err
	}

	gen.fontSize = size
	gen.lineHeight = lineHeight

	gen.lines = make([]string, len(sentences))
	for i := 0; i < len(gen.lines); i++ {
		gen.lines[i] = strings.Join(sentences[i], " ")
	}

	return nil
}

// conjureSharepic from the prepared template and the calculated parameters.
func (gen *generator) conjureSharepic() (jpegData []byte, err error) {
	mw := imagick.NewMagickWand()

	if err = mw.ReadImageBlob(gen.tmpfileSvg.Bytes()); err != nil {
		return
	}

	if err = mw.ResizeImage(uint(gen.sharepicTempl.Sharepic.Width), uint(gen.sharepicTempl.Sharepic.Height), imagick.FILTER_LANCZOS); err != nil {
		return
	}

	if !gen.sharepicTempl.MessageBox.Disable {
		dw := imagick.NewDrawingWand()
		dw.SetFontSize(float64(gen.fontSize))

		if err = dw.SetFont(gen.sharepicTempl.Font.Name); err != nil {
			return
		}

		dwPw := imagick.NewPixelWand()
		if !dwPw.SetColor(gen.sharepicTempl.Font.Color) {
			return nil, fmt.Errorf("cannot use color %q for font's pixel wand", gen.sharepicTempl.Font.Color)
		}
		dw.SetFillColor(dwPw)

		for i, line := range gen.lines {
			dw.Annotation(
				float64(gen.sharepicTempl.MessageBox.MarginWidth),
				float64(gen.sharepicTempl.MessageBox.MarginHeight+((i+1)*gen.lineHeight)), line)
		}

		if err = mw.DrawImage(dw); err != nil {
			return
		}
	}

	if err = mw.StripImage(); err != nil {
		return
	}

	if err = mw.SetImageFormat("JPEG"); err != nil {
		return
	}
	if err = mw.SetCompression(imagick.COMPRESSION_JPEG); err != nil {
		return
	}
	if err = mw.SetCompressionQuality(80); err != nil {
		return
	}

	jpegData = mw.GetImageBlob()
	return
}

// GenSharepic based on the initial generator state.
func (gen *generator) GenSharepic(ctx context.Context) ([]byte, error) {
	var wg sync.WaitGroup
	var errCustomize, errOptimize error

	if err := gen.prepareInputText(); err != nil {
		return nil, err
	}

	wg.Add(2)

	go func() { errCustomize = gen.createTemplate(ctx); wg.Done() }()
	go func() { errOptimize = gen.findOptParams(ctx); wg.Done() }()

	wg.Wait()

	if errCustomize != nil && errOptimize != nil {
		return nil, fmt.Errorf("%w, %w", errCustomize, errOptimize)
	} else if errCustomize != nil {
		return nil, errCustomize
	} else if errOptimize != nil {
		return nil, errOptimize
	}

	return gen.conjureSharepic()
}

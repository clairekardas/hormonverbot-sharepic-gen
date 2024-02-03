// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This file contains functions to calculate an optimum regarding text size and
// line breaks for the multi-line message within the sharepic.
//
// For external usage, only the ConjureBox function is relevant.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"gopkg.in/gographics/imagick.v3/imagick"
)

// heightOverflowErr if the height exceeds the box's height.
var heightOverflowErr = fmt.Errorf("box height overflows")

// conjureWordDimensions calculates the rendered length of all word subsets.
//
// This function determines the length of all substrings of the words array,
// beginning from only the first word to all words. The two dimensional dims
// array will contain an array of the length of words, each value being itself
// an array - containing the width and height.
func conjureWordDimensions(ctx context.Context, font string, size int, words []string) (dims [][]int, err error) {
	if len(words) == 0 {
		return nil, fmt.Errorf("cannot work on an empty words array")
	}

	var dimsMutex, errMutex sync.Mutex
	dims = make([][]int, len(words))

	var wg sync.WaitGroup
	wg.Add(len(words))

	for i := range words {
		go func(i int) {
			defer wg.Done()

			var tmpErr error
			defer func() {
				if tmpErr != nil {
					errMutex.Lock()
					err = tmpErr
					errMutex.Unlock()
				}
			}()

			if tmpErr = ctx.Err(); tmpErr != nil {
				return
			}

			mw := imagick.NewMagickWand()
			dw := imagick.NewDrawingWand()
			pw := imagick.NewPixelWand()

			if tmpErr = mw.NewImage(0, 0, pw); tmpErr != nil {
				return
			}

			dw.SetFontSize(float64(size))
			if tmpErr = dw.SetFont(font); tmpErr != nil {
				return
			}

			fm := mw.QueryFontMetrics(dw, strings.Join(words[:i+1], " "))

			dimsMutex.Lock()
			defer dimsMutex.Unlock()
			dims[i] = []int{int(fm.TextWidth), int(fm.TextHeight)}
		}(i)
	}

	wg.Wait()

	return
}

// conjureBoxWrap creates sentences of the words for the font within a box.
//
// By utilizing the conjureWordDimensions function, sentences are being built
// to be rendered as text not exceeding the box's borders. If it is not possible
// to create sentences within the given constraints, the heightOverflow error is
// returned.
func conjureBoxWrap(ctx context.Context, font string, size int, words []string, boxWidth, boxHeight int) (sentences [][]string, lineHeight int, err error) {
	min := func(x, y int) int {
		if x > y {
			return y
		}
		return x
	}

	height := 0

	for len(words) > 0 {
		// Limit to a maximum of eight words as this is sufficient for our use case.
		lineOpts, lineOptsErr := conjureWordDimensions(ctx, font, size, words[:min(len(words), 8)])
		if lineOptsErr != nil {
			return nil, 0, fmt.Errorf("cannot conjure dimensions, %w", lineOptsErr)
		}

		var i int
		for i = 0; i < len(lineOpts) && lineOpts[i][0] < boxWidth; i++ {
		}
		i--

		if i < 0 {
			return nil, 0, fmt.Errorf("cannot find line options fitting box width for '%v'", words)
		}

		// +1 as lineOpts with index 0 represents one word and [:0] would be the
		// empty slice. Generalized, index i represents i+1 words.
		sentences = append(sentences, words[:i+1])
		words = words[i+1:]

		// Multiply with 1.1 for some margin between the text; highly opinionated.
		lineHeight = int(1.1 * float64(lineOpts[i][1]))

		height += lineHeight
		if height > boxHeight {
			return nil, 0, heightOverflowErr
		}
	}

	return
}

// ConjureBox calculates the optimal sentences and font size.
//
// This is achieved by a parallel execution of the conjureBoxWrap function with
// varying font sizes. Eventually, the "biggest" results will be used.
func ConjureBox(ctx context.Context, font string, sizes []int, words []string, boxWidth, boxHeight int) (sentences [][]string, size, lineHeight int, err error) {
	resolutionChan := make(chan struct {
		sentences  [][]string
		size       int
		lineHeight int
		err        error
	})

	for _, testSize := range sizes {
		go func(size int) {
			sentences, lineHeight, err := conjureBoxWrap(ctx, font, size, words, boxWidth, boxHeight)
			resolutionChan <- struct {
				sentences  [][]string
				size       int
				lineHeight int
				err        error
			}{sentences, size, lineHeight, err}
		}(testSize)
	}

	for range sizes {
		resolution := <-resolutionChan

		if resolution.err != nil && !errors.Is(resolution.err, heightOverflowErr) {
			log.Printf("size %d resulted in unexpected error, %v", resolution.size, resolution.err)
		}
		if resolution.err != nil || resolution.size <= size {
			continue
		}

		sentences = resolution.sentences
		lineHeight = resolution.lineHeight
		size = resolution.size
	}

	if size <= 0 {
		return nil, 0, 0, fmt.Errorf("cannot select a fitting font size for input")
	}

	return
}

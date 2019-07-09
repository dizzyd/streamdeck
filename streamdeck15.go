// ***************************************************************************
//
//  Copyright 2019 David (Dizzy) Smith, dizzyd@dizzyd.com
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
// ***************************************************************************
package streamdeck

import (
	"fmt"
	"github.com/pkg/errors"
)

const columns = 5

type streamDeck15 struct {
	*streamDeckBase
}

func (deck streamDeck15) Reset() error {
	return deck.device.WriteFeature([]byte{0x0b, 0x63})
}

// Set the handler for all keys
func (deck streamDeck15) SetGlobalKeyHandler(fn KeyPressFn) {
	deck.handlers[255] = fn
}

// Clear the handler for all keys
func (deck streamDeck15) ClearGlobalKeyHandler() {
	delete(deck.handlers, 255)
}

func (deck streamDeck15) SetKeyHandler(key byte, fn KeyPressFn) error {
	if key < 0 || key > 14 {
		return errors.WithStack(ErrInvalidKey)
	}
	deck.handlers[key] = fn
	return nil
}

func (deck streamDeck15) ClearKeyHandler(key byte) error {
	if key < 0 || key > 14 {
		return errors.WithStack(ErrInvalidKey)
	}
	delete(deck.handlers, key)
	return nil
}

func (deck streamDeck15) SetKeyImage(key byte, filename string) error {
	// The deck uses 1-based numbering for images, so we invert the key into an
	// id and make sure to add one
	id := deck.invertKeyOrId(key) + 1

	img, err := loadImage(filename)
	if err != nil {
		return err
	}

	// Streamdeck images are right-to-left/top-to-bottom (BGR representation)
	// PNG images are left-to-right/top-to-bottom (RGBA representation)
	// Thus we walk scanline-by-scanline, placing the rightmost (max x) pixel first
	var imageBytes []byte
	if img != nil {
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Max.X; x > bounds.Min.X; x-- {
				r, g, b, _ := img.At(x, y).RGBA()
				imageBytes = append(imageBytes, byte(b), byte(g), byte(r))
			}
		}
	} else {
		// If no image was loaded, generate an empty (black) image
		imageBytes = make([]byte, 72*72*3)
	}

	// The following headers and constants are taken from:
	// https://github.com/abcminiuser/python-elgato-streamdeck/blob/master/src/StreamDeck/Devices/StreamDeckOriginal.py#L131
	// (MIT License)
	header1 := []byte{
		0x02, 0x01, 0x01, 0x00, 0x00, id, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x42, 0x4d, 0xf6, 0x3c, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x48, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xc0, 0x3c, 0x00, 0x00, 0xc4, 0x0e,
		0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	header2 := []byte{
		0x02, 0x01, 0x02, 0x00, 0x01, id, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	imageBytesOnPage1 := 2583 * 3

	err = writePage(deck.device, header1, imageBytes[:imageBytesOnPage1])
	if err != nil {
		return errors.Wrapf(err, "failed to write page 1")
	}

	err = writePage(deck.device, header2, imageBytes[imageBytesOnPage1:])
	if err != nil {
		return errors.Wrapf(err, "failed to write page 2")
	}

	return nil
}

func (deck streamDeck15) ClearKeyImage(key byte) error {
	return deck.SetKeyImage(key, "")
}

func (deck streamDeck15) ProcessEvents(timeout int) error {
	report := make([]byte, 16)
	bytes, err := deck.device.ReadTimeout(report, timeout)
	if err != nil {
		return errors.Wrapf(err, "error reading key press")
	}

	if bytes > 0 {
		reportId := report[0]
		if reportId == 1 {
			// TODO: Revisit for chording support
			for id, state := range report[1:] {
				key := deck.invertKeyOrId(byte(id))
				if state == 1 {
					deck.dispatchKey(255)
					deck.dispatchKey(key)
				}
			}
		} else {
			fmt.Printf("Ignoring unexpected report from device: %d\n", reportId)
		}
	}

	return nil
}

func (deck streamDeck15) dispatchKey(key byte) {
	handler, exists := deck.handlers[key]
	if exists {
		if !handler(key) {
			delete(deck.handlers, key)
		}
	}
}

func (deck streamDeck15) invertKeyOrId(value byte) byte {
	col := value % columns
	return (value - col) + ((columns - 1) - col)
}


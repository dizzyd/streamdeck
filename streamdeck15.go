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
	id := deck.keyToInternalId(key)
	if id == 255 {
		return errors.WithStack(ErrInvalidKey)
	}

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
				key := deck.internalIdToKey(byte(id))
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

func (deck streamDeck15) internalIdToKey(id byte) byte {
	// The deck is setup right-to-left, with 5 keys on each row; translate to a zero-based, left-to-right index
	switch id {
	case 0x0:
		return 4
	case 0x1:
		return 3
	case 0x2:
		return 2
	case 0x3:
		return 1
	case 0x4:
		return 0
	case 0x5:
		return 9
	case 0x6:
		return 8
	case 0x7:
		return 7
	case 0x8:
		return 6
	case 0x9:
		return 5
	case 0xA:
		return 14
	case 0xB:
		return 13
	case 0xC:
		return 12
	case 0xD:
		return 11
	case 0xE:
		return 10
	default:
		return 255
	}
}

func (deck streamDeck15) keyToInternalId(key byte) byte {
	// Translate a left-to-right index to deck-native ID
	switch key {
	case 4:
		return 0x0
	case 3:
		return 0x1
	case 2:
		return 0x2
	case 1:
		return 0x3
	case 0:
		return 0x4
	case 9:
		return 0x5
	case 8:
		return 0x6
	case 7:
		return 0x7
	case 6:
		return 0x8
	case 5:
		return 0x9
	case 14:
		return 0xA
	case 13:
		return 0xB
	case 12:
		return 0xC
	case 11:
		return 0xD
	case 10:
		return 0xE
	default:
		return 255
	}
}

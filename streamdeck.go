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
	"github.com/dizzyd/hid"
	"github.com/pkg/errors"
)

const vendor = 4057

const streamDeck15Id = 96

var ErrUnknownDevice = errors.New("unknown device")
var ErrNoDevices = errors.New("no devices found")
var ErrInvalidKey = errors.New("invalid key")

type KeyPressFn func(key byte) bool

// StreamDeck provides an interface for controlling a deck. Keys are
// zero-based, left-to-right, top-to-bottom
type StreamDeck interface {
	// Reset the device
	Reset() error

	// Set the handler for all keys
	SetGlobalKeyHandler(fn KeyPressFn)

	// Clear the handler for all keys
	ClearGlobalKeyHandler()

	// Set the handler for a given key
	SetKeyHandler(key byte, fn KeyPressFn) error

	// Remove the handler for a given key
	ClearKeyHandler(key byte) error

	// Set the image on a given key; only PNG is currently supported
	SetKeyImage(key byte, filename string) error

	// Clear the image on a given key
	ClearKeyImage(key byte) error

	// Process any key press events; blocks up to timeout milliseconds. Use
	// zero as a timeout for non-blocking behaviour, use -1 for blocking until
	// a key is pressed.
	ProcessEvents(timeout int) error
}

// Base structure for all StreamDeck implementations
type streamDeckBase struct {
	device   *hid.Device
	handlers map[byte]KeyPressFn
}

// OpenStreamDeck finds the first available deck and returns an instance of the StreamDeck interface
func OpenStreamDeck() (StreamDeck, error) {
	devices := hid.Enumerate(vendor, 0)
	for _, deviceInfo := range devices {
		device, err := deviceInfo.Open()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to open device %d-%d", deviceInfo.VendorID, deviceInfo.ProductID)
		}

		switch deviceInfo.ProductID {
		case streamDeck15Id:
			return &streamDeck15{streamDeckBase: newStreamDeckBase(device)}, nil
		default:
			return nil, errors.New(fmt.Sprintf("unknown device %d-%d", deviceInfo.VendorID, deviceInfo.ProductID))
		}
	}

	return nil, ErrNoDevices
}

func newStreamDeckBase(device *hid.Device) *streamDeckBase {
	return &streamDeckBase{
		device:   device,
		handlers: make(map[byte]KeyPressFn),
	}
}

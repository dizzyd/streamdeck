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
	"github.com/dizzyd/hid"
	"github.com/pkg/errors"
	"image"
	_ "image/png"
	"os"
)

func writePage(device *hid.Device, header, payload []byte) error {
	var page []byte
	page = append(page, header...)
	page = append(page, payload...)
	_, err := device.Write(page)
	return errors.WithStack(err)
}

func loadImage(filename string) (image.Image, error) {
	if filename == "" {
		return nil, nil
	}

	reader, err := os.Open(filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return img, nil
}

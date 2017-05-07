// Copyright 2017 orijtech. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package px500_test

import (
	"fmt"
	"log"

	"github.com/orijtech/500px/v1"
)

func Example_client_ListPhotos() {
	client, err := px500.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	preq := &px500.PhotoRequest{
		LimitPerPage:  10,
		MaxPageNumber: 2,
		Feature:       px500.FeaturePopular,
	}

	pagesChan, cancelFn, err := client.ListPhotos(preq)

	if err != nil {
		log.Fatal(err)
	}

	count := uint64(0)
	for page := range pagesChan {
		fmt.Printf("Page: #%d\n\n", page.PageNumber)
		for i, photo := range page.Photos {
			count += 1
			fmt.Printf("#%d: %#v\n\n", i, photo)
		}

		if count >= 13 {
			cancelFn()
		}
		fmt.Printf("\n\n")
	}
}

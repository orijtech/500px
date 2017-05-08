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
	"encoding/json"
	"fmt"
	"log"
	"os"

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

func Example_client_PhotoSearch() {
	client, err := px500.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ps := &px500.PhotoSearch{
		Term:          "the universe",
		LimitPerPage:  10,
		MaxPageNumber: 2,
	}

	pagesChan, cancelFn, err := client.SearchPhotos(ps)

	if err != nil {
		log.Fatal(err)
	}

	count := uint64(0)
	for page := range pagesChan {
		fmt.Printf("Page: #%d\n\n", page.PageNumber)
		if err := page.Err; err != nil {
			fmt.Printf("err: %v\n", err)
			continue
		}

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

func Example_client_PhotoByID() {
	client, err := px500.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	photo, err := client.PhotoByID("210717663")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The Photo's info: %#v\n", photo)
}

func Example_client_CommentsForPhoto() {
	client, err := px500.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	cr := &px500.CommentsRequest{
		PhotoID: "210717663",
		Nested:  true,
	}

	pagesChan, cancelFn, err := client.CommentsForPhoto(cr)

	if err != nil {
		log.Fatal(err)
	}

	count := uint64(0)
	for page := range pagesChan {
		fmt.Printf("Page: #%d\n\n", page.PageNumber)
		if err := page.Err; err != nil {
			fmt.Printf("err: %v\n", err)
			continue
		}

		for i, comment := range page.Comments {
			count += 1
			fmt.Printf("#%d: %#v\n\n", i, comment)
			for j, reply := range comment.Replies {
				fmt.Printf("\t\tReply: #%d reply: %#v\n\n", j, reply)
			}
		}

		if count >= 24 {
			cancelFn()
		}
		fmt.Printf("\n\n")
	}
}

func Example_client_UploadPhoto() {
	client, err := px500.NewOAuth1ClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open("./v1/testdata/sfPanorama.jpeg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	photo, err := client.UploadPhoto(&px500.UploadRequest{
		Body:     f,
		Filename: "billion dollar view",
		PhotoInfo: &px500.Photo{
			Title: "SF Panorama, Billion Dollar View",
			ISO:   "iPhone 6",
			Tags:  []string{"sf", "bayBridge", "California", "Piers"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Uploaded photo: %#v\n", photo)
}

func Example_oAuth1TokenFromEnv() {
	token, err := px500.OAuth1AuthorizationByEnv()
	if err != nil {
		log.Fatal(err)
	}
	blob, _ := json.Marshal(token)

	outpath := "500px-credentials.json"
	f, err := os.Create(outpath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if _, err := f.Write(blob); err == nil {
		fmt.Printf("Successfully saved the JSON credentials to %q\n", outpath)
	} else {
		fmt.Printf("err: %v\n", err)
	}
}

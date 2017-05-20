# 500px
Go API client for 500px

## Usage

* Preamble
```go
import (
	"fmt"
	"log"

	"github.com/orijtech/500px/v1"
)
```

* List photos
```go
func listPhotos() {
	client, err := px500.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	preq := new(px500.PhotoRequest)
	preq.LimitPerPage = 10
	preq.MaxPageNumber = 2
	preq.Feature = px500.FeaturePopular

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

* Search for photos
```go
func searchForPhotos() {
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
```

* Retrieve a photo by ID
```go
func findPhotoByID() {
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

```

* Retrieve comments for a photo
```go
func retrieveCommentsForPhoto() {
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
```

* Upload a photo
```go
func uploadAPhoto() {
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
```

* Update a photo
```go
func updatePhoto() {
	client, err := px500.NewOAuth1ClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	photo, err := client.UpdatePhoto(&px500.UpdateRequest{
		PhotoID: "211020335",
		Content: &px500.Photo{
			Title:  "Updated in tests",
			Tags:   []string{"tests", "api-client", "golang"},
			Camera: "iphone 6",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated photo: %#v\n", photo)
}
```

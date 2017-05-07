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
```

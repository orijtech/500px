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

package px500

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/orijtech/otils"
)

type Comment struct {
	ID       int64  `json:"id"`
	Body     string `json:"body,omitempty"`
	Author   *User  `json:"user,omitempty"`
	AuthorID int64  `json:"user_id,omitempty"`

	ToWhomUserID int64 `json:"to_whom_user_id,omitempty"`

	CreatedAt *time.Time `json:"created_at"`
	ParentID  int64      `json:"parent_id"`
	Flagged   bool       `json:"flagged"`
	Rating    uint64     `json:"rating"`

	Voted bool `json:"voted"`

	Replies []*Comment `json:"replies,omitempty"`
}

type CommentsPage struct {
	MediaType  string `json:"media_type"`
	PageNumber int64  `json:"current_page"`
	TotalPages int64  `json:"total_pages"`
	TotalItems int64  `json:"total_items"`

	Comments []*Comment `json:"comments"`

	Err error
}

type CommentsRequest struct {
	PhotoID    string `json:"photo_id"`
	PageNumber int64  `json:"page"`

	// Nested if set returns comments in nested for
	// that is with replies to comments included.
	Nested bool `json:"nested"`

	MaxPageNumber int64 `json:"max_page_number"`
}

var errNilCommentsRequest = errors.New("expecting a non-nil commentsRequest")

func (creq *CommentsRequest) Validate() error {
	if creq == nil {
		return errNilCommentsRequest
	}
	if creq.PhotoID == "" {
		return errEmptyPhotoID
	}
	return nil
}

type commentsPager struct {
	PageNumber int64 `json:"page"`
	Nested     bool  `json:"nested"`
}

func (cp *commentsPager) adjustPaginationParams() {
	// 500px comment page numbers are 1-based.
	if cp.PageNumber <= 0 {
		cp.PageNumber = 1
	}
}

func (c *Client) CommentsForPhoto(creq *CommentsRequest) (pagesChan chan *CommentsPage, cancelFn func(), err error) {
	if err := creq.Validate(); err != nil {
		return nil, nil, err
	}

	cpager := &commentsPager{
		PageNumber: creq.PageNumber,
		Nested:     creq.Nested,
	}

	cpager.adjustPaginationParams()

	maxPageNumber := creq.MaxPageNumber
	pageExceeds := func(pageNumber int64) bool {
		if maxPageNumber <= 0 {
			return false
		}
		return pageNumber >= maxPageNumber
	}

	cancelChan, cancelFn := makeCanceler()
	pagesChan = make(chan *CommentsPage)

	go func() {
		defer close(pagesChan)
		throttle := time.Duration(200 * time.Millisecond)
		photoID := creq.PhotoID

		for {
			cpage := new(CommentsPage)
			qv, err := otils.ToURLValues(cpager)
			if err != nil {
				cpage.Err = err
				pagesChan <- cpage
				return
			}
			qv.Set("consumer_key", c.consumerKey())

			fullURL := fmt.Sprintf("%s/photos/%s/comments?%s", baseURL, photoID, qv.Encode())
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				cpage.Err = err
				pagesChan <- cpage
				return
			}

			slurp, _, err := c.doAuthAndRequest(req)
			if err != nil {
				cpage.Err = err
				pagesChan <- cpage
				return
			}

			if err := json.Unmarshal(slurp, cpage); err != nil {
				cpage.Err = err
				pagesChan <- cpage
				return
			}

			// No more comments to retrieve since
			// pages are meant to be contiguous and filled
			// with comments before we encounter the first
			// page with no comments.
			if len(cpage.Comments) < 1 {
				return
			}

			pagesChan <- cpage

			select {
			case <-cancelChan:
				return
			case <-time.After(throttle):
			}

			if pageExceeds(cpager.PageNumber) {
				break
			}

			cpager.PageNumber += 1
		}
	}()

	return pagesChan, cancelFn, nil
}

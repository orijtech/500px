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
	"net/url"
	"time"

	"github.com/orijtech/otils"
)

type PhotoRequest struct {
	// Feature is always required.
	Feature Feature `json:"feature"`

	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Only     string    `json:"only"`
	Exclude  string    `json:"exclude"`
	SortBy   SortOrder `json:"sort"`

	ImageSize Size `json:"image_size"`

	IncludeStore Store    `json:"include_store"`
	Tags         []string `json:"tags"`

	// PageNumber is the specific page in the photo stream.
	// Note that Page numbering is 1-based.
	PageNumber int64 `json:"page"`

	LimitPerPage int `json:"rpp"`

	MaxPageNumber int64 `json:"-"`
}

type PhotoPage struct {
	Feature     Feature                `json:"feature"`
	Filters     map[string]interface{} `json:"filters"`
	CurrentPage int                    `json:"current_page"`
	TotalPage   int                    `json:"total_page"`
	TotalItems  int                    `json:"total_items"`
	Photos      []*Photo               `json:"photos"`

	Err        error
	PageNumber int64
}

type GalleryKind uint

const (
	// Any photo on 500px.
	GalleryGeneral GalleryKind = 0

	// Marketplace photos.
	GalleryLightbox GalleryKind = 1

	// Photos displayed on the portfolio page.
	GalleryPortfolio GalleryKind = 3

	// Photos uploaded by the gallery owner.
	GalleryProfile GalleryKind = 4

	// Photos favorited by the gallery owner.
	GalleryFavorite GalleryKind = 5
)

func (p *PhotoRequest) adjustPaginationParams() {
	if p.PageNumber <= 0 {
		p.PageNumber = 1
	}

	if p.LimitPerPage <= 0 {
		p.LimitPerPage = 20
	}

	if p.LimitPerPage >= 100 {
		p.LimitPerPage = 100
	}
}

type Gallery struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Title       string `json:"name"`
	Description string `json:"description"`
	Subtitle    string `json:"subtitle"`

	ItemCount uint64 `json:"items_count"`
	Private   bool   `json:"privacy"`

	Kind GalleryKind `json:"kind"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`

	CustomSlug string     `json:"custom_path"`
	FeaturedAt *time.Time `json:"featured_at"`

	FeaturedInEditorsChoice bool `json:"editors_choice"`

	// TokenSignature is set only for a private
	// gallery URL and it is only returned if
	// the request was made by the gallery owner.
	TokenSignature string `json:"token"`

	LastAddedPhoto *Photo `json:"last_added_photo"`

	User *User `json:"user"`
}

type PhotoSearch struct {
	Term string `json:"term"`
	Tag  string `json:"tag"`

	Only        Category `json:"only"`
	Exclude     Category `json:"exclude"`
	ExcludeNSFW bool     `json:"exclude_nude"`

	// PageNumber is the specific page in the photo stream.
	// Note that Page numbering is 1-based.
	PageNumber int64 `json:"page"`

	LimitPerPage int `json:"rpp"`

	Tags   []string `json:"tags"`
	UserID string   `json:"user_id"`

	ImageSizes   []Size        `json:"image_size"`
	LicenseTypes []LicenseType `json:"license_type"`

	SortBy SortOrder `json:"sort"`

	MaxPageNumber int64 `json:"-"`
}

var errNilPhotoSearch = errors.New("expecting a non-nil photoSearch")

func (ps *PhotoSearch) adjustPaginationParams() {
	if ps.PageNumber <= 0 {
		ps.PageNumber = 1
	}

	if ps.LimitPerPage <= 0 {
		ps.LimitPerPage = 20
	}

	if ps.LimitPerPage >= 100 {
		ps.LimitPerPage = 100
	}
}

func (c *Client) SearchPhotos(ops *PhotoSearch) (resChan chan *PhotoPage, cancel func(), err error) {
	if ops == nil {
		return nil, nil, errNilPhotoSearch
	}

	ps := new(PhotoSearch)
	*ps = *ops

	maxPageNumber := ps.MaxPageNumber
	pageExceeds := func(page int64) bool {
		if maxPageNumber <= 0 {
			return false
		}
		return page >= maxPageNumber
	}

	resChan = make(chan *PhotoPage)
	cancelChan, cancelFn := makeCanceler()
	go func() {
		defer close(resChan)
		throttle := time.Duration(150 * time.Millisecond)

		for {
			pp := new(PhotoPage)
			qv, err := otils.ToURLValues(ps)
			if err != nil {
				pp.Err = err
				resChan <- pp
				return
			}
			qv.Set("consumer_key", c.consumerKey())

			fullURL := fmt.Sprintf("%s/photos/search?%s", baseURL, qv.Encode())
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				pp.Err = err
				resChan <- pp
				return
			}

			slurp, _, err := c.doAuthAndRequest(req)
			if err != nil {
				pp.Err = err
				resChan <- pp
				return
			}

			if err := json.Unmarshal(slurp, pp); err != nil {
				pp.Err = err
				resChan <- pp
				return
			}

			// If there are no more photos returned, just end it
			if len(pp.Photos) < 1 {
				resChan <- pp
				return
			}

			pp.PageNumber = ps.PageNumber

			resChan <- pp
			select {
			case <-cancelChan:
				return
			case <-time.After(throttle):
			}

			if pageExceeds(ps.PageNumber) {
				break
			}

			ps.PageNumber += 1
		}
	}()

	return resChan, cancelFn, nil
}

var errEmptyPhotoID = errors.New("expecting a non-empty photoID")

type PhotoWrap struct {
	Photo *Photo `json:"photo"`
}

func (c *Client) PhotoByID(photoID string) (*Photo, error) {
	if photoID == "" {
		return nil, errEmptyPhotoID
	}
	qv := make(url.Values)
	qv.Set("consumer_key", c.consumerKey())

	fullURL := fmt.Sprintf("%s/photos/%s?%s", baseURL, photoID, qv.Encode())
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	slurp, _, err := c.doAuthAndRequest(req)
	if err != nil {
		return nil, err
	}
	pwrap := new(PhotoWrap)
	if err := json.Unmarshal(slurp, pwrap); err != nil {
		return nil, err
	}
	return pwrap.Photo, nil
}

type LicenseType int

const (
	LicenseStandard500PX LicenseType = iota
	LicenseCreativeCommonsNonCommericalAttribution
	LicenseCreativeCommonsNonCommericalNoDerivative
	LicenseCreativeCommonsNonCommericalShareAlike
	LicenseCreativeCommonsLicenseAttribution
	LicenseCreativeCommonsLicenseNoDerivatives
	LicenseCreativeCommonsLicenseShareAlike
	LicenseCreativeCommonsLicensePublicDomainMark1Point0
	LicenseCreativeCommonsLicensePublicDomainDedication
)

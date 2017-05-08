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
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/orijtech/otils"

	"github.com/odeke-em/go-uuid"
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

type Photo struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`

	Title        otils.NullableString `json:"name"`
	Description  otils.NullableString `json:"description"`
	Camera       otils.NullableString `json:"camera"`
	Lens         otils.NullableString `json:"lens"`
	FocalLength  otils.NullableString `json:"focal_length"`
	ISO          otils.NullableString `json:"iso"`
	ShutterSpeed otils.NullableString `json:"shutter_speed"`
	Aperture     otils.NullableString `json:"aperture"`
	ViewCount    uint64               `json:"times_viewed"`
	Rating       float32              `json:"rating"`
	Status       int                  `json:"status"`
	CreatedAt    *time.Time           `json:"created_at"`
	Category     Category             `json:"category"`
	Location     otils.NullableString `json:"location"`

	HighResolutionUploaded int `json:"high_res_uploaded"`

	Private bool `json:"privacy"`

	Latitude  float32    `json:"latitude"`
	Longitude float32    `json:"longitude"`
	TakenAt   *time.Time `json:"taken_at"`
	ForSale   bool       `json:"for_sale"`

	Width  int `json:"width"`
	Height int `json:"height"`

	VoteCount      uint64 `json:"votes_count"`
	FavoritesCount uint64 `json:"favorites_count"`
	CommentCount   uint64 `json:"comments_count"`

	NSFW bool `json:"nsfw"`

	SalesCount uint64 `json:"sales_count"`

	HighestRating float32 `json:"highest_rating"`

	HighestRatingDate *time.Time `json:"highest_rating_date"`

	Converted otils.NumericBool `json:"converted"`

	Author *User `json:"user"`

	GalleryCount uint64 `json:"galleries_count"`

	Feature Feature `json:"feature"`

	CanvasPrint bool `json:"store_print"`
	InDownload  bool `json:"store_download"`

	// Voted reports whether the currently
	// authenticated user has voted on this photo.
	Voted bool `json:"voted"`

	// Purchased reports whether the currently
	// authenticated user has purchased this photo.
	Purchased bool `json:"purchased"`

	Comments []*Comment `json:"comments"`

	FeaturedInEditorsChoice bool `json:"editors_choice"`

	Tags []string `json:"tags"`
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

type UploadRequest struct {
	Filename    string    `json:"filename"`
	Body        io.Reader `json:"-"`
	PhotoInfo   *Photo    `json:"photo"`
	ContentType string    `json:"content_type"`
}

func (ur *UploadRequest) nonBlankFilename() string {
	fname := strings.TrimSpace(ur.Filename)
	if fname == "" {
		fname = uuid.NewRandom().String()
	}
	return fname
}

var (
	errNilBody  = errors.New("expecting a non-nil body")
	errNilPhoto = errors.New("expecting non-nil photo information")
)

func (ureq *UploadRequest) Validate() error {
	if ureq == nil || ureq.Body == nil {
		return errNilBody
	}
	if ureq.PhotoInfo == nil {
		return errNilPhoto
	}
	return nil
}

func (c *Client) UploadPhoto(ureq *UploadRequest) (photo *Photo, err error) {
	if err := ureq.Validate(); err != nil {
		return nil, err
	}

	qv, err := otils.ToURLValues(ureq.PhotoInfo)
	if err != nil {
		// TODO: Figure out if we can clean up the
		// previously created upload initialization.
		return nil, err
	}

	prc, pwc := io.Pipe()
	mpartW := multipart.NewWriter(pwc)

	go func() {
		body := ureq.Body
		formFile, err := mpartW.CreateFormFile("file", ureq.nonBlankFilename())
		if err != nil {
			return
		}
		_, _ = io.Copy(formFile, body)

		contentType := strings.TrimSpace(ureq.ContentType)
		if contentType == "" {
			contentType, _, _ = fDetectContentType(body)
		}
		writeStringFormField(mpartW, "Content-Type", contentType)

		_ = mpartW.Close()
		_ = pwc.Close()
	}()

	fullURL := fmt.Sprintf("%s/photos/upload?%s", baseURL, qv.Encode())
	req, err := http.NewRequest("POST", fullURL, prc)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mpartW.FormDataContentType())

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

func writeStringFormField(mw *multipart.Writer, key, value string) {
	if value != "" {
		w, err := mw.CreateFormField(key)
		if err == nil {
			io.WriteString(w, value)
		}
	}
}

func fDetectContentType(r io.Reader) (ct string, err error, seekable bool) {
	if r == nil {
		return "", errNilBody, false
	}

	seeker, seekable := r.(io.Seeker)
	if !seekable {
		// Unfortunately we can't sniff it
		// without modifying the content of the body
		return "", nil, false
	}

	sniffBuf := make([]byte, 512)
	n, err := io.ReadAtLeast(r, sniffBuf, 1)
	if err != nil {
		return "", err, seekable
	}

	// Otherwise seek back
	_, _ = seeker.Seek(int64(n), io.SeekStart)
	contentType := http.DetectContentType(sniffBuf)
	return contentType, nil, seekable
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

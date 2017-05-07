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
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/orijtech/otils"
)

const (
	baseURL = "https://api.500px.com/v1"
)

type Feature string

const (
	FeaturePopular        Feature = "popular"
	FeatureUpcoming       Feature = "upcoming"
	FeatureEditors        Feature = "editors"
	FeatureFreshToday     Feature = "fresh_today"
	FeatureFreshYesterday Feature = "fresh_yesterday"
	FeatureFreshWeek      Feature = "fresh_week"
	FeatureUser           Feature = "user"
	FeatureUserFriends    Feature = "user_friends"
	FeatureUserFavorites  Feature = "user_favorites"
)

type SortOrder string

const (
	SortCreatedAt      SortOrder = "created_at"
	SortRating         SortOrder = "rating"
	SortTimesViewed    SortOrder = "times_viewed"
	SortVotesCount     SortOrder = "votes_count"
	SortFavoritesCount SortOrder = "favorites_count"
	SortCommentsCount  SortOrder = "comments_count"
	SortTakenAt        SortOrder = "taken_at"
)

type Store string

const (
	StoreDownload Store = "store_download"
	StorePrint    Store = "store_print"
)

type Size uint

const (
	Size1 Size = 1
	Size2 Size = 2
	Size3 Size = 3
	Size4 Size = 4
)

var (
	errNilPhotoRequest = errors.New("expecting a non-nil photoRequest")
	errEmptyFeature    = errors.New("expecting a non-empty feature")
)

type Client struct {
	sync.RWMutex

	rt http.RoundTripper

	_consumerKey string
}

func NewClient(keys ...string) (*Client, error) {
	consumerKey := otils.FirstNonEmptyString(keys...)
	if consumerKey != "" {
		return &Client{_consumerKey: consumerKey}, nil
	}
	return NewClientFromEnv()
}

var env500PxAPIKey = "FIVE00_PX_API_KEY"

func NewClientFromEnv() (*Client, error) {
	consumerKey := strings.TrimSpace(os.Getenv(env500PxAPIKey))
	if consumerKey == "" {
		return nil, fmt.Errorf("%q was not found in your environment", env500PxAPIKey)
	}
	return &Client{_consumerKey: consumerKey}, nil
}

func (preq *PhotoRequest) Validate() error {
	if preq == nil {
		return errNilPhotoRequest
	}
	if preq.Feature == "" {
		return errEmptyFeature
	}
	return nil
}

func makeCanceler() (<-chan bool, func()) {
	cancelChan := make(chan bool)
	var cancelOnce sync.Once
	cancelFn := func() {
		cancelOnce.Do(func() {
			close(cancelChan)
		})
	}

	return cancelChan, cancelFn
}

func (c *Client) SetHTTPRoundTripper(rt http.RoundTripper) {
	c.Lock()
	defer c.Unlock()

	c.rt = rt
}

func (c *Client) consumerKey() string {
	c.RLock()
	defer c.RUnlock()

	return c._consumerKey
}

func (c *Client) httpClient() *http.Client {
	c.RLock()
	rt := c.rt
	c.RUnlock()

	if rt == nil {
		rt = http.DefaultTransport
	}

	return &http.Client{Transport: rt}
}

var errUnimplemented = errors.New("unimplemented")

func (c *Client) doAuthAndRequest(req *http.Request) ([]byte, http.Header, error) {
	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if !otils.StatusOK(res.StatusCode) {
		errMsg := res.Status
		if res.Body != nil {
			slurp, _ := ioutil.ReadAll(res.Body)
			if len(slurp) > 0 {
				errMsg = string(slurp)
			}
		}
		return nil, res.Header, errors.New(errMsg)
	}

	slurp, err := ioutil.ReadAll(res.Body)
	return slurp, res.Header, err
}

func (c *Client) ListPhotos(oreq *PhotoRequest) (pagesChan chan *PhotoPage, cancelFn func(), err error) {
	if err := oreq.Validate(); err != nil {
		return nil, nil, err
	}

	preq := new(PhotoRequest)
	if oreq != nil {
		*preq = *oreq
	}
	preq.adjustPaginationParams()

	maxPageNumber := preq.MaxPageNumber
	pageExceeds := func(page int64) bool {
		if maxPageNumber <= 0 {
			return false
		}
		return page >= maxPageNumber
	}

	pagesChan = make(chan *PhotoPage)
	cancelChan, cancelFn := makeCanceler()
	go func() {
		defer close(pagesChan)
		throttle := time.Duration(150 * time.Millisecond)

		for {
			pp := new(PhotoPage)
			qv, err := otils.ToURLValues(preq)
			if err != nil {
				pp.Err = err
				pagesChan <- pp
				return
			}
			qv.Set("consumer_key", c.consumerKey())

			fullURL := fmt.Sprintf("%s/photos?%s", baseURL, qv.Encode())
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				pp.Err = err
				pagesChan <- pp
				return
			}

			slurp, _, err := c.doAuthAndRequest(req)
			if err != nil {
				pp.Err = err
				pagesChan <- pp
				return
			}

			if err := json.Unmarshal(slurp, pp); err != nil {
				pp.Err = err
				pagesChan <- pp
				return
			}

			pp.PageNumber = preq.PageNumber

			pagesChan <- pp
			select {
			case <-cancelChan:
				return
			case <-time.After(throttle):
			}

			if pageExceeds(preq.PageNumber) {
				break
			}

			preq.PageNumber += 1
		}
	}()

	return pagesChan, cancelFn, nil
}

type Camera string

type Image struct {
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

	Images []*Image `json:"images"`

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
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	City      string `json:"city"`
	Country   string `json:"country"`

	ProfilePictureURL string `json:"userpic_url"`

	UpgradeStatus int `json:"upgrade_status"`

	FollowerCount uint64    `json:"followers_count"`
	Affection     Affection `json:"affection"`
}

type Affection int

type Sex string

const (
	SexUnspecified Sex = "0"
	SexMale        Sex = "1"
	SexFemale      Sex = "2"
)

func (s *Sex) UnmarshalJSON(b []byte) error {
	unquoted, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	switch Sex(unquoted) {
	case SexMale:
		*s = SexMale
	case SexFemale:
		*s = SexFemale
	default:
		*s = SexUnspecified
	}

	return nil
}

type Category string

const (
	CategoryUncategorized       Category = "Uncategorized"
	CategoryAbstract            Category = "Abstract"
	CategoryAnimals             Category = "Animals"
	CategoryBlackAndWhite       Category = "Black and white"
	CategoryCelebrities         Category = "Celebrities"
	CategoryCityAndArchitecture Category = "City and Architecture"
	CategoryCommercial          Category = "Commercial"
	CategoryConcert             Category = "Concert"
	CategoryFamily              Category = "Family"
	CategoryFashion             Category = "Fashion"
	CategoryFilm                Category = "Film"
	CategoryFineArt             Category = "Fine Art"
	CategoryFood                Category = "Food"
	CategoryJournalism          Category = "Journalism"
	CategoryLandscapes          Category = "Landscapes"
	CategoryMacro               Category = "Macro"
	CategoryNature              Category = "Nature"
	CategoryNude                Category = "Nude"
	CategoryPeople              Category = "People"
	CategoryPerformingArts      Category = "Performing Arts"
	CategorySport               Category = "Sport"
	CategoryStillLife           Category = "Still Life"
	CategoryStreet              Category = "Street"
	CategoryTransportation      Category = "Transportation"
	CategoryTravel              Category = "Travel"
	CategoryUnderwater          Category = "Underwater"
	CategoryUrbanExploration    Category = "Urban Exploration"
	CategoryWedding             Category = "Wedding"
)

var categoryToIntMap = map[Category]int{
	CategoryUncategorized:       0,
	CategoryAbstract:            10,
	CategoryAnimals:             11,
	CategoryBlackAndWhite:       5,
	CategoryCelebrities:         1,
	CategoryCityAndArchitecture: 9,
	CategoryCommercial:          15,
	CategoryConcert:             16,
	CategoryFamily:              20,
	CategoryFashion:             14,
	CategoryFilm:                2,
	CategoryFineArt:             24,
	CategoryFood:                23,
	CategoryJournalism:          3,
	CategoryLandscapes:          8,
	CategoryMacro:               12,
	CategoryNature:              18,
	CategoryNude:                4,
	CategoryPeople:              7,
	CategoryPerformingArts:      19,
	CategorySport:               17,
	CategoryStillLife:           6,
	CategoryStreet:              21,
	CategoryTransportation:      26,
	CategoryTravel:              13,
	CategoryUnderwater:          22,
	CategoryUrbanExploration:    27,
	CategoryWedding:             25,
}

var intToCategoryMap map[int]Category

func init() {
	intToCategoryMap = make(map[int]Category)
	for category, i := range categoryToIntMap {
		intToCategoryMap[i] = category
	}
}

func intToCategory(i int) Category {
	return intToCategoryMap[i]
}

func categoryToInt(cat Category) int {
	return categoryToIntMap[cat]
}

func (cat *Category) UnmarshalJSON(b []byte) error {
	str := string(b)
	// Firstly try as an int
	if iv, err := strconv.ParseInt(str, 10, 32); err == nil {
		// Well and good an integer was parsed
		*cat = intToCategoryMap[int(iv)]
		return nil
	}

	// Next fallback to string
	var st string
	err := json.Unmarshal(b, &st)
	if err == nil {
		*cat = Category(st)
	}

	return err
}

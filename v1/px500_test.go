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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/orijtech/500px/v1"
)

func TestListPhotos(t *testing.T) {
	client, err := px500.NewClient(consumerKey1)
	if err != nil {
		t.Fatalf("initializing the client: %v", err)
	}

	rt := &testBackend{route: listPhotosRoute}
	client.SetHTTPRoundTripper(rt)

	tests := [...]struct {
		req     *px500.PhotoRequest
		wantErr bool
		want    *px500.PhotoPage
	}{
		0: {
			req: &px500.PhotoRequest{
				Feature: px500.FeaturePopular,
			},
			want: listPhotosPageFromFile(string(px500.FeaturePopular)),
		},
		1: {req: nil, wantErr: true},
	}

	for i, tt := range tests {
		lchan, cancelFn, err := client.ListPhotos(tt.req)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d want a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: gotErr: %v", i, err)
			continue
		}

		got := <-lchan
		cancelFn()

		gotBlob := jsonMarshal(got)
		wantBlob := jsonMarshal(tt.want)
		if !bytes.Equal(gotBlob, wantBlob) {
			t.Errorf("#%d:\ngotBlob:  %s\nwantBlob: %s", i, gotBlob, wantBlob)
		}
	}
}

func TestCommentsForPhoto(t *testing.T) {
	client, err := px500.NewClient(consumerKey2)
	if err != nil {
		t.Fatalf("initializing the client: %v", err)
	}

	rt := &testBackend{route: commentsForPhotoRoute}
	client.SetHTTPRoundTripper(rt)

	tests := [...]struct {
		params  *px500.CommentsRequest
		wantErr bool
		want    *px500.CommentsPage
	}{
		0: {
			params: &px500.CommentsRequest{
				PhotoID: photoID1,
				Nested:  true,
			},
			want: commentsForPageForPhoto(photoID1, true),
		},

		1: {
			params: &px500.CommentsRequest{
				PhotoID: photoID1,
				Nested:  false,
			},
			want: commentsForPageForPhoto(photoID1, false),
		},

		2: {
			params: nil, wantErr: true,
		},

		// No PhotoID
		3: {
			params: &px500.CommentsRequest{
				PhotoID: "",
				Nested:  true,
			},
			wantErr: true,
		},

		4: {
			params: &px500.CommentsRequest{
				PhotoID: "",
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		lchan, cancelFn, err := client.CommentsForPhoto(tt.params)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d want a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: gotErr: %v", i, err)
			continue
		}

		got := <-lchan
		if got == nil {
			t.Errorf("#%d expected a non-nil page", i)
			continue
		}

		if err := got.Err; err != nil {
			t.Errorf("#%d err: %v", i, err)
			continue
		}

		cancelFn()

		gotBlob := jsonMarshal(got)
		wantBlob := jsonMarshal(tt.want)

		if !bytes.Equal(gotBlob, wantBlob) {
			t.Errorf("#%d:\ngotBlob:  %s\nwantBlob: %s", i, gotBlob, wantBlob)
		}
	}
}

func TestPhotoSearch(t *testing.T) {
	client, err := px500.NewClient(consumerKey2)
	if err != nil {
		t.Fatalf("initializing the client: %v", err)
	}

	rt := &testBackend{route: searchPhotosRoute}
	client.SetHTTPRoundTripper(rt)

	tests := [...]struct {
		req     *px500.PhotoSearch
		wantErr bool
		want    *px500.PhotoPage
	}{
		0: {
			req: &px500.PhotoSearch{
				Term:          "the universe",
				LimitPerPage:  10,
				MaxPageNumber: 2,
			},
			want: searchPhotosPageFromFile("the universe"),
		},
		1: {req: nil, wantErr: true},
	}

	for i, tt := range tests {
		lchan, cancelFn, err := client.SearchPhotos(tt.req)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d want a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: gotErr: %v", i, err)
			continue
		}

		got := <-lchan
		cancelFn()

		gotBlob := jsonMarshal(got)
		wantBlob := jsonMarshal(tt.want)

		if !bytes.Equal(gotBlob, wantBlob) {
			t.Errorf("#%d:\ngotBlob:  %s\nwantBlob: %s", i, gotBlob, wantBlob)
		}
	}
}

const (
	photoID1 = "id1"
	photoID2 = "id2"
)

func TestPhotoByID(t *testing.T) {
	client, err := px500.NewClient(consumerKey2)
	if err != nil {
		t.Fatalf("initializing the client: %v", err)
	}

	rt := &testBackend{route: photoByIDRoute}
	client.SetHTTPRoundTripper(rt)

	tests := [...]struct {
		photoID string
		wantErr bool
		want    *px500.Photo
	}{
		0: {
			photoID: photoID1,
			want:    photoFromFileByID(photoID1),
		},
		1: {
			// A random ID that's ephemeral and unknown
			photoID: fmt.Sprintf("%v", time.Now().Unix()),
			wantErr: true,
		},
		2: {
			photoID: "",
			wantErr: true,
		},
	}

	for i, tt := range tests {
		photo, err := client.PhotoByID(tt.photoID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d want a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: gotErr: %v", i, err)
			continue
		}

		if photo == nil {
			t.Errorf("#%d: photo: %#v", i, photo)
			continue
		}

		gotBlob := jsonMarshal(photo)
		wantBlob := jsonMarshal(tt.want)

		if !bytes.Equal(gotBlob, wantBlob) {
			t.Errorf("#%d:\ngotBlob:  %s\nwantBlob: %s", i, gotBlob, wantBlob)
		}
	}

}

func jsonMarshal(v interface{}) []byte {
	blob, _ := json.Marshal(v)
	return blob
}

type testBackend struct {
	route string
}

var errUnimplemented = errors.New("unimplemented")

const (
	listPhotosRoute       = "list-photos"
	searchPhotosRoute     = "search-photos"
	commentsForPhotoRoute = "comments-for-photo"
	photoByIDRoute        = "photo-by-id"

	consumerKey1 = "consumer-key-1"
	consumerKey2 = "consumer-key-2"
)

func authorizedConsumerKey(ckey string) bool {
	switch ckey {
	case consumerKey1, consumerKey2:
		return true
	default:
		return false
	}
}

func (tb *testBackend) RoundTrip(req *http.Request) (*http.Response, error) {
	// Well firstly we've got to check if they are authenticated
	query := req.URL.Query()
	if !authorizedConsumerKey(query.Get("consumer_key")) {
		return makeResp("Unauthorized consumer_key", http.StatusUnauthorized, nil), nil
	}

	switch tb.route {
	case listPhotosRoute:
		return tb.listPhotosRoundTrip(req)
	case searchPhotosRoute:
		return tb.searchPhotosRoundTrip(req)
	case commentsForPhotoRoute:
		return tb.commentsForPhotoRoundTrip(req)
	case photoByIDRoute:
		return tb.photoByIDRoundTrip(req)
	default:
		return nil, errUnimplemented
	}
}

func listPhotosPath(id string) string {
	return fmt.Sprintf("./testdata/listPhotos-%s.json", id)
}

func searchPhotosPath(term string) string {
	escapedTerm := url.QueryEscape(term)
	return fmt.Sprintf("./testdata/search-%s.json", escapedTerm)
}

func searchCommentsForPhoto(photoID string) string {
	return fmt.Sprintf("./testdata/commentsForPhoto-%s.json", photoID)
}

func searchPhotosPageFromFile(term string) *px500.PhotoPage {
	path := searchPhotosPath(term)
	return photoPageFromFile(path)
}

func listPhotosPageFromFile(id string) *px500.PhotoPage {
	path := listPhotosPath(id)
	return photoPageFromFile(path)
}

func commentsForPageForPhotoPath(photoID string, nested bool) string {
	if nested {
		photoID += "-nested"
	}
	return fmt.Sprintf("./testdata/commentsForPhoto-%s.json", photoID)
}

func commentsForPageForPhoto(photoID string, nested bool) *px500.CommentsPage {
	path := commentsForPageForPhotoPath(photoID, nested)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	sav := new(px500.CommentsPage)
	if err := json.Unmarshal(data, sav); err != nil {
		return nil
	}
	return sav

}

func photoPageFromFile(path string) *px500.PhotoPage {
	sav := new(px500.PhotoPage)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(data, sav); err != nil {
		return nil
	}
	return sav
}

func (tb *testBackend) listPhotosRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		msg := fmt.Sprintf("only accepting \"GET\" not %q", req.Method)
		return makeResp(msg, http.StatusMethodNotAllowed, http.NoBody), nil
	}

	query := req.URL.Query()
	featStr := query.Get("feature")
	path := listPhotosPath(featStr)

	f, err := os.Open(path)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, http.NoBody), nil
	}

	return makeResp("200 OK", http.StatusOK, f), nil
}

func (tb *testBackend) photoByIDRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		msg := fmt.Sprintf("only accepting \"GET\" not %q", req.Method)
		return makeResp(msg, http.StatusMethodNotAllowed, http.NoBody), nil
	}

	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 2 {
		return makeResp("expecting the id", http.StatusBadRequest, http.NoBody), nil
	}
	id := splits[len(splits)-1]

	diskPath := photoByIDPath(id)
	f, err := os.Open(diskPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, http.NoBody), nil
	}
	return makeResp("200 OK", http.StatusOK, f), nil
}

func photoByIDPath(id string) string {
	return fmt.Sprintf("./testdata/photo-by-id-%s.json", id)
}

func photoFromFileByID(id string) *px500.Photo {
	path := photoByIDPath(id)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	pwrap := new(px500.PhotoWrap)
	if err := json.Unmarshal(data, pwrap); err != nil {
		return nil
	}
	return pwrap.Photo
}

func (tb *testBackend) searchPhotosRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		msg := fmt.Sprintf("only accepting \"GET\" not %q", req.Method)
		return makeResp(msg, http.StatusMethodNotAllowed, http.NoBody), nil
	}

	query := req.URL.Query()
	term := query.Get("term")
	path := searchPhotosPath(term)

	f, err := os.Open(path)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, http.NoBody), nil
	}

	return makeResp("200 OK", http.StatusOK, f), nil
}

func (tb *testBackend) commentsForPhotoRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		msg := fmt.Sprintf("only accepting \"GET\" not %q", req.Method)
		return makeResp(msg, http.StatusMethodNotAllowed, http.NoBody), nil
	}

	// Expecting the form:
	//    /v1/photos/210717663/comments
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 2 {
		return makeResp("expecting the photoId", http.StatusBadRequest, http.NoBody), nil
	}

	photoID := splits[len(splits)-2]

	query := req.URL.Query()
	nested, _ := strconv.ParseBool(query.Get("nested"))
	path := commentsForPageForPhotoPath(photoID, nested)

	f, err := os.Open(path)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, http.NoBody), nil
	}

	return makeResp("200 OK", http.StatusOK, f), nil

}

func makeResp(status string, code int, body io.ReadCloser) *http.Response {
	return &http.Response{
		Status:     status,
		StatusCode: code,
		Body:       body,
		Header:     make(http.Header),
	}
}

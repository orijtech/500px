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
	"os"
	"testing"

	"github.com/orijtech/500px/v1"
)

func TestListPhotos(t *testing.T) {
	client, err := px500.NewClient()
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

func jsonMarshal(v interface{}) []byte {
	blob, _ := json.Marshal(v)
	return blob
}

type testBackend struct {
	route string
}

var errUnimplemented = errors.New("unimplemented")

const (
	listPhotosRoute = "list-photos"
)

func (tb *testBackend) RoundTrip(req *http.Request) (*http.Response, error) {
	switch tb.route {
	case listPhotosRoute:
		return tb.listPhotosRoundTrip(req)
	default:
		return nil, errUnimplemented
	}
}

func listPhotosPath(id string) string {
	return fmt.Sprintf("./testdata/listPhotos-%s.json", id)
}

func listPhotosPageFromFile(id string) *px500.PhotoPage {
	sav := new(px500.PhotoPage)
	data, err := ioutil.ReadFile(listPhotosPath(id))
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

func makeResp(status string, code int, body io.ReadCloser) *http.Response {
	return &http.Response{
		Status:     status,
		StatusCode: code,
		Body:       body,
		Header:     make(http.Header),
	}
}

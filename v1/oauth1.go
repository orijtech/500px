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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dghubble/oauth1"
)

var oauth1Endpoint = oauth1.Endpoint{
	RequestTokenURL: fmt.Sprintf("%s/oauth/request_token", baseURL),
	AccessTokenURL:  fmt.Sprintf("%s/oauth/access_token", baseURL),
	AuthorizeURL:    fmt.Sprintf("%s/oauth/authorize", baseURL),
}

type OAuth1Info struct {
	ConsumerToken  string `json:"consumer_token"`
	ConsumerSecret string `json:"consumer_secret"`
	AccessToken    string `json:"access_token"`
	AccessSecret   string `json:"access_secret"`
	CallbackURL    string `json:"callback_url"`
}

const (
	envConsumerSecretKey = "PX500_CONSUMER_SECRET"
	envConsumerKeyKey    = "PX500_CONSUMER_KEY"
	envAccessSecretKey   = "PX500_ACCESS_SECRET"
	envAccessTokenKey    = "PX500_ACCESS_TOKEN"
)

func OAuth1ConsumerInfoFromEnv() (*OAuth1Info, error) {
	secret := os.Getenv(envConsumerSecretKey)
	key := os.Getenv(envConsumerKeyKey)
	var errsList []string
	if secret == "" {
		errsList = append(errsList, fmt.Sprintf("%q was not set", envConsumerSecretKey))
	}
	if key == "" {
		errsList = append(errsList, fmt.Sprintf("%q was not set", envConsumerKeyKey))
	}
	if len(errsList) > 0 {
		errsList = append(errsList, "in your environment")
		return nil, errors.New(strings.Join(errsList, "\n"))
	}

	return &OAuth1Info{ConsumerSecret: secret, ConsumerToken: key}, nil
}

func OAuth1AccessInfoFromEnv() (*OAuth1Info, error) {
	accessSecret := os.Getenv(envAccessSecretKey)
	accessToken := os.Getenv(envAccessTokenKey)
	var errsList []string
	if accessSecret == "" {
		errsList = append(errsList, fmt.Sprintf("%q was not set", envAccessSecretKey))
	}
	if accessToken == "" {
		errsList = append(errsList, fmt.Sprintf("%q was not set", envAccessTokenKey))
	}
	if len(errsList) > 0 {
		errsList = append(errsList, "in your environment")
		return nil, errors.New(strings.Join(errsList, "\n"))
	}

	return &OAuth1Info{AccessSecret: accessSecret, AccessToken: accessToken}, nil
}

func OAuth1AuthorizationByEnv() (*oauth1.Token, error) {
	ainfo, err := OAuth1ConsumerInfoFromEnv()
	if err != nil {
		return nil, err
	}
	return OAuth1Authorization(ainfo)
}

type verifierTokenPair struct {
	requestToken string
	verifier     string
}

func (info *OAuth1Info) toOAuth1Token() *oauth1.Token {
	return &oauth1.Token{
		Token:  info.AccessToken,
		TokenSecret: info.AccessSecret,
	}
}

func (info *OAuth1Info) toOAuth1Config() *oauth1.Config {
	return &oauth1.Config{
		ConsumerKey:    info.ConsumerToken,
		ConsumerSecret: info.ConsumerSecret,
		Endpoint:       oauth1Endpoint,
		CallbackURL:    info.CallbackURL,
	}

}

func OAuth1Authorization(info *OAuth1Info) (*oauth1.Token, error) {
	tokenServer := &http.Server{Addr: ":9999"}
	callbackURL := info.CallbackURL
	if callbackURL == "" {
		callbackURL = fmt.Sprintf("http://localhost%s/", tokenServer.Addr)
	}

	config := info.toOAuth1Config()
	config.CallbackURL = callbackURL

	requestToken, requestSecret, err := config.RequestToken()
	if err != nil {
		return nil, err
	}
	log.Printf("requestToken: %q requestSecret: %q\n", requestToken, requestSecret)

	authorizationURL, err := config.AuthorizationURL(requestToken)
	if err != nil {
		return nil, err
	}
	log.Printf("To authorize access, visit:\n%s\n", authorizationURL)

	recvChan := make(chan *verifierTokenPair, 1)
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		requestToken, verifier, err := oauth1.ParseAuthorizationCallback(req)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		recvChan <- &verifierTokenPair{requestToken: requestToken, verifier: verifier}
		fmt.Fprintf(rw, "Got a response")
	})

	go func() {
		if err := tokenServer.ListenAndServe(); err != nil {
			log.Fatalf("serving http err: %v", err)
		}
	}()

	vtPair := <-recvChan
	requestToken, verifier := vtPair.requestToken, vtPair.verifier
	accessToken, accessSecret, err := config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		return nil, err
	}

	return oauth1.NewToken(accessToken, accessSecret), nil
}

type tokenSource struct {
	token *oauth1.Token
}

var _ oauth1.TokenSource = (*tokenSource)(nil)

func (ts *tokenSource) Token() (*oauth1.Token, error) {
	return ts.token, nil
}

func NewOAuth1ClientFromEnv() (*Client, error) {
	var errsList []string
	consumerInfo, err := OAuth1ConsumerInfoFromEnv()
	if err != nil {
		errsList = append(errsList, err.Error())
	}

	accessInfo, err := OAuth1AccessInfoFromEnv()
	if err != nil {
		errsList = append(errsList, err.Error())
	}

	if len(errsList) > 0 {
		return nil, errors.New(strings.Join(errsList, "\n"))
	}

	return NewOAuth1Client(&OAuth1Info{
		AccessToken:  accessInfo.AccessToken,
		AccessSecret: accessInfo.AccessSecret,

		ConsumerSecret: consumerInfo.ConsumerSecret,
		ConsumerToken:  consumerInfo.ConsumerToken,
	})
}

func NewOAuth1Client(oinfo *OAuth1Info) (*Client, error) {
	config := oinfo.toOAuth1Config()
	token := oinfo.toOAuth1Token()
	oauthClient := oauth1.NewClient(context.Background(), config, token)
	client := new(Client)
	client.rt = oauthClient.Transport
	return client, nil
}

func OAuth1TokenFromFile(path string) (*oauth1.Token, error) {
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	token := new(oauth1.Token)
	if err := json.Unmarshal(blob, token); err != nil {
		return nil, err
	}
	return token, nil
}

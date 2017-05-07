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
	"time"
)

type Profile struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Sex       string `json:"sex"`
	City      string `json:"city"`
	State     string `json:"state"`
	Country   string `json:"country"`

	RegistrationDate *time.Time `json:"registration_date"`

	// About is the user's about text.
	About string `json:"about"`

	// Domain is the host name of the user's portfolio.
	Domain string `json:"domain"`

	Locale string `json:"locale"`

	UpgradeStatus int `json:"upgrade_status"`

	// Whether the user has content filter disabled
	ContentFilteringDisabled bool `json:"show_nude"`

	ProfilePictureURL string `json:"userpic_url"`

	StoreEnabled bool `json:"store_on"`

	Contacts map[string]string

	Equipment map[string][]string `json:"equipment"`

	ActivePhotoCount      uint64 `json:"photos_count"`
	VisibleGalleriesCount uint64 `json:"galleries_count"`

	FriendCount   uint64 `json:"friends_count"`
	FollowerCount uint64 `json:"followers_count"`

	Admin bool `json:"admin"`

	Avatars map[string]string `json:"avatars"`

	Email                 string     `json:"email"`
	UploadLimit           uint64     `json:"upload_limit"`
	UploadLimitExpiryDate *time.Time `json:"upload_limit_expiry"`

	UpgradeExpiryDate *time.Time `json:"upgrade_expiry_date"`

	Auth map[string]string `json:"auth"`

	Following bool `json:"following"`
}

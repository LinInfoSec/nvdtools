// Copyright (c) LINAGORA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
)

var DB_DSN string
var DATA_DIR string
var NOTIFICATION_ENDPOINT string

func LoadConfig() {
	DB_DSN = os.Getenv("LININFOSEC_MYSQL_DSN")
	DATA_DIR = os.Getenv("LININFOSEC_DATA_DIR")
	NOTIFICATION_ENDPOINT = os.Getenv("LININFOSEC_NOTIFICATION_ENDPOINT")
	if DB_DSN == "" {
		panic("No database configured")
	}
	if DATA_DIR == "" {
		panic("No data directory configured")
	}
}

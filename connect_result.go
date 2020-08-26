// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fanmerge

import (
	"time"
	"math/rand"

	"github.com/miekg/dns"
)

type response struct {
	client   Client
	response *dns.Msg
	start    time.Time
	err      error
}

func mergeResponse(left, right *response) *response {
	if right == nil {
		return left
	}
	if left == nil {
		return right
	}
	if right.err != nil {
		return left
	}
	if left.err != nil {
		return nil
	}
	if right.response == nil {
		return left
	}
	if left.response == nil {
		return right
	}
	if left.response.MsgHdr.Rcode == dns.RcodeSuccess &&
		right.response.MsgHdr.Rcode == dns.RcodeSuccess {
			// We want to randomly pick how to merge this
			if rand.Intn(1000) % 2 == 1 {
				log.Info("append right to left")
				left.response.Answer = append(left.response.Answer,right.response.Answer...)
			} else {
				log.Info("append left to right")
				left.response.Answer = append(right.response.Answer,left.response.Answer...)
			}
			return left
	}
	return nil
}

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
		return pruneCNames(left)
	}
	if left == nil {
		return pruneCNames(right)
	}
	if right.err != nil {
		return pruneCNames(left)
	}
	if left.err != nil {
		return nil
	}
	if right.response == nil {
		return pruneCNames(left)
	}
	if left.response == nil {
		return pruneCNames(right)
	}
	if left.response.MsgHdr.Rcode == dns.RcodeSuccess &&
		right.response.MsgHdr.Rcode == dns.RcodeSuccess {
			// We want to randomly pick how to merge this
			left.response.Answer = append(left.response.Answer,right.response.Answer...)
			// rand.Shuffle(len(left.response.Answer), func(i, j int) { left.response.Answer[i], left.response.Answer[j] = left.response.Answer[j], left.response.Answer[i]})
			return pruneCNames(left)
	}
	return nil
}

func pruneCNames(in *response) *response {
	if in == nil || in.response == nil {
		return in
	}
	cname := []dns.RR{}
	address := []dns.RR{}
	mx := []dns.RR{}
	rest := []dns.RR{}

	for _, r := range in.response.Answer {
		switch r.Header().Rrtype {
		case dns.TypeCNAME:
			cname = append(cname, r)
		case dns.TypeA, dns.TypeAAAA:
			address = append(address, r)
		case dns.TypeMX:
			mx = append(mx, r)
		default:
			rest = append(rest, r)
		}
	}

	out := append(mx, address...)
	in.response.Answer = out
	return in
}

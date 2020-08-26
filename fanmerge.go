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
	"context"
	"crypto/tls"
	"errors"
	"time"
	"math/rand"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/debug"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("fanmerge")

// Fanmerge represents a plugin instance that can do async requests to list of DNS servers.
type Fanmerge struct {
	clients        []Client
	tlsConfig      *tls.Config
	excludeDomains Domain
	tlsServerName  string
	timeout        time.Duration
	net            string
	from           string
	attempts       int
	workerCount    int
	Next           plugin.Handler
}

// New returns reference to new Fanmerge plugin instance with default configs.
func New() *Fanmerge {
	// Seed the random generator to produce random results in the output
	rand.Seed(time.Now().UnixNano())
	return &Fanmerge{
		tlsConfig:      new(tls.Config),
		net:            "udp",
		attempts:       3,
		timeout:        defaultTimeout,
		excludeDomains: NewDomain(),
	}
}

func (f *Fanmerge) addClient(p Client) {
	f.clients = append(f.clients, p)
	f.workerCount++
}

// Name implements plugin.Handler.
func (f *Fanmerge) Name() string {
	return "fanmerge"
}

// ServeDNS implements plugin.Handler.
func (f *Fanmerge) ServeDNS(ctx context.Context, w dns.ResponseWriter, m *dns.Msg) (int, error) {
	req := request.Request{W: w, Req: m}
	if !f.match(&req) {
		return plugin.NextOrFailure(f.Name(), f.Next, ctx, w, m)
	}
	timeoutContext, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	clientCount := len(f.clients)
	workerChannel := make(chan Client, f.workerCount)
	defer close(workerChannel)
	responseCh := make(chan *response, clientCount)
	go func() {
		for i := 0; i < clientCount; i++ {
			client := f.clients[i]
			workerChannel <- client
		}
	}()
	for i := 0; i < f.workerCount; i++ {
		go func() {
			for c := range workerChannel {
				responseCh <- f.processClient(timeoutContext, c, &request.Request{W: w, Req: m})
			}
		}()
	}
	result := f.getFanmergeResult(timeoutContext, responseCh)
	if result == nil {
		return dns.RcodeServerFailure, timeoutContext.Err()
	}
	if result.err != nil {
		return dns.RcodeServerFailure, result.err
	}
	dnsTAP := toDnstap(ctx, result.client.Endpoint(), f.net, &req, result.response, result.start)
	if !req.Match(result.response) {
		debug.Hexdumpf(result.response, "Wrong reply for id: %d, %s %d", result.response.Id, req.QName(), req.QType())
		formerr := new(dns.Msg)
		formerr.SetRcode(req.Req, dns.RcodeFormatError)
		logErrIfNotNil(w.WriteMsg(formerr))
		return 0, dnsTAP
	}
	logErrIfNotNil(w.WriteMsg(result.response))
	return 0, dnsTAP
}

func (f *Fanmerge) getFanmergeResult(ctx context.Context, responseCh <-chan *response) *response {
	count := len(f.clients)
	var result *response
	for {
		select {
		case <-ctx.Done():
			return result
		case r := <-responseCh:
			count--
			result = mergeResponse(result, r)
			if count == 0 {
				return result
			}
			break
		}
	}
}

func (f *Fanmerge) match(state *request.Request) bool {
	if !plugin.Name(f.from).Matches(state.Name()) || f.excludeDomains.Contains(state.Name()) {
		return false
	}
	return true
}

func (f *Fanmerge) processClient(ctx context.Context, c Client, r *request.Request) *response {
	start := time.Now()
	for j := 0; j < f.attempts || f.attempts == 0; <-time.After(attemptDelay) {
		if ctx.Err() != nil {
			return &response{client: c, response: nil, start: start, err: ctx.Err()}
		}
		msg, err := c.Request(ctx, r)
		if err == nil {
			return &response{client: c, response: msg, start: start, err: err}
		}
		if f.attempts != 0 {
			j++
		}
	}
	return &response{client: c, response: nil, start: start, err: errors.New("attempt limit has been reached")}
}

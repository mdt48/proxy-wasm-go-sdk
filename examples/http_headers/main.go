// Copyright 2020-2021 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	// "net/http"

	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

var authHeader string

func main() {
	// proxywasm.SetTickPeriodMilliseconds(1000)
	proxywasm.SetVMContext(&vmContext{})
}

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext

	// headerName and headerValue are the header to be added to response. They are configured via
	// plugin configuration during OnPluginStart.
	headerName  string
	headerValue string

	callBack func(numHeaders, bodySize, numTrailers int)
}

// Override types.DefaultPluginContext.
func (p *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpHeaders{
		contextID:   contextID,
		headerName:  p.headerName,
		headerValue: p.headerValue,
		callBack:    p.callBack,
	}
}

func (p *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	// proxywasm.LogDebug("loading plugin config")
	data, err := proxywasm.GetPluginConfiguration()
	if data == nil {
		return types.OnPluginStartStatusOK
	}

	if err != nil {
		// proxywasm.LogCriticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}

	if !gjson.Valid(string(data)) {
		// proxywasm.LogCritical(`invalid configuration format; expected {"header": "<header name>", "value": "<header value>"}`)
		return types.OnPluginStartStatusFailed
	}

	p.headerName = strings.TrimSpace(gjson.Get(string(data), "header").Str)
	p.headerValue = strings.TrimSpace(gjson.Get(string(data), "value").Str)

	if p.headerName == "" || p.headerValue == "" {
		proxywasm.LogCritical(`invalid configuration format; expected {"header": "<header name>", "value": "<header value>"}`)
		return types.OnPluginStartStatusFailed
	}
	p.callBack = func(numHeaders, bodySize, numTrailers int) {
		respHeaders, _ := proxywasm.GetHttpCallResponseHeaders()
		proxywasm.LogInfof("respHeaders: %v", respHeaders)

		for _, headerPairs := range respHeaders {
			if headerPairs[0] == "authorization" {
				authHeader = headerPairs[1]
			}
		}
	}
	// proxywasm.LogInfof("header from config: %s = %s", p.headerName, p.headerValue)

	return types.OnPluginStartStatusOK
}

type httpHeaders struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID   uint32
	headerName  string
	headerValue string
	callBack    func(numHeaders, bodySize, numTrailers int)
}

type node struct {
	timeStamp int64
	power     float64
}

type dataStore struct {
	reqID2Info map[string]node
	currID     string
	counter    int
}

var ds = dataStore{
	currID:     "",
	reqID2Info: make(map[string]node),
	counter:    0,
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	// get request id from http request header
	reqID, err := proxywasm.GetHttpRequestHeader("x-request-id")
	if err != nil {
		proxywasm.LogCriticalf("failed to get request id: %v", err)
	}

	currTime := time.Now().UnixNano() / 1000000000

	ds.currID = reqID
	ds.reqID2Info[reqID] = node{
		timeStamp: currTime,
		power:     0,
	}

	// proxywasm.LogInfof("IN THE REQUEST FOR OUTGOING")

	// direction, err := proxywasm.GetHttpRequestHeader(":method")

	// if err != nil {
	// 	proxywasm.LogCriticalf("\n\nfailed to get request Method so it is outgoing: %v", err)
	// } else {
	// 	// log direction
	// 	proxywasm.LogInfof("\n\ndirection: %s", direction)
	// }
	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpResponseHeaders(_ int, _ bool) types.Action {
	currID := ds.currID
	currNode := ds.reqID2Info[currID]
	timeDelta := time.Now().UnixNano()/1000000000 - currNode.timeStamp
	rand.Seed(time.Now().UnixNano())
	// concatenate the string "x-power" with curID
	// powerKey := "x-power-" + currID[:5]
	powerKey := ds.counter
	power := rand.Intn(100)
	powerKey = power + powerKey
	ds.counter++

	powerKeyString := strconv.Itoa(powerKey)

	// // convert the integer power to a string
	powerString := strconv.Itoa(power)

	// init_reqid, err := proxywasm.GetHttpRequestHeader("x-request-id")

	// log powerkey and powerkeystring
	proxywasm.LogInfof("\n\n\tpower: %d\n\tpowerKeyString: %s", power, powerKeyString)

	// power, err = proxGetHttpResponseHeader(powerKey)

	// log the powerKey response header
	proxywasm.LogInfof("\n\n\tcurrent id: %s\n\tcurrent node: %v\n\ttime delta: %d", currID, currNode, timeDelta)

	// for k, v := range ds.reqID2Info {
	// 	// fmt.Printf("key[%s] value[%s]\n", k, v)
	// 	proxywasm.LogInfof("\nkey[%s] value[%s]\n", k, v.power)
	// }

	// headers, err := ctx.GetHttpResponseHeaders()
	// if err != nil {
	// 	proxywasm.LogWarnf("Failed to get response headers: %v", err)
	// 	return types.ActionContinue
	// }

	// // Print all of the response headers.
	// for _, header := range headers {
	// 	proxywasm.LogInfof("Response header: %s: %s", header[0], header[1])
	// }

	// resp, err := http.Get("127.0.0.1:8020/fib")
	// if err != nil {
	// 	proxywasm.LogCriticalf("error %s", err)
	// }
	// hs := [][2]string{
	// 	{":method", "GET"}, {":authority", "127.0.0.1:8000"}, {":path", "/fib"}, {"accept", "*/*"},
	// }

	// _, err := proxywasm.DispatchHttpCall("fib_ingress", hs, nil, nil, 5000, ctx.callBack)

	// if err != nil {
	// 	proxywasm.LogCriticalf("error %s", err)
	// }

	// // proxywasm.LogInfof("\n\nresponse: %s", resp.body)

	// // proxywasm.LogInfof("response: %s", resp)

	// // proxywasm.LogInfof("headerName = %s, header	Value = %s", ctx.headerName, ctx.headerValue)
	// proxywasm.LogInfof("IN THE RESPONSE FOR OUTGOING")
	// proxywasm.AddHttpResponseHeader("x-TEST", authHeader)
	// proxywasm.LogInfof("\n\n%s", authHeader)

	// proxywasm.
	// Get and log the headers
	hs, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get response headers: %v", err)
	}

	for _, h := range hs {
		// if strings.Contains(h[0], "x-power") {
		proxywasm.LogInfof("response header <-- %s: %s\n\n", h[0], h[1])
		// proxywasm.AddHttpResponseHeader(h[0], h[1])
		// }

	}

	proxywasm.AddHttpResponseHeader("x-power-"+""+powerKeyString, powerString)

	hs, err = proxywasm.GetHttpResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get response headers: %v", err)
	}

	for _, h := range hs {
		if strings.Contains(h[0], "x-power") {
			proxywasm.LogInfof("response header <-- %s: %s", h[0], h[1])
			// proxywasm.AddHttpResponseHeader(h[0], h[1])
		}

	}

	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpStreamDone() {
	proxywasm.LogInfof("%d finished", ctx.headerName)
}

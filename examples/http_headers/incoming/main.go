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
	"strings"

	"github.com/tidwall/gjson"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

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
}

// Override types.DefaultPluginContext.
func (p *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpHeaders{
		contextID:   contextID,
		headerName:  p.headerName,
		headerValue: p.headerValue,
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
}

// type node struct {
// 	timeStamp int64
// 	power     float64
// }

// type dataStore struct {
// 	reqID2Info map[string]node
// 	currID     string
// }

// var ds = dataStore{
// 	currID:     "",
// 	reqID2Info: make(map[string]node),
// }

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	// get request id from http request header
	// reqID, err := proxywasm.GetHttpRequestHeader("x-request-id")
	// if err != nil {
	// 	proxywasm.LogCriticalf("failed to get request id: %v", err)
	// }

	// currTime := time.Now().UnixNano() / 1000000000

	// ds.currID = reqID
	// ds.reqID2Info[reqID] = node{
	// 	timeStamp: currTime,
	// 	power:     0,
	// }
	proxywasm.LogInfof("IN THE REQUEST FOR INCOMING")

	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpResponseHeaders(_ int, _ bool) types.Action {
	// currID := ds.currID
	// currNode := ds.reqID2Info[currID]
	// timeDelta := time.Now().UnixNano()/1000000000 - currNode.timeStamp

	// concatenate the string "x-power" with curID
	// powerKey := "x-power-" + currID[:5]

	// // create a random variable between 0 and 100
	// power := rand.Intn(100)

	// // convert the integer power to a string
	// powerString := strconv.Itoa(power)

	// add random power value to response header with key "x-power"
	// err := proxywasm.AddHttpResponseHeader(powerKey, powerString)
	// if err != nil {
	// 	proxywasm.LogCriticalf("failed to add response header: %v", err)
	// }
	// power, err = GetHttpResponseHeader(powerKey)

	// log the powerKey response header
	// proxywasm.LogInfof("\n\n\tcurrent id: %s\n\tcurrent node: %v\n\ttime delta: %d", currID, currNode, timeDelta)

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

	// proxywasm.LogInfof("headerName = %s, headerValue = %s", ctx.headerName, ctx.headerValue)
	proxywasm.LogInfof("IN THE RESPONSE FOR")
	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpStreamDone() {
	proxywasm.LogInfof("%d finished", ctx.contextID)
}

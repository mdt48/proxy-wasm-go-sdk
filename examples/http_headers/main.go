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
	"time"

	"github.com/tidwall/gjson"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

// var dataStore *proxywasm.ThreadLocalStore
type dataStore struct {
	requestID   string
	requestTime int64
}

var ds = dataStore{
	requestID:   "",
	requestTime: 0,
}

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

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	requestID, err := proxywasm.GetHttpRequestHeader("x-request-id")
	if err != nil {
		proxywasm.LogCriticalf("failed to get request header: %v", err)
	}
	proxywasm.LogInfof("\n\nrequest id: %d\n\n", requestID)

	//requestID to string
	// strUUID := string(requestID[:])
	// b := []byte(requestID)
	// err = proxywasm.SetSharedData("requestID", b, 0)
	// if err != nil {
	// 	proxywasm.LogCriticalf("failed to set shared data: %v", err)
	// }
	ds.requestID = requestID

	// t := proxywasm.GetCurrentTimeNanoseconds()
	t := time.Now().UnixNano() / 1000000000

	ds.requestTime = t
	// err = proxywasm.SetSharedData("requestTime", []byte(strconv.FormatInt(t, 10)), 0)
	// if err != nil {
	// 	proxywasm.LogCriticalf("failed to set shared data: %v", err)
	// }
	proxywasm.LogInfof("\n\nrequest ID: %s\n\n", ds.requestID)
	proxywasm.LogInfof("\n\nrequest time: %d\n\n", ds.requestTime)

	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpResponseHeaders(_ int, _ bool) types.Action {
	proxywasm.LogInfof("adding header: %s=%s", ctx.headerName, ctx.headerValue)

	// Add a hardcoded header
	if err := proxywasm.AddHttpResponseHeader("x-another-test", "TESTHAHAHA"); err != nil {
		proxywasm.LogCriticalf("failed to set response constant header: %v", err)
	}

	// Add the header passed by arguments
	if ctx.headerName != "" {
		if err := proxywasm.AddHttpResponseHeader(ctx.headerName, ctx.headerValue); err != nil {
			proxywasm.LogCriticalf("failed to set response headers: %v", err)
		}
	}

	responseTime := time.Now().UnixNano() / 1000000000

	reqTime := ds.requestTime
	reqID := ds.requestID

	// log requestID, requestTime, responseTime and responseTime - requestTime
	proxywasm.LogInfof("\n\nrequest ID: %s\n\n", reqID)
	proxywasm.LogInfof("\n\nrequest time: %d\n\n", reqTime)
	proxywasm.LogInfof("\n\nresponse time: %d\n\n", responseTime)
	proxywasm.LogInfof("\n\ndiff: %d\n\n", responseTime-reqTime)

	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpStreamDone() {
	proxywasm.LogInfof("%d finished", ctx.contextID)
}

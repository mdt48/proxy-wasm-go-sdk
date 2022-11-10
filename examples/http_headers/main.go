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
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"strings"
	"encoding/base64"
	// "github.com/golang-jwt/jwt/v4"
)
const clusterName = "web_service"

func main() {
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
}

// Override types.DefaultPluginContext.
func (*pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpHeaders{contextID: contextID}
}

type httpHeaders struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID uint32
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
	}

	for _, h := range hs {
		proxywasm.LogInfof("request header --> %s: %s", h[0], h[1])
		if h[0] == "cookie" {
			// cookie was set with JWT 
			if h[1] != "" {
				proxywasm.LogInfof("going to authenticate this request")
				token := strings.SplitAfterN(h[1], "token=", 2)
				proxywasm.LogInfof("this is our extracted token: %s", token[1])

				// decode token header to extract alg
				header_payload := strings.Split(token[1], ".")
				header, err := base64.StdEncoding.DecodeString(header_payload[0])
				if err != nil {
					proxywasm.LogInfof("some err thrown when decoding header")
				}
				alg := strings.Split(string(header), "{\"typ\":\"JWT\",\"alg\":\"")
				
				if alg[1] == "none" {
					if _, err := proxywasm.DispatchHttpCall(clusterName, hs, nil, nil,
						50000, httpCallResponseCallback); err != nil {
						proxywasm.LogCriticalf("dipatch httpcall failed: %v", err)
						return types.ActionContinue
					}
				}
			}
		}
	}
	
	path, err := proxywasm.GetHttpRequestHeader("path")
	if err != nil { // check if path field has been set yet (i.e. are we the first on the path?)
		err := proxywasm.ReplaceHttpRequestHeader("path", "IP here") // TODO: extract IP to set
		if err != nil {
			proxywasm.LogCritical("failed to set request header: path")
		}
		proxywasm.LogInfof("request header --> path: IP here")
	} else { // TODO: extract path and append new IP to it
		proxywasm.LogInfof("should not be here for now: %s", path)
	}

	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpResponseHeaders(numHeaders int, endOfStream bool) types.Action {
	hs, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get response headers: %v", err)
	}

	for _, h := range hs {
		proxywasm.LogInfof("response header <-- %s: %s", h[0], h[1])
	}
	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpStreamDone() {
	proxywasm.LogInfof("%d finished", ctx.contextID)
}

func httpCallResponseCallback(numHeaders, bodySize, numTrailers int) {
	
	body := "access forbidden: JWT has alg none!!"
	proxywasm.LogInfo(body)
	if err := proxywasm.SendHttpResponse(403, [][2]string{
		{"powered-by", "proxy-wasm-go-sdk!!"},
	}, []byte(body), -1); err != nil {
		proxywasm.LogErrorf("failed to send local response: %v", err)
		proxywasm.ResumeHttpRequest()
	}
		
}
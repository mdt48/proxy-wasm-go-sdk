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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

const tickMilliseconds uint32 = 1000

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
func (ctx *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	if err := proxywasm.SetTickPeriodMilliSeconds(tickMilliseconds); err != nil {
		proxywasm.LogCriticalf("failed to set tick period: %v", err)
		return types.OnPluginStartStatusFailed
	}
	proxywasm.LogInfof("set tick period milliseconds: %d", tickMilliseconds)
	return types.OnPluginStartStatusOK
}

// type httpHeaders struct {
// 	// Embed the default http context here,
// 	// so that we don't need to reimplement all the methods.
// 	types.DefaultHttpContext
// 	contextID   uint32
// 	headerName  string
// 	headerValue string
// 	cnt         int
// 	// callBack    func(numHeaders, bodySize, numTrailers int)
// 	plugin      pluginContext

// 	// cnt       int
// }

// Override types.DefaultPluginContext.
func (*pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpContext{contextID: contextID}
}

type httpContext struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	// contextID is the unique identifier assigned to each httpContext.
	contextID uint32
	// pendingDispatchedRequest is the number of pending dispatched requests.
	pendingDispatchedRequest int
}

type node struct {
	timeStamp int64
	power     float64
	path      string
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

type rps_struct struct {
	requests int
	rps      int
}

// const quota = 100

var RPS = rps_struct{requests: 0, rps: 100}

func (ctx *httpContext) OnHttpRequestHeaders(_ int, _ bool) types.Action {
	reqID, err := proxywasm.GetHttpRequestHeader("x-request-id")
	if err != nil {
		proxywasm.LogCriticalf("failed to get request id: %v", err)
	}

	path, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		proxywasm.LogCriticalf("failed to get path: %v", err)
	}

	total_energy, err := proxywasm.GetHttpRequestHeader("total_energy")
	if err != nil {
		proxywasm.LogCriticalf("failed to get path: %v", err)
	}

	quota, err := proxywasm.GetHttpRequestHeader("quota")
	if err != nil {
		proxywasm.LogCriticalf("failed to get path: %v", err)
	}

	total_energy_float, err := strconv.ParseFloat(total_energy, 64)
	if err != nil {
		proxywasm.LogCriticalf("failed to convert total energy to float: %v", err)
	}
	quota_float, err := strconv.ParseFloat(quota, 64)
	if err != nil {
		proxywasm.LogCriticalf("failed to convert quota to float: %v", err)
	}

	proxywasm.LogCriticalf("\n\nquota=%f\n\n", quota_float)
	proxywasm.LogCriticalf("\n\ntotal energy: %f\n\n", total_energy_float)
	proxywasm.LogCriticalf("\n\nratio: %f\n\n", total_energy_float/quota_float)

	// log boolean value
	proxywasm.LogCriticalf("\n\nboolean: %t\n\n", total_energy_float/quota_float > 0.7)
	if total_energy_float/quota_float > 0.7 {
		proxywasm.LogCriticalf("Routing to lower energy server")

		const authorityKey = ":authority"
		value, err := proxywasm.GetHttpRequestHeader(authorityKey)
		if err != nil {
			proxywasm.LogCritical("failed to get request header: ':authority'")
			return types.ActionPause
		}

		path, err = proxywasm.GetHttpRequestHeader(":path")
		if err != nil {
			proxywasm.LogCriticalf("failed to get path: %v", err)
		}

		replaced_path := strings.Replace(path, "resnet152", "efficientnetb0", 1)

		value += "-save-energy"
		if err := proxywasm.ReplaceHttpRequestHeader(":authority", value); err != nil {
			proxywasm.LogCritical("failed to set request header: test")
			return types.ActionPause
		}

		if err := proxywasm.ReplaceHttpRequestHeader(":path", replaced_path); err != nil {
			proxywasm.LogCritical("failed to set request header: test")
			return types.ActionPause
		}

		currTime := time.Now().UnixNano() / 1000000000

		ds.currID = reqID
		ds.reqID2Info[reqID] = node{
			timeStamp: currTime,
			power:     0,
			path:      replaced_path,
		}

		RPS.requests++
	} else {

		currTime := time.Now().UnixNano() / 1000000000

		ds.currID = reqID
		ds.reqID2Info[reqID] = node{
			timeStamp: currTime,
			power:     0,
			path:      path,
		}

		RPS.requests++

	}

	return types.ActionContinue
}

func (ctx *httpContext) OnHttpResponseHeaders(_ int, _ bool) types.Action {

	currID := ds.currID
	currNode := ds.reqID2Info[currID]
	now := time.Now().UnixNano() / 1000000000
	timeDelta := now - currNode.timeStamp
	path := currNode.path

	split_path := strings.Split(path, "/")
	proxywasm.LogCriticalf("\n\n\tsplit path: %s\n\n", split_path)
	img_size := split_path[2]
	rps := split_path[3]

	// ds.counter++

	// log the powerKey response header
	proxywasm.LogInfof("\n\n\tpath: %s\n\ttime: %d\n\ttcurrent id: %s\n\tcurrent node ts: %d\n\ttime delta: %d", path, now, currID, currNode.timeStamp, timeDelta)

	proxywasm.LogInfof("\nrps %d\n", RPS.rps)

	// concatentate the path with the time delta
	new_path := fmt.Sprintf("/model/%s/%d/%s/%s", rps, timeDelta, img_size, img_size)
	// new_path = fmt.

	headers := [][2]string{
		{":method", "GET"},
		{":authority", ""},
		{"accept", "*/*"}, 
		{":path", new_path},
		{"rps", rps},
		{"img_size", img_size},
		{"execution_time", strconv.Itoa(int(timeDelta))},
		{"original_path", currNode.path},
		{"time_delta", strconv.Itoa(int(timeDelta))},
	}
	// Pick random value to select the request path.

	if _, err := proxywasm.DispatchHttpCall("model_ingress", headers, nil, nil, 5000, ctx.dispatchCallback); err != nil {
		proxywasm.LogCriticalf("dispatch httpcall failed: %v", err)
	}

	ctx.pendingDispatchedRequest++;

	// headers, err := proxywasm.GetHttpCallResponseHeaders()
	// proxywasm.LogInfof("response header for the dispatched call: %s: %s", headers[0][0], headers[0][1])
	// if err != nil && err != types.ErrorStatusNotFound {
	// 	panic(err)
	// }

	// for _, h := range headers {
	// 	proxywasm.LogInfof("response header for the dispatched call: %s: %s", h[0], h[1])
	// }

	return types.ActionPause
}


// dispatchCallback is the callback function called in response to the response arrival from the dispatched request.
func (ctx *httpContext) dispatchCallback(numHeaders, bodySize, numTrailers int) {
	// Decrement the pending request counter.
	// ctx.pendingDispatchedRequest--
	// if ctx.pendingDispatchedRequest == 0 {
	// 	// This case, all the dispatched request was processed.
	// 	// Adds a response header to the original response.
	// 	proxywasm.AddHttpResponseHeader("total-dispatched", strconv.Itoa(totalDispatchNum))
	// 	// And then contniue the original reponse.
	// 	proxywasm.ResumeHttpResponse()
	// 	proxywasm.LogInfof("response resumed after processed %d dispatched request", totalDispatchNum)
	// } else {
	// 	proxywasm.LogInfof("pending dispatched requests: %d", ctx.pendingDispatchedRequest)
	// }

	// ctx.pendingDispatchedRequest--;
	// if ctx.pendingDispatchedRequest == 0 {
	headers, err := proxywasm.GetHttpCallResponseHeaders()
	if err != nil && err != types.ErrorStatusNotFound {
		panic(err)
	}
	for _, h := range headers {
		proxywasm.LogInfof("response header for the dispatched call: %s: %s", h[0], h[1])
	}

	power_string := headers[3][1]
	stripped_power_string := strings.Replace(power_string, "[", "", -1)
	stripped_power_string = strings.Replace(stripped_power_string, "]", "", -1)


	proxywasm.LogInfof("\n\npower from model %s\n\n", headers[3][1])

	proxywasm.AddHttpResponseHeader("power-from-model", stripped_power_string)
	proxywasm.ResumeHttpResponse()
	proxywasm.LogInfo("response resumed after processed")
	// }
	// // proxywasm.LogInfof("called %d for contextID=%d", ctx.cnt, ctx.contextID)
	// headers, err := proxywasm.GetHttpCallResponseHeaders()
	// if err != nil && err != types.ErrorStatusNotFound {
	// 	panic(err)
	// }
	// for _, h := range headers {
	// 	proxywasm.LogInfof("response header for the dispatched call: %s: %s", h[0], h[1])
	// }
	// headers, err = proxywasm.GetHttpCallResponseTrailers()
	// if err != nil && err != types.ErrorStatusNotFound {
	// 	panic(err)
	// }
	// for _, h := range headers {
	// 	proxywasm.LogInfof("response trailer for the dispatched call: %s: %s", h[0], h[1])
	// }
}

// Override types.DefaultPluginContext.
func (ctx *pluginContext) OnTick() {
	RPS.rps = RPS.requests

	RPS.requests = 0
}

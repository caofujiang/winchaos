/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web"
	"github.com/sirupsen/logrus"
)

type HttpServer struct {
}

func NewHttpServer() web.APiServer {
	return &HttpServer{}
}

func (this HttpServer) RegisterHandler(handlerName string, handler web.ServerHandler) error {
	http.HandleFunc("/"+handlerName, func(writer http.ResponseWriter, request *http.Request) {
		logrus.Infof("RegisterHandler-request: %+v", request)
		err := request.ParseForm()
		if err != nil {
			logrus.Warnf("http handler: %s, get request param wrong, err: %v", handlerName, err)
			return
		}
		form := request.Form
		type Param struct {
			Cmd string `json:"cmd"`
			Ts  string `json:"ts"`
		}
		var warp struct {
			Params Param `json:"params"`
		}
		for k, _ := range form {
			if err := json.Unmarshal([]byte(k), &warp); err != nil {
				logrus.Warnf("json.Unmarshal-failed err: %v", err)
				return
			}
		}

		param := make(map[string]string)
		param["ts"] = warp.Params.Ts
		param["cmd"] = warp.Params.Cmd
		req := transport.Request{
			Headers: make(map[string]string),
			Params:  param,
		}
		m, _ := json.Marshal(req)

		//result, err := handler.Handle(request.Form["body"][0])
		result, err := handler.Handle(string(m))
		if err != nil {
			errBytes := fmt.Sprintf("handle %s request err, %v", handlerName, err)
			// TODO 存在 json 返回的风险
			logrus.Warningln(errBytes)
			result = errBytes
		}
		logrus.Infof("handler result: %s", string(result))
		_, err = writer.Write([]byte(result))
		if err != nil {
			logrus.Warningf("write response for %s err, %v", handlerName, err)
		}
	})
	return nil
}

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
package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/caofujiang/winchaos/pkg/options"
	"github.com/caofujiang/winchaos/transport"
)

type HttpClient struct {
	Config  transport.ServerConfig
	timeout uint32
	inited  bool

	client *http.Client
	url    url.URL
}

func NewHttpClient(config options.TransportConfig) (transport.TransportChannel, error) {
	if config.Endpoint == "" {
		logrus.Error("Transport endpoint is empty.")
		config.Endpoint = "10.148.55.113"
		//return nil, errors.New("transport endpoint is empty")
	}
	/*
	   namespace="default"
	   license="0813d72a71ba41ed986e507e2e0ead1b"
	   appInstance="chaos-default-app"
	   appGroup="chaos-default-app-group"
	   port="19527"
	   endpoint="10.148.55.113"
	   chaosctl.sh install -k  0813d72a71ba41ed986e507e2e0ead1b  -p  chaos-default-app  -g  chaos-default-app-group  -P 19527    -t 10.148.55.113
	*/

	hostAndPort := strings.SplitN(config.Endpoint, ":", 2)
	var port = 80
	if len(hostAndPort) > 1 {
		port, _ = strconv.Atoi(hostAndPort[1])
	}
	serverConfig := transport.ServerConfig{
		ClientVpcId:       options.Opts.VpcId,
		ClientIp:          options.Opts.Ip,
		ClientProcessFlag: options.ProgramName,
		ServerIp:          hostAndPort[0],
		ServerPort:        uint32(port),
		Timeout:           config.Timeout,
	}
	return GetDirectInstance(serverConfig), nil
}

func GetDirectInstance(config transport.ServerConfig) transport.TransportChannel {
	client := http.DefaultClient
	trans := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   config.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	trans.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client.Transport = trans

	return &HttpClient{
		Config:  config,
		inited:  true,
		timeout: uint32(config.Timeout.Milliseconds()),
		client:  client,
		url:     url.URL{Scheme: "http", Host: config.ServerIp + ":" + strconv.FormatUint(uint64(config.ServerPort), 10)},
	}
}

func (hc *HttpClient) DoInvoker(uri transport.Uri, jsonParam string) (string, error) {
	// 1. build request body
	var request transport.Request
	json.Unmarshal([]byte(jsonParam), &request)
	reqBody, _ := json.Marshal(request.GetBody())

	// 2. build request
	url := hc.url.String() + "/" + uri.HandlerName
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	// 3. send post request
	response, err := hc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// 4. handler response
	result, err := ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		if err != nil {
			return "", fmt.Errorf("direct http call %s and read message from response failed", uri.HandlerName)
		}
		return "", fmt.Errorf("direct http call %s failed, code: %d, body: %s", uri.HandlerName, response.StatusCode, string(result))
	}
	return string(result), nil
}

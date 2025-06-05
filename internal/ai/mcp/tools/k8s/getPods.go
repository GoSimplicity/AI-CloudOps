/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

func GetK8sPodsTool() mcp.Tool {
	return mcp.NewTool(
		"get_k8s_pods",
		mcp.WithDescription("获取k8s指定命名空间下所有的pod"),
		mcp.WithString("namespace",
			mcp.Description("Kubernetes命名空间，如果不指定则默认为default"),
		),
		mcp.WithString("cluster_id",
			mcp.Description("Kubernetes集群ID，如果不指定则默认为1"),
		),
	)
}

func GetK8sPodsToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := fmt.Sprintf("http://localhost:8889/api/k8s/pods/%s?namespace=%s", request.Params.Arguments["cluster_id"], request.Params.Arguments["namespace"])
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Apifox/1.0.0 (https://apifox.com)")
	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJLNW1CUEJZTlFlTldFQnZDVEU1bXNvZzNLU0dUZGhteCIsImV4cCI6MTk2NTEzMzYwMCwiVWlkIjoxLCJVc2VybmFtZSI6ImFkbWluIiwiU3NpZCI6ImMyMWM0OWNjLTM4NzQtNDg3Ni1hNGVlLWJkMjNmOWEyMTFkNyIsIlVzZXJBZ2VudCI6IkFwaWZveC8xLjAuMCAoaHR0cHM6Ly9hcGlmb3guY29tKSIsIkNvbnRlbnRUeXBlIjoiYXBwbGljYXRpb24vanNvbiJ9.w1Y_uSkC4EgLXx2pn7kEUaovl06O5prqz7jm52PCGNsiJesYslkPxpWYNx73wW_MLJSGmsfIvmTGW5m2aBnbGg")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Host", "localhost:8889")
	req.Header.Add("Connection", "keep-alive")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return utils.TextResult(string(body))
}

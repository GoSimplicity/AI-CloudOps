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

package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// PostWithJson 发送带有JSON字符串的POST请求
func PostWithJson(ctx context.Context, client *http.Client, l *zap.Logger, url string, jsonStr string, params map[string]string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		l.Error("创建 HTTP 请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if _, exists := headers["Content-Type"]; !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		l.Error("发送 HTTP 请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error("读取响应体失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		l.Error("服务器返回非2xx状态码",
			zap.String("url", url),
			zap.Int("statusCode", resp.StatusCode),
			zap.String("responseBody", string(bodyBytes)),
		)
		return bodyBytes, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	return bodyBytes, nil
}

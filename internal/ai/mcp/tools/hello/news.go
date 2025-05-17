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

package hello

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/PuerkitoBio/goquery"
	"github.com/mark3labs/mcp-go/mcp"
)

func GetDoubanTopMovies() mcp.Tool {
	return mcp.NewTool(
		"get_douban_top_movies",
		mcp.WithDescription("获取豆瓣评分前十的电影"),
	)
}

func GetDoubanTopMoviesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := "https://movie.douban.com/top250"

	// 创建请求，添加 User-Agent 防止被拦截
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/90.0.4430.212 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求豆瓣失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("豆瓣返回错误状态码: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析页面失败: %v", err)
	}

	var result strings.Builder
	result.WriteString("🎬 **豆瓣电影评分前十名：**\n\n")

	doc.Find(".grid_view li").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i >= 10 {
			return false // 只取前10
		}
		title := s.Find(".title").First().Text()
		rating := s.Find(".rating_num").Text()
		link, _ := s.Find(".hd a").Attr("href")

		result.WriteString(fmt.Sprintf("%d. [%s](%s) - ⭐️ %s 分\n", i+1, title, link, rating))
		return true
	})

	return utils.TextResult(result.String())
}

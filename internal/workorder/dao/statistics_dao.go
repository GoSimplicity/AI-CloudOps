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

package dao

import "context"

type StatisticsDAO interface {
	GetOverview(ctx context.Context)
	GetTrend(ctx context.Context)
	GetCategoryStats(ctx context.Context)
	GetPerformanceStats(ctx context.Context)
	GetUserStats(ctx context.Context)
}

type statisticsDAO struct {
}

func NewStatisticsDAO() StatisticsDAO {
	return &statisticsDAO{}
}

// GetCategoryStats implements StatisticsDAO.
func (s *statisticsDAO) GetCategoryStats(ctx context.Context) {
	panic("unimplemented")
}

// GetOverview implements StatisticsDAO.
func (s *statisticsDAO) GetOverview(ctx context.Context) {
	panic("unimplemented")
}

// GetPerformanceStats implements StatisticsDAO.
func (s *statisticsDAO) GetPerformanceStats(ctx context.Context) {
	panic("unimplemented")
}

// GetTrend implements StatisticsDAO.
func (s *statisticsDAO) GetTrend(ctx context.Context) {
	panic("unimplemented")
}

// GetUserStats implements StatisticsDAO.
func (s *statisticsDAO) GetUserStats(ctx context.Context) {
	panic("unimplemented")
}

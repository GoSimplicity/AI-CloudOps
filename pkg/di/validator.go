package di

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

import (
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
)

var trans ut.Translator

// InitTrans 初始化中文翻译器
func InitTrans() error {
	// 从 gin 中获取 validator 实例
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if !ok {
			return fmt.Errorf("初始化翻译器失败：无法获取 Validator 实例")
		}

		// 注册结构体字段 JSON tag 名称到验证器中
		v.RegisterTagNameFunc(extractJSONTag)

		// 初始化中文翻译器
		zhT := zh.New()
		uni := ut.New(zhT, zhT)

		// 获取中文翻译器
		var found bool
		trans, found = uni.GetTranslator("zh")
		if !found {
			return fmt.Errorf("初始化翻译器失败：无法找到中文翻译器")
		}

		// 注册中文翻译
		if err := zhTranslations.RegisterDefaultTranslations(v, trans); err != nil {
			return fmt.Errorf("注册中文翻译器失败：%v", err)
		}
	}

	return nil
}

// extractJSONTag 提取结构体字段的 JSON tag 作为验证的字段名称
func extractJSONTag(fld reflect.StructField) string {
	tag := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	if tag == "-" {
		return ""
	}
	return tag
}

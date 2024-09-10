package di

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
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return fmt.Errorf("初始化翻译器失败：无法获取 Validator 实例")
	}

	// 注册结构体字段 JSON tag 名称到验证器中
	v.RegisterTagNameFunc(extractJSONTag)

	// 初始化中文翻译器
	zhT := zh.New()
	uni := ut.New(zhT)

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

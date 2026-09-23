package middleware

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
)

var Translator ut.Translator

func InitValidator() {
	binding.EnableDecoderDisallowUnknownFields = true
	binding.EnableDecoderUseNumber = true

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			return strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		})

		uni := ut.New(en.New(), en.New())
		Translator, _ = uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(v, Translator)
	}
}

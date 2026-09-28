package middleware

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func TestInitValidatorEnablesStrictJSONAndUsesJSONFieldNames(t *testing.T) {
	InitValidator()
	if !binding.EnableDecoderDisallowUnknownFields || !binding.EnableDecoderUseNumber || Translator == nil {
		t.Fatal("InitValidator() did not enable strict decoding and initialize translations")
	}

	type request struct {
		DisplayName string `json:"display_name" binding:"required"`
	}
	err := binding.Validator.ValidateStruct(request{})
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok || len(validationErrors) != 1 || validationErrors[0].Field() != "display_name" {
		t.Fatalf("validation error=%#v, want JSON field name display_name", err)
	}
}

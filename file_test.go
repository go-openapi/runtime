package runtime

import (
	"testing"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/validate"
	"github.com/stretchr/testify/assert"
)

func TestValidateFile(t *testing.T) {
	fileParam := spec.FileParam("f")
	validator := validate.NewParamValidator(fileParam, nil)

	result := validator.Validate("str")
	assert.Equal(t, 1, len(result.Errors))
	assert.Equal(
		t,
		`f in formData must be of type file: "string"`,
		result.Errors[0].Error(),
	)

	result = validator.Validate(&File{})
	assert.True(t, result.IsValid())

	result = validator.Validate(File{})
	assert.True(t, result.IsValid())
}

package runtime

import (
	"testing"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFile(t *testing.T) {
	fileParam := spec.FileParam("f")
	validator := validate.NewParamValidator(fileParam, nil)

	result := validator.Validate("str")
	require.Len(t, result.Errors, 1)
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

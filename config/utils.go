package config

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type Validatable interface {
	Validate(validate *validator.Validate) error
}

func validateStruct(validate *validator.Validate, value any) error {
	err := validate.Struct(value)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			fmt.Println(err)
			return err
		}
	}
	return err
}

type Resolvable interface {
	Resolve() error
}

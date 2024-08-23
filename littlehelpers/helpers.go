package littlehelpers

import "errors"

func IfErrJoin(primaryErr error, resultErrors ...error) error {
	allErrors := []error{primaryErr}
	for _, err := range resultErrors {
		allErrors = append(allErrors, err)
	}

	return errors.Join(allErrors...)
}

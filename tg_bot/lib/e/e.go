package e

import "fmt"

func Wrap(mes string, err error) error {
	return fmt.Errorf("%s: %w", mes, err)
}

func WrapIfNil(mes string, err error) error {
	if err == nil {
		return nil
	}

	return Wrap(mes, err)
}

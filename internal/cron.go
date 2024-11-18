package internal

import (
	"errors"
	"fmt"
)

func RunCron() error {
	GetConfig()

	cronLocations, err := GetDueCronLocations(nil)
	if err != nil {
		return err
	}
	var errs []error
	for _, locationString := range cronLocations {
		if l, ok := GetLocation(locationString); ok {
			if err := l.RunCron(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("Encountered errors during cron process:\n%w", errors.Join(errs...))
	}
	return nil
}

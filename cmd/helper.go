package cmd

import (
	"errors"
	"fmt"

	"github.com/dromara/carbon/v2"
)

func ConvertTimeToCarbon(beginDate, endDate string) (carbon.Carbon, carbon.Carbon, error) {
	b := carbon.Parse(beginDate)
	if b.Error != nil {
		return carbon.Carbon{}, carbon.Carbon{}, fmt.Errorf("invalid begin date: %w", b.Error)
	}
	e := carbon.Parse(endDate)
	if e.Error != nil {
		return carbon.Carbon{}, carbon.Carbon{}, fmt.Errorf("invalid end date: %w", e.Error)
	}
	if b.Gt(e) {
		return carbon.Carbon{}, carbon.Carbon{}, errors.New("begin date is after end date")
	}
	return b, e, nil
}

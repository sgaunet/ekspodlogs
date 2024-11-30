package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

func InitAWSConfig(ctx context.Context, profile string) (cfg aws.Config, err error) {
	if len(ssoProfile) == 0 {
		cfg, err = config.LoadDefaultConfig(ctx)
	} else {
		// Try to connect with the SSO profile put in parameter
		cfg, err = config.LoadDefaultConfig(
			ctx,
			config.WithSharedConfigProfile(ssoProfile),
		)
	}
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to load SDK config: %w", err)
	}
	return cfg, nil
}

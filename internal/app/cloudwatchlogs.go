package app

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)



// recursive function to list on stdout tge loggroup
func (a *App) recurseListLogGroup(ctx context.Context, client *cloudwatchlogs.Client, NextToken string) (loggroups []string, err error) {
	return a.recurseListLogGroupWithDepth(ctx, client, NextToken, 0)
}

// recurseListLogGroupWithDepth handles pagination with depth limiting
func (a *App) recurseListLogGroupWithDepth(ctx context.Context, client *cloudwatchlogs.Client, NextToken string, depth int) (loggroups []string, err error) {
	var params cloudwatchlogs.DescribeLogGroupsInput
	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	if err := a.logGroupRateLimit.Wait(ctx); err != nil {
		return loggroups, fmt.Errorf("rate limit wait error: %w", err)
	}
	res, err := client.DescribeLogGroups(ctx, &params)
	if err != nil {
		return loggroups, err
	}
	for _, i := range res.LogGroups {
		// fmt.Printf("%s\n", *i.LogGroupName)
		loggroups = append(loggroups, *i.LogGroupName)
		// var glgfi cloudwatchlogs.GetLogGroupFieldsInput
		// glgfi.LogGroupName = i.LogGroupName

		// glgfo, _ := client.GetLogGroupFields(context.TODO(), &glgfi)
		// for _, logGrpF := range glgfo.LogGroupFields {
		// fmt.Println(*logGrpF.Name)
		// }
		// fmt.Println("")
	}
	if res.NextToken == nil {
		return loggroups, err
	} else {
		lg, err := a.recurseListLogGroupWithDepth(ctx, client, *res.NextToken, depth+1)
		loggroups = append(loggroups, lg...)
		return loggroups, err
	}
}
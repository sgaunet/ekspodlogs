package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// Recursive function that will return if the groupName parameter has been found or not
func (a *App) findLogGroup(clientCloudwatchlogs *cloudwatchlogs.Client, groupName string, NextToken string) bool {
	var params cloudwatchlogs.DescribeLogGroupsInput

	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	res, err := clientCloudwatchlogs.DescribeLogGroups(context.TODO(), &params)
	if err != nil {
		a.appLog.Errorln(err.Error())
		os.Exit(1)
	}
	for _, i := range res.LogGroups {
		a.appLog.Infof("## Parse Log Group Name : %s\n", *i.LogGroupName)
		if *i.LogGroupName == groupName {
			return true
		}
	}
	if res.NextToken == nil {
		// No token given, end of potential recursive call to parse the list of loggroups
		return false
	} else {
		return a.findLogGroup(clientCloudwatchlogs, groupName, *res.NextToken)
	}
}

// Parse every events of every streams of a group
// Recursive function
func (a *App) parseAllStreamsOfGroup(clientCloudwatchlogs *cloudwatchlogs.Client, groupName string, logStream string, nextToken string, minTimeStamp int64, maxTimeStamp int64) error {
	var paramsLogStream cloudwatchlogs.DescribeLogStreamsInput
	var stopToParseLogStream bool
	// Search logstreams of groupName
	// Ordered by last event time
	// descending
	paramsLogStream.LogGroupName = &groupName
	paramsLogStream.OrderBy = "LastEventTime"
	descending := true
	paramsLogStream.Descending = &descending

	if len(nextToken) != 0 {
		paramsLogStream.NextToken = &nextToken
	}
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs#Client.DescribeLogStreams
	res2, err := clientCloudwatchlogs.DescribeLogStreams(context.TODO(), &paramsLogStream)
	if err != nil {
		return err
	}
	// Loop over streams
	for _, j := range res2.LogStreams {
		if strings.Contains(*j.LogStreamName, logStream) {
			a.appLog.Debugln("Stream Name: ", *j.LogStreamName)
			a.appLog.Debugln("LasteventTimeStamp: ", *j.LastEventTimestamp)
			tm := time.Unix(*j.LastEventTimestamp/1000, 0) // aws timestamp are in ms
			a.appLog.Debugf("Parse stream : %s (Last event %v)\n", *j.LogStreamName, tm)

			// No need to parse old logstream older than minTimeStamp
			if *j.LastEventTimestamp < minTimeStamp {
				stopToParseLogStream = true
				a.appLog.Debugf("%v < %v\n", *j.LastEventTimestamp, maxTimeStamp)
				a.appLog.Debugln("stop to parse, *j.LastEventTimestamp < minTimeStamp")
				break
			} else {
				err = a.getEvents(context.TODO(), groupName, *j.LogStreamName, clientCloudwatchlogs, minTimeStamp, maxTimeStamp)
				if err != nil {
					a.appLog.Errorln(err.Error())
				}
			}
		}
	}

	if res2.NextToken != nil && !stopToParseLogStream {
		err := a.parseAllStreamsOfGroup(clientCloudwatchlogs, groupName, logStream, *res2.NextToken, minTimeStamp, maxTimeStamp)
		if err != nil {
			return err
		}
	}
	return err
}

// recursive function to list on stdout tge loggroup
func (a *App) recurseListLogGroup(client *cloudwatchlogs.Client, NextToken string) (loggroups []string, err error) {
	var params cloudwatchlogs.DescribeLogGroupsInput
	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	res, err := client.DescribeLogGroups(context.TODO(), &params)
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
		lg, err := a.recurseListLogGroup(client, *res.NextToken)
		loggroups = append(loggroups, lg...)
		return loggroups, err
	}
}

// function that parses every streams of loggroup groupName
func (a *App) FindLogStream(cfg aws.Config, groupName string, logStream string, startTime time.Time, endTime time.Time) error {
	clientCloudwatchlogs := cloudwatchlogs.NewFromConfig(cfg)

	doesGroupNameExists := a.findLogGroup(clientCloudwatchlogs, groupName, "")
	if !doesGroupNameExists {
		err := fmt.Errorf("GroupName %s not found", groupName)
		a.appLog.Errorln(err.Error())
		return err
	}

	minTimeStampInMs := startTime.Unix() * 1000
	maxTimeStampInMs := endTime.Unix() * 1000
	err := a.parseAllStreamsOfGroup(clientCloudwatchlogs, groupName, logStream, "", minTimeStampInMs, maxTimeStampInMs)
	return err
}

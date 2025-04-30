[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/ekspodlogs)](https://goreportcard.com/report/github.com/sgaunet/ekspodlogs)
[![GitHub release](https://img.shields.io/github/release/sgaunet/ekspodlogs.svg)](https://github.com/sgaunet/ekspodlogs/releases/latest)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/ekspodlogs/total)
[![GoDoc](https://godoc.org/github.com/sgaunet/ekspodlogs?status.svg)](https://godoc.org/github.com/sgaunet/ekspodlogs)
[![License](https://img.shields.io/github/license/sgaunet/ekspodlogs.svg)](LICENSE)

# ekspodlogs

It's a little utility to print logs of pods in an EKS cluster (Amazon Web Services). The logs have to be synchronised from cloudwatch first, there is no interaction with kubernetes API.

I want to keep it as is, and don't want to make a generic utility to print logs of cloudwatch. The goal is to get the logs of pods that have been written in cloudwatch by fluentd.

Here are some documentation to setup fluentd :

* https://docs.aws.amazon.com/fr_fr/AmazonCloudWatch/latest/monitoring/Container-Insights-setup-logs.html
* https://aws.amazon.com/fr/blogs/containers/fluent-bit-integration-in-cloudwatch-container-insights-for-eks/

The log events are retrieved from the loggroup named /aws/containerinsights/**Name of your cluster**/application.

**The initial development has been done in quick and dirty mode.** Maybe, this problem will be adressed in the future but it's a side project with very very low priority so don't expect a lot of features or improvements.

## Usage

```bash
$ ekspodlogs
Tool to parse logs of applications in an EKS cluster from AWS Cloudwatch

First, you need to configure your AWS credentials with the AWS CLI.
Then, you will have to synchronise the local database with the logs of cloudwatch for a period.

Finally, you will be able to request the logs of a specific logstream for a period.

Usage:
  ekspodlogs [flags]
  ekspodlogs [command]

Available Commands:
  help        Help about any command
  list-groups list-groups lists the log groups
  purge       Purge the local database
  req         requests the local database
  sync        synchronise the local database with the logs of cloudwatch
  version     print version of gitlab-expiration-token

Flags:
  -h, --help   help for ekspodlogs

Use "ekspodlogs [command] --help" for more information about a command.
```

Option -p should be used to login to AWS API when you have an SSO configured. It is the name of the profile to use.

```bash
$ grep profile  ~/.aws/config 
[profile dev]
[profile prod]
$ aws sso login --profile dev
...
$ ekspodlogs list-groups -p dev
...
```

If there are multiples EKS clusters, you have to specify the name of the log group with -g option.

The -g option is optionnal, if you have only one loggroup named /aws/containerinsights/**Name of your cluster**/application, no need to specify it.

Start date and end date allow to select logs that happened in this range of time.
Option -n allow to filter to the name of the pod which appears in the name of log stream.

## Execution

List loggroups if needed :

```bash
$ aws sso login --profile dev
...
$ ekspodlogs list-groups -p dev
...
```

Synchronise the local database with the logs of cloudwatch :

```bash
$ ekspodlogs sync -p dev -n mypodname -b "2021-01-01 00:00:00" -e "2021-01-01 23:59:59"
...
```

Request the logs of a specific logstream for a period :

```bash
$ ekspodlogs req -p dev -n mypodname -b "2021-01-01 00:00:00" -e "2021-01-01 23:59:59"
...
```

## Dependency

### Ubuntu/Debian

```bash
sudo apt-get update
sudo apt-get install -y libsqlite3-dev
```

### Fedora/RHEL/CentOS

```bash
sudo dnf install -y sqlite-devel
```

### Arch Linux

```bash
sudo pacman -Syu sqlite
```

## Debug

Set env variable DEBUGLEVEL to one of this value :

* error
* warn
* info (default)
* debug

## Misc

A log group contains the log streams. A log stream is a sequence of log events that share the same source. Each log stream has a unique name within its log group.

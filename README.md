# ekspodlogs

It's a little utility to print logs of pods in an EKS cluster (Amazon Web Services). The logs are parsed from cloudwatch, there is no interaction with kubernetes API.

I want to keep it as is, et don't want to make a generic utility to print logs of cloudwatch. The goal is to get the logs of pods that have been written in cloudwatch by fluentd. 

Here are some documentation to setup fluentd :

* https://docs.aws.amazon.com/fr_fr/AmazonCloudWatch/latest/monitoring/Container-Insights-setup-logs.html
* https://aws.amazon.com/fr/blogs/containers/fluent-bit-integration-in-cloudwatch-container-insights-for-eks/


So the pods log parsed on the hosts are /var/log/containers/*.log
And they are copied in the loggroup named /aws/containerinsights/**Name of your cluster**/application


**A little comment, there is no tests, the initial development has been done in quick and dirty mode. Maybe, this problem will be adressed in the future but it's a side project with very very low priority so don't expect a lot of features or improvements.**

## Usage

```
$ ekspodlogs -h
Usage of ekspodlog:
  -e string
        end date  (YYYY-MM-DD HH:MM:SS)
  -g string
        LogGroup to parse
  -l string
        logstream to search
  -lg
        List journal group
  -p string
        Auth by SSO
  -s string
        start date (YYYY-MM-DD HH:MM:SS)
  -v    Get version
```

Option -p should be used to login to AWS API when you have an SSO configured. It is the name of the profile to use.

```
$ grep profile  ~/.aws/config 
[profile dev]
[profile prod]
$ ekspodlogs -lg -p dev
...
```

So you must specify the loggroup of pods to the -g option. If you want to find it, use the -lg option to list all loggroup (Don't forget, it's like /aws/containerinsights/**Name of your cluster**/application).

The -g option is optionnal, if you have only one loggroup named /aws/containerinsights/**Name of your cluster**/application, no need to specify it.

Start date and end date allow to select logs that happened in this range of time. 
Option -l allow to filter to the name of the logstream (which should be like the podname).


## Execution 

List loggroups if needed :

```
$ ekspodlogs -lg -p dev
```

Get logs of stream named like kubewatch

```
$ ekslpodlogs -g /aws/containerinsights/prod-EKS/application -p prod -l kubewatch -s "2022-02-27 18:50" -e "2022-02-27 18:51"
```


# Debug

Set env variable DEBUGLEVEL to one of this value :

* error
* warn
* info (default)
* debug


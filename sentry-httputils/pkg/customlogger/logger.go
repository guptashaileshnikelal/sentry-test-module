package customlogger

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
)

//Flogger ...
var Flogger *fluent.Fluent

//Location ...
var Location *time.Location

//IsFloggerSuccess ...
var IsFloggerSuccess bool

//InitFluent initialize connection to fluent
func init() {
	var err error

	fluentHostArr := strings.Split(os.Getenv("FLUENT_HOST"), ",")
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(fluentHostArr))
	fluentHost := fluentHostArr[index]

	fluentPort := os.Getenv("FLUENT_PORT")
	fport, err := strconv.Atoi(fluentPort)
	Flogger, err = fluent.New(fluent.Config{FluentPort: fport, FluentHost: fluentHost, Timeout: 5 * time.Second, WriteTimeout: 2 * time.Second})
	IsFloggerSuccess = true
	if err != nil {
		fmt.Printf("ERROR::Could not connect to Fluent at %s Error : %v", os.Getenv("FLUENT_HOST"), err)
		IsFloggerSuccess = false
	}

	Location, _ = time.LoadLocation("Asia/Kolkata")
}

func Print(LogLevel string, TracePattern string, msg string) {
	counter, _, _, success := runtime.Caller(2)
	if !success {
		fmt.Print("functionName: runtime.Caller: failed")
	}
	currentFunction := runtime.FuncForPC(counter).Name()

	//msg := fmt.Sprintf(format, values...)
	if IsFloggerSuccess && Flogger != nil {

		awsExecuterName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME") + ":" + os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")

		// If its a batch the lambda function name will be empty
		if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
			awsExecuterName = os.Getenv("AWS_BATCH_JQ_NAME")
		}

		finalMsg := "[" + LogLevel + "][" + os.Getenv("GIT_REPO_NAME") + "][" + awsExecuterName + "][" + currentFunction + "] :: [" + TracePattern + "] :: [] :: " + msg

		AppName := "sentry"
		streamName := "stderr"
		if LogLevel == "INFO" {
			streamName = "stdout"
		}
		fluentTag := "kube_rewrite." + os.Getenv("ENV_NAME") + "." + AppName + "." + awsExecuterName + "." + streamName + ""
		logTime := fmt.Sprint(time.Now().Format(time.RFC3339))
		jsonMsg := map[string]string{"app_name": AppName, "log": finalMsg, "stream": streamName, "tag": fluentTag, "log_time": logTime, "pod_name": awsExecuterName, "request_id": TracePattern}

		if err := Flogger.Post(fluentTag, jsonMsg); err != nil {
			fmt.Printf("ERROR::Could not log to fluent %s : Message :  %v", err, msg)
			IsFloggerSuccess = false
			// : Raise panic alert
		}
		return
	}
}

// Error
func Error(v ...interface{}) {
	msg := fmt.Sprint(v...)
	Print("FATAL", GetTraceId(), msg)
}

// Errorf
func Errorf(formatStr string, args ...interface{}) {
	msg := fmt.Sprintf(formatStr, args...)
	Print("FATAL", GetTraceId(), msg)
}

// Info
func Info(v ...interface{}) {
	msg := fmt.Sprint(v...)
	Print("INFO", GetTraceId(), msg)
}

// Infof
func Infof(formatStr string, args ...interface{}) {
	msg := fmt.Sprintf(formatStr, args...)
	Print("INFO", GetTraceId(), msg)
}

// Fatal
func Fatal(v ...interface{}) {
	msg := fmt.Sprint(v...)
	Print("FATAL", GetTraceId(), msg)
}

func Fatalf(formatStr string, args ...interface{}) {
	msg := fmt.Sprintf(formatStr, args...)
	Print("FATAL", GetTraceId(), msg)
}

// Warning
func Warning(v ...interface{}) {
	msg := fmt.Sprint(v...)
	Print("WARNING", GetTraceId(), msg)
}

func Warningf(formatStr string, args ...interface{}) {
	msg := fmt.Sprintf(formatStr, args...)
	Print("WARNING", GetTraceId(), msg)
}

func SetTraceId(traceId string) {
	awsRequestID := os.Getenv("AwsRequestID")
	os.Setenv("TraceId_"+awsRequestID, traceId)
}

func GetTraceId() string {
	awsRequestID := os.Getenv("AwsRequestID")
	return os.Getenv("TraceId_" + awsRequestID)
}

func SetAwsRequestId(awsRequestId string) {
	os.Setenv("AwsRequestID", awsRequestId)
}

func GetAwsRequestId() string {
	return os.Getenv("AwsRequestID")
}

func InitLambdaGlobals(awsRequestId string) {
	SetAwsRequestId(awsRequestId)
	os.Setenv("StartTime_"+awsRequestId, time.Now().Format(time.RFC3339))
	// Over write the trace ID everytime
	SetTraceId("X-X-X-" + time.Now().Format("20060102150405"))
}

func GetLambdaStartTime() time.Time {
	awsRequestId := GetAwsRequestId()
	startTime := os.Getenv("StartTime_" + awsRequestId)
	startTimeObj, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return time.Now()
	}
	return startTimeObj
}

func PrintExitLogLine() {
	Infof("AWS_REQUEST_ID: %s Execution time %f seconds", GetAwsRequestId(), time.Now().Sub(GetLambdaStartTime()).Seconds())
}

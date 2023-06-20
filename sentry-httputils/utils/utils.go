package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	logger "sentry-httputils/pkg/customlogger"
	"sentry-httputils/pkg/retry"
	"sentry-httputils/utils/consts"
	"strconv"
	"time"
)

const logFormatRetry = "Retrying number: %d, for uri=%s, last err: %s"
const logFormatHeader = "POST CALL OPTIONAL HEADERS :: %s"
const timeoutText = "timeout"
const headerText = "headers"
const queryParamText = "query_params"
const queryParamGetText = "queryParams"

func getTimeout(optionalParams map[string]interface{}) int {
	timeOut := consts.DefaultHTTPTimeout

	if len(os.Getenv("HTTP_TIMEOUT")) > 0 {
		timeOut, _ = strconv.Atoi(os.Getenv("HTTP_TIMEOUT"))
	}
	
	// If we have a custom timeout defined, otherwise it will use the default timeput set
	if _, ok := optionalParams[timeoutText].(int); ok {
		if optionalParams[timeoutText].(int) > 0 {
			timeOut = optionalParams[timeoutText].(int)
		}
	}

	return timeOut
}

func appendQueryParams(optionalParams map[string]interface{}, req *http.Request, paramName string) *http.Request {
	if _, ok := optionalParams[paramName].(map[string]string); ok {
		if len(optionalParams[paramName].(map[string]string)) > 0 {
			q := req.URL.Query()

			for i, v := range optionalParams[paramName].(map[string]string) {
				q.Add(i, v)
			}

			req.URL.RawQuery = q.Encode()
		}
	}
	return req
}

func appendHeaders(optionalParams map[string]interface{}, req *http.Request) *http.Request {
	if _, ok := optionalParams[headerText].(map[string]string); ok {
		if len(optionalParams[headerText].(map[string]string)) > 0 {
			debugHeaders, _ := json.Marshal(optionalParams[headerText].(map[string]string))
			logger.Infof(logFormatHeader, debugHeaders)
			for i, v := range optionalParams[headerText].(map[string]string) {
				req.Header.Set(i, v)
			}
		}
	}
	return req
}

func DoGetWithoutPort(server, uri string) (response string, statusCode int, err error) {
	uriPath := fmt.Sprintf("%s%s", server, uri)
	logger.Infof("uriPath=%s", uriPath)

	return retry.Do(
		func() (string, int, error) {
			response, statusCode, err := DoGet(uriPath, nil)
			if err != nil {
				logger.Infof("Error: Response for GET %s: resp = %s, status=%d, err=%s",uriPath, response, statusCode, err.Error())
				return response,statusCode, err
			}
			//logger.Infof("Response for GET %s: resp = %s, status=%d",uriPath, response,statusCode)
			return response, statusCode, nil
		},
		retry.DelayType(func(n uint, config *retry.Config) time.Duration {
			return time.Duration(n*2) * time.Second
		}),
		retry.OnRetry(func(n uint, err error) {
			logger.Infof(logFormatRetry, n, uriPath, err.Error())
		}),
		retry.Attempts(3),
	)
}

func DoPostWithoutPort(server, resource string, payloadBytes []byte) (response string, statusCode int, err error) {
	uriPath := fmt.Sprintf("%s%s", server, resource)
	logger.Infof("uripath=%s", uriPath)
	// return DoPost(uriPath, payloadBytes, nil)
	return retry.Do(
		func() (string, int, error) {
			response, statusCode, err := DoPost(uriPath, payloadBytes, nil)
			if err != nil {
				logger.Infof("Error: Response for POST %s: resp = %s, status=%d, err=%s",uriPath, response, statusCode, err.Error())
				return response,statusCode, err
			}
			// logger.Infof("Response for GET %s: resp = %s, status=%d",uriPath, response,statusCode)
			return response, statusCode, nil
		},
		retry.DelayType(func(n uint, config *retry.Config) time.Duration {
			return time.Duration(n*2) * time.Second
		}),
		retry.OnRetry(func(n uint, err error) {
			logger.Infof(logFormatRetry, n, uriPath, err.Error())
		}),
		retry.Attempts(3),
	)
}

func DoPatchWithoutPort(uriPath string, payloadBytes []byte, optionalParams map[string]interface{}) (response string, statusCode int, err error) {
	//return DoPatch(uriPath, payloadBytes, optionalParams)
	return retry.Do(
		func() (string, int, error) {
			response, statusCode, err := DoPatch(uriPath, payloadBytes, nil)
			if err != nil {
				logger.Infof("Error: Response for Patch %s: resp = %s, status=%d, err=%s",uriPath, response, statusCode, err.Error())
				return response,statusCode, err
			}
			//logger.Infof("Response for GET %s: resp = %s, status=%d",uriPath, response,statusCode)
			return response, statusCode, nil
		},
		retry.DelayType(func(n uint, config *retry.Config) time.Duration {
			return time.Duration(n*2) * time.Second
		}),
		retry.OnRetry(func(n uint, err error) {
			logger.Infof(logFormatRetry, n, uriPath, err.Error())
		}),
		retry.Attempts(3),
	)
}

func DoPost(uriPath string, payloadBytes []byte, optionalParams map[string]interface{}) (response string, statusCode int, err error) {

	timeOut := getTimeout(optionalParams)

	client := &http.Client{Timeout: time.Duration(timeOut) * time.Second}

	noData := ""

	logger.Infof("POST CALL REQUEST :: URL: %s , PAYLOAD: %s", uriPath, string(payloadBytes))
	// // resp, err := http.Post(uriPath, "application/json", bytes.NewBuffer(payloadBytes))
	req, err := http.NewRequest("POST", uriPath, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return noData, 0, err
	}

	req = appendQueryParams(optionalParams, req, queryParamText)
	req = appendHeaders(optionalParams, req)

	startTime := time.Now()

	resp, err := client.Do(req)

	timeDiff := time.Since(startTime)

	if err != nil {
		return noData, 0, err
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return noData, resp.StatusCode, readErr
	}

	logger.Infof("POST CALL RESPONSE :: %s, TIME TAKEN:: %v", string(body), timeDiff)

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return string(body), resp.StatusCode, errors.New(string(body))
	}
	return string(body), resp.StatusCode, nil
}

func DoGet(uriPath string, optionalParams map[string]interface{}) (response string, statusCode int, err error) {
	timeOut := getTimeout(optionalParams)

	client := &http.Client{Timeout: time.Duration(timeOut) * time.Second}

	noData := ""
	errRespCode := 0
	req, err := http.NewRequest("GET", uriPath, http.NoBody)
	if err != nil {
		logger.Errorf("Error recvd while creating request, err=%s", err.Error())
		return noData, errRespCode, err
	}

	req = appendQueryParams(optionalParams, req, queryParamGetText)
	req = appendHeaders(optionalParams, req)

	logger.Infof("GET CALL REQUEST :: URL: %s , REQUEST: %s", uriPath, req.URL.String())

	startTime := time.Now()

	resp, err := client.Do(req)

	timeDiff := time.Since(startTime)

	if err != nil {
		logger.Errorf("Error recvd while firing GET call, err=%s", err.Error())
		return noData, errRespCode, err
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		logger.Errorf("Error recvd while reading response body, err=%s", readErr.Error())
		return noData, resp.StatusCode, readErr
	}
	defer resp.Body.Close()

	logger.Infof("GET CALL RESPONSE :: %s, TIME TAKEN:: %v", string(body), timeDiff)

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("return status code not OK, statusCode=%d", resp.StatusCode)
		return string(body), resp.StatusCode, errors.New("failed response")
	}
	return string(body), resp.StatusCode, nil
}

func DoPatch(uriPath string, payloadBytes []byte, optionalParams map[string]interface{}) (response string, statusCode int, err error) {
	timeOut := getTimeout(optionalParams)

	client := &http.Client{Timeout: time.Duration(timeOut) * time.Second}

	logger.Infof("PATCH CALL REQUEST :: URL: %s , PAYLOAD: %s", uriPath, string(payloadBytes))

	noData := ""
	req, err := http.NewRequest("PATCH", uriPath, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return noData, 0, err
	}
	req = appendQueryParams(optionalParams, req, queryParamText)
	req = appendHeaders(optionalParams, req)

	startTime := time.Now()

	resp, err := client.Do(req)

	timeDiff := time.Since(startTime)

	if err != nil {
		return noData, 0, err
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return noData, resp.StatusCode, readErr
	}

	logger.Infof("PATCH CALL RESPONSE :: %s, TIME TAKEN:: %v", string(body), timeDiff)

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return string(body), resp.StatusCode, errors.New(string(body))
	}
	return string(body), resp.StatusCode, nil
}

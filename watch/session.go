package watch

import (
	"errors"
	"net/http"
	"net/url"

	"time"

	"io"

	"encoding/json"

	"bytes"

	"github.com/foomo/petze/config"
	"io/ioutil"
)

func runSession(service *config.Service, r *Result, client *http.Client) error {
	//log.Println("running session with session length:", len(service.Session))
	// utils.JSONDump(service)
	endPointURL, errURL := service.GetURL()
	if errURL != nil {
		return errors.New("can not run session: " + errURL.Error())
	}
	for _, call := range service.Session {
		// copy URL
		callURL := &url.URL{}
		*callURL = *endPointURL

		uriURL, errURIURL := call.GetURL()
		if errURIURL != nil {
			return errURIURL
		}

		callURL.Path = uriURL.Path
		callURL.RawQuery = uriURL.RawQuery

		var body io.Reader
		method := http.MethodGet
		if call.Method != "" {
			method = call.Method
		}
		if call.Data != nil {
			dataBytes, errDataBytes := json.Marshal(call.Data)
			if errDataBytes != nil {
				return errors.New("could not encode data bytes: " + errDataBytes.Error())
			}
			body = bytes.NewBuffer(dataBytes)
		}

		req, errNewRequest := http.NewRequest(method, callURL.String(), body)
		if errNewRequest != nil {
			return errNewRequest
		}
		start := time.Now()
		response, errResponse := client.Do(req)
		if errResponse != nil {
			return errResponse
		}
		defer response.Body.Close()

		duration := time.Since(start)
		responseBody, errReadAll := ioutil.ReadAll(response.Body)
		if errReadAll != nil {
			r.Errors = append(r.Errors, Error{Error: "could not read from response" + errReadAll.Error(), Type: ErrorBadResponseBody})
		}

		for _, chk := range call.Check {

			ctx := &CheckContext{
				response:     response,
				responseBody: responseBody,
				check:        chk,
				call:         call,
				duration:     duration,
			}
			r.Errors = append(r.Errors, checkResponse(ctx)...)
		}

	}
	return nil
}

type CheckContext struct {
	response     *http.Response
	responseBody []byte
	check        config.Check
	call         config.Call
	duration     time.Duration
}

var ContextValidators = []ValidatorFunc{
	ValidateJsonPath,
	ValidateGoQuery,
	ValidateDuration,
	ValidateContentType,
	ValidateRegex,
}

func checkResponse(ctx *CheckContext) []Error {
	errs := []Error{}

	for _, validator := range ContextValidators {
		errs = append(errs, validator(ctx)...)
	}

	return errs
}

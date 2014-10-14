package util

import (
    "bytes"
    "errors"
    "io"
    "net/http"
    //"path/filepath"
)


type readCloser struct {
    io.Reader
}

func (readCloser) Close() error {
    return nil
}

func CallAPI(method, url string, content *[]byte, h map[string]string) (*http.Response, error) {
    if len(h) % 2 == 1 {
        return nil, errors.New("syntax err: # header != # of values")
    }

    req, err := http.NewRequest(method, url, nil)
    if err != nil {
        return nil, err
    }

    for k, v := range h {
        req.Header.Set(k,v)
    }

    req.ContentLength = int64(len(*content))

    if req.ContentLength > 0 {
        req.Body = readCloser{bytes.NewReader(*content)}
    }

    return (new(http.Client)).Do(req)
}

func CheckHttpResponseStatusCode(resp *http.Response) error {
	switch resp.StatusCode {
	case 200, 201, 202, 204:
		return nil
	case 400:
		return errors.New("Error: response == 400 bad request")
	case 401:
		return errors.New("Error: response == 401 unauthorised")
	case 403:
		return errors.New("Error: response == 403 forbidden")
	case 404:
		return errors.New("Error: response == 404 not found")
	case 405:
		return errors.New("Error: response == 405 method not allowed")
	case 409:
		return errors.New("Error: response == 409 conflict")
	case 413:
		return errors.New("Error: response == 413 over limit")
	case 415:
		return errors.New("Error: response == 415 bad media type")
	case 422:
		return errors.New("Error: response == 422 unprocessable")
	case 429:
		return errors.New("Error: response == 429 too many request")
	case 500:
		return errors.New("Error: response == 500 instance fault / server err")
	case 501:
		return errors.New("Error: response == 501 not implemented")
	case 503:
		return errors.New("Error: response == 503 service unavailable")
	}
	return errors.New("Error: unexpected response status code")
}

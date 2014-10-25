package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
    "net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
    "time"
    "github.com/belogik/goes"
)

type Walker struct {
	Dirs  []string
	Files []string
}

type readCloser struct {
	io.Reader
}

func (readCloser) Close() error {
	return nil
}

func Trace(s string, args ...interface{}) (string, time.Time) {
	log.Println("Task Starting: ", time.Now(), s, args)
	fmt.Println("Task Starting: ", time.Now(), s, args)
	return s, time.Now()
}

func Un(s string, startTime time.Time, args ...interface{}) {
	endTime := time.Now()
	log.Println("Task Ending:", endTime, s, args, "ElapsedTime: ", endTime.Sub(startTime))
	fmt.Println("Task Ending:", endTime, s, args, "ElapsedTime: ", endTime.Sub(startTime))
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func Task(f func(string, string, string, chan http.Response) error, path, url, traceLevel string, ch chan http.Response) error {

	_, t1 := Trace(GetFunctionName(f), path)
	defer Un(GetFunctionName(f), t1, path)

	err := f(path, url, traceLevel, ch)

	if err != nil {
		log.Fatal(err)
	}
	return err
	//    if len(args) > 1 {
	//        go f.(func(...interface{}))(args)
	//    } else if len(args) == 1 {
	//        go f.(func(interface{}))(args[0])
	//    } else {
	//        go f.(func())()
	//    }
}
func GetDirAndFileList(path string) (Walker, error) {

	walker := new(Walker)
	err := filepath.Walk(path,
		func(path string, fi os.FileInfo, err error) error {
			if fi == nil {
				fmt.Println(err)
				log.Println(err)
				return err
			}
			if fi.IsDir() {
				walker.Dirs = append(walker.Dirs, path)

			} else {
				walker.Files = append(walker.Files, path)
			}
			return nil
		})

	return *walker, err
}

func CallAPI(method, url string, content *[]byte, h map[string]string) (*http.Response, error) {

	_, t1 := Trace(reflect.TypeOf(CallAPI).Name(), method, url)
	defer Un(reflect.TypeOf(CallAPI).Name(), t1, method, url)
	if len(h)%2 == 1 {
		return nil, errors.New("syntax err: # header != # of values")
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range h {
		req.Header.Set(k, v)
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
	fmt.Println("Error: unexpected response status code: ", resp.StatusCode)
	log.Fatal("Error: unexpected response status code: ", resp.StatusCode)
	return errors.New("Error: unexpected response status code")
}


/**Upload File(s) to polaris storage
 * @param path the file(s) to upload 
 * @param url the API uri
 * @param traceLevel the log level
 * @param ch the chan to transit http.Response
 */
func UploadFile(path string, url string, traceLevel string, ch chan http.Response) error {

	method := "PUT"
	var headers map[string]string
	headers = make(map[string]string)
	headers["Authorization"] = "Bearer " + os.Getenv("TOKEN")
	headers["Content-type"] = "text/plain"

	fContent, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("url: %s\nmethod: %s\n", url, method)
	if strings.ToLower(traceLevel) == "debug" {
		for k, v := range headers {
			fmt.Printf("%s: %s\n", k, v)
			log.Printf("%s: %s\n", k, v)
		}
	}

	res, err := CallAPI(method, url, &fContent, headers)

	if err != nil {
		log.Fatal(err)
	}

	err = CheckHttpResponseStatusCode(res)
	if err != nil {
		if 401 == res.StatusCode {
			fmt.Println(err)
			log.Fatal(err)
		} else {
			log.Fatal(err)
		}
	}

	ch <- *res

	return err
}

/**Index the metadata to Elasticsearch
 * @esConn the connection between client and ES
 * @d the metadata document to indexing
 * @extraArgs extral arguments sent to ES
 * @ch the chan for goroutine communications
 */
func IndexMetadata(esConn goes.Connection, d goes.Document, extraArgs url.Values, ch chan goes.Response) (err error) {
    r, err := esConn.Index(d, extraArgs)
    ch <- r
    return 
}

/**Delete the metadata from ES
 * @esConn the connection between client and ES
 * @d the metadata document to delete from ES
 * @extraArgs extral arguments sent to ES
 * @ch the chan for goroutine communications
 */
 func DeleteMetadata(esConn goes.Connection, d goes.Document, extraArgs url.Values, ch chan goes.Response) (err error) {
     r, err := esConn.Delete(d, extraArgs)
     ch <- r
     return 
 }

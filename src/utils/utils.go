package utils

import (
	"fmt"
	"io"
    "os"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
    "time"
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

func FindElementInArray(array []string, e interface{}) (pos int, has bool) {

    for i, element := range array {
        if e == element {
            pos, has = i, true
            return
        }
    }
    return -1, false
} 

func Task(f func(string, string, string, chan http.Response) error, path, url, traceLevel string, ch chan http.Response) error {

	s, t1 := Trace(GetFunctionName(f), path)
	defer Un(s, t1, path)

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

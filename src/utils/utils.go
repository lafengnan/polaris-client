package utils

import (
    "fmt"
    "io"
    "os"
    "log"
    "net/http"
    "path/filepath"
    "reflect"
    "runtime"
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

func Pinfo(logger *log.Logger, format string, args... interface{}) {
    fmt.Printf(format, args...)
    if logger != nil {
        logger.Printf(format, args...)
    }
}

func Perr(logger *log.Logger, err error, fatal bool) {
    if err != nil {
        fmt.Println(err)
        if logger != nil {
            logger.Println(err)
        }
        if fatal {
            os.Exit(1)
        }
    }
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

func NewTask(f interface{}, args... interface{}) (err error) {
    s, t1 := Trace(GetFunctionName(f), args)
    defer Un(s, t1, args)
    if len(args) > 1 {
        err = f.(func(...interface{})(err error))(args)
    } else if len(args) == 1 {
        err = f.(func(interface{})(err error))(args[0])
    } else {
        err = f.(func()(err error))()
    }

    return
}

/**Start a File operations Task
 * @f the function to execute
 * @path the file/dir path
 * @url the service uri
 * @ch the chan with *http.Response
 */
func FileTask(f func(chan *http.Response, ... interface{}) error, ch chan *http.Response, args... interface{}) {

	s, t1 := Trace(GetFunctionName(f), args...)
	defer Un(s, t1, args...)

	err := f(ch, args...)
    Perr(nil, err, true)
}

/**Get dirs and files list of a given path
 * @path the file/dir path to analysis
 */
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

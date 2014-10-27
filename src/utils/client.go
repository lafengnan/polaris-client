package utils 

import (
    "os"
    "fmt"
    "log"
    "time"
    "errors"
    "bytes"
    "strings"
    "runtime"
    "reflect"
    "net/http"
    "net/url"
    "io/ioutil"
    "path/filepath"
    "github.com/belogik/goes"
)

var Commands []string  = []string {
    "UploadFile",
    "DeleteFile",
    "ListFiles",
    "IndexDocument",
    "DeleteDocument",
}

const (
    WAITTING = iota
    RUNNING
    DONE
    UNKOWN
)

type PolarisCommand struct {
    Status int
    Command string
}

type PolarisClient struct {
    ClientId string
    UserId string
    Token string
    StorageServiceURL string
    MetadataServiceURL string
    TraceLevel string
    Command *PolarisCommand
    Logger *log.Logger
    TotalTasks int
    ActiveTasks int
    Timeout chan int
}

type ReturnedFile struct {
    Path string
    ContentLenght int
    Etag string
    ContentType string
    LastModified string
    UUID string
}

type FileOps interface {
    UploadDir(path string, ch chan *http.Response) (err error)
    ListFile(path string, ch chan *http.Response) (err error)
    UploadFile(path string, ch chan *http.Response) (err error)
    DeleteFile(path string, ch chan *http.Response) (err error)
}

type MetadataOps interface {
    IndexDocument(esConn goes.Connection, d goes.Document, extraArgs url.Values, ch chan *goes.Response) (err error)
    DeleteDocument(esConn goes.Connection, d goes.Document, extraArgs url.Values, ch chan *goes.Response) (err error)
}

/**Display stats of the test
 * @begin the begin time for Stat
 * @end the end time for Stat
 */
func (c *PolarisClient) Stat(begin, end time.Time) (err error) {
    if end.Sub(begin) < 0 {
        err = errors.New("End time should later than begin time")
    }
    if err == nil{
        completed := c.TotalTasks - c.ActiveTasks
        duration := end.Sub(begin).Nanoseconds()
        Parallel := runtime.NumCPU()

        fmt.Println(completed, "Files Uploaded!")
        c.Logger.Println(completed, "Files Uploaded!")
        fmt.Printf("Concurrency: %d, Parallel: %d\n", int64(c.TotalTasks)*1E9/duration, Parallel)
        c.Logger.Printf("Concurrency: %d, Parallel: %d\n", int64(c.TotalTasks)*1E9/duration, Parallel)
    }

    return 
}

func (c *PolarisClient) Info() {
    c.Logger.Println("userId: ", c.UserId)
    c.Logger.Println("clientId: ", c.ClientId)
    c.Logger.Printf("log level: %s\n", c.TraceLevel)
    c.Logger.Printf("total tasks: %d\n", c.TotalTasks)
    c.Logger.Printf("active tasks: %d\n", c.ActiveTasks)
    c.Logger.Printf("command: {%s:%d}\n", c.Command.Command, c.Command.Status)
    c.Logger.Println("storage service: ", c.StorageServiceURL)
    c.Logger.Println("metadata service: ", c.MetadataServiceURL)
    c.Logger.Println("token: ", c.Token)
}

/**Initialize Polaris Cient
 * @clientId client id
 * @userId user id for a specific user
 * @token authorization token for the user
 * @stVC storage service uri
 * @mdVC metadata service uri
 * @traceLevel log level to display logs
 * @cmd command list to run 
 * @logger the logger fo client
 */
func (c *PolarisClient)Init(clientId, UserId, token, stVC, mdVC, traceLevel string, cmd *PolarisCommand, logger *log.Logger, tasks int, timeoutCh chan int) (errs []error) {

    s, t1 := Trace(GetFunctionName(c.Init))
    defer Un(s, t1)

    *c = PolarisClient{clientId, UserId, token, stVC, mdVC, strings.ToLower(traceLevel), cmd, logger, tasks, 0, timeoutCh}

    if c.Logger == nil {
        errs = append(errs, errors.New("logger of client is not set"))
    }
    if len(c.ClientId) == 0 {
        errs = append(errs, errors.New("client id is not set"))
    }
    if len(c.UserId) == 0 || len(c.Token) == 0 {
        c.Logger.Println("user id: ", c.UserId, "token: ", token)
        errs = append(errs, errors.New("user id or token is not set correctly"))
    }
    if len(traceLevel) == 0 {
        c.Logger.Println("Tracelevle is default: info")
    }
    if len(c.StorageServiceURL) == 0 && len(c.MetadataServiceURL) == 0 {
        errs = append(errs, errors.New("please set any of services:[storage,metadata]"))
    }

    if c.Command == nil {
        errs = append(errs, errors.New("No commands set!"))
    } else {
        _, ok := FindElementInArray(Commands, cmd.Command)
        if ok == false {
            errs = append(errs, errors.New("Wrong Command"))
        }
    }

    return  
}

/**Call CallHTTP Restful API
 * @method the standard HTTP method
 * @url the endpoint of Restufl API
 * @content the content to send
 * @h headers of the request
 */
func CallAPI(method, url string, content *[]byte, h map[string]string) (*http.Response, error) {

	_, t1 := Trace(reflect.TypeOf(CallAPI).Name(), method, url)
	defer Un(reflect.TypeOf(CallAPI).Name(), t1, method, url)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range h {
		req.Header.Set(k, v)
	}

	if content != nil {
        req.ContentLength = int64(len(*content))
        if req.ContentLength > 0 {
            req.Body = readCloser{bytes.NewReader(*content)}
        }
	}

	return (new(http.Client)).Do(req)
}

/**Check the response status code
 * @resp the http response
 */
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

/**Upload a given directory to storage
 * @dir the directory path to Upload
 * @ch the chan for communication
 */
func (c *PolarisClient) UploadDir(dir string, ch chan *http.Response ) (err error) {
    fileInfo, err := os.Stat(dir)
    if err != nil {
        c.Logger.Fatal(err)
    }
    if fileInfo.IsDir() == false {
        c.Logger.Printf("%s is not a directory", dir)
        os.Exit(1)
    } else {
        walker, err := GetDirAndFileList(dir)
        ch = make(chan *http.Response, len(walker.Files))
        if err != nil {
            c.Logger.Fatal(err)
        }
        fmt.Printf("Preapare to upload %d files\n", len(walker.Files))
        c.Logger.Printf("Preapare to upload %d files\n", len(walker.Files))
        
        if  c.TraceLevel == "debug" {
            for _, f := range walker.Files {
                fmt.Println(f)
                c.Logger.Println(f)
            }
        }
        t1 := time.Now()
        for _, filename := range walker.Files {
            go FileTask(c.UploadFile, filename, ch)
        }

        t2 := time.Now()
        for i := 0; i < len(walker.Files); i++ {
            select {
            case <-c.Timeout:
                fmt.Println("Timeout!")
                c.Logger.Println("Timeout!")
                c.Command.Status = UNKOWN
                break
            case r := <-ch:
                c.ActiveTasks--
                if c.TraceLevel == "debug" {
                    fmt.Println(r)
                    c.Logger.Println(r)
                }
            }
        }
        if c.Command.Status != UNKOWN {
            c.Command.Status = DONE
        }

        defer c.Stat(t1, t2)
    }
    return
}

/**Upload File(s) to polaris storage
 * @param path the file(s) to upload 
 * @param url the API uri
 * @param traceLevel the log level
 * @param ch the chan to transit http.Response
 */
 func (c *PolarisClient) UploadFile(path string, ch chan *http.Response) (err error) {

     c.ActiveTasks++
     method := "PUT"
     var headers map[string]string
     headers = make(map[string]string)
     headers["Authorization"] = "Bearer " + c.Token
     headers["Content-type"] = "text/plain"

     fContent, err := ioutil.ReadFile(path)
     if err != nil {
         c.Logger.Fatal(err)
     }

     if err != nil {
         c.Logger.Fatal(err)
     }

     url := c.StorageServiceURL + "/" + c.UserId + "/files/" + filepath.Base(path) + "?previous="

     c.Logger.Printf("url: %s\nmethod: %s\n", url, method)
     if strings.ToLower(c.TraceLevel) == "debug" {
         for k, v := range headers {
             fmt.Printf("%s: %s\n", k, v)
             c.Logger.Printf("%s: %s\n", k, v)
         }
     }

     r, err := CallAPI(method, url, &fContent, headers)

     if err != nil {
         c.Logger.Println(err)
     }

     err = CheckHttpResponseStatusCode(r)
     if err != nil {
         if 401 == r.StatusCode {
             fmt.Println(err)
             c.Logger.Println(err)
         } else {
             c.Logger.Println(err)
         }
     }
     ch <- r
     return
 }

func (c *PolarisClient) DeleteFile(path string, ch chan *http.Response) (err error) {
    return
}

func (c *PolarisClient) ListFile(path string, ch chan *http.Response)(err error) {

    method := "GET"
    var headers map[string]string
    headers = make(map[string]string)
    headers["Authorization"] = "Bearer " + c.Token
    c.ActiveTasks++

    url := c.StorageServiceURL + "/" + c.UserId + "/files"

    r, err := CallAPI(method, url, nil, headers)

    if err != nil {
        c.Logger.Println(err)
    }

    err = CheckHttpResponseStatusCode(r)
     if err != nil {
         if 401 == r.StatusCode {
             fmt.Println(err)
             c.Logger.Println(err)
         } else {
             c.Logger.Println(err)
         }
     }

     ch <- r
    return 
}

/**Index the metadata to Elasticsearch
 * @esConn the connection between client and ES
 * @d the metadata document to indexing
 * @extraArgs extral arguments sent to ES
 */
func (c *PolarisClient) IndexDocument(es goes.Connection, d goes.Document, extraArgs url.Values) (ch chan goes.Response, err error) {
    r, err := es.Index(d, extraArgs)
    ch <- r
    return
}

/**Delete the metadata from ES
 * @esConn the connection between client and ES
 * @d the metadata document to delete from ES
 * @extraArgs extral arguments sent to ES
 */
func (c *PolarisClient) DeleteDocument(es goes.Connection, d goes.Document, extraArgs url.Values) (ch chan goes.Response, err error) {
    r, err := es.Delete(d, extraArgs)
    ch <- r
    return
}

package utils 

import (
    "os"
    "fmt"
    "log"
    "time"
    "bytes"
    "errors"
    "strconv"
    "strings"
    "runtime"
    "reflect"
    "net/http"
    "net/url"
    "io/ioutil"
    "path/filepath"
    "github.com/belogik/goes"
    "config"
)

var Commands []string  = []string {
    "ListFiles",
    "UploadFile",
    "DeleteFile",
    "DeleteAllFiles",
    "IndexDocument",
    "DeleteDocument",
}

const (
    WAITING = iota
    RUNNING
    DONE
    UNKOWN
)

type PolarisCommand struct {
    Command string
    Status int
}

type PolarisClient struct {
    ClientId string
    Users map[string]string
    StorageServiceURL string
    MetadataServiceURL string
    TraceLevel string
    Command *PolarisCommand
    Logger *log.Logger
    TotalTasks int
    ActiveTasks int
    Timeout chan int
}

type PolarisFile struct {
    Path string
    ContentLenght int
    Etag string
    ContentType string
    LastModified string
    UUID string
}

type FileOps interface {
    UploadDir(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error)
    ListFile(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error)
    UploadFile(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error)
    DeleteFile(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error)
    DeleteAllFiles(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error)
}

type MetadataOps interface {
    IndexDocument(esConn goes.Connection, d goes.Document, extraArgs url.Values, ch chan *goes.Response) (err error)
    DeleteDocument(esConn goes.Connection, d goes.Document, extraArgs url.Values, ch chan *goes.Response) (err error)
}

/**Display stats of the test
 * @begin the begin time for Stat
 * @end the end time for Stat
 */
func (c *PolarisClient) Stat(taskname string, begin, end time.Time) (err error) {
    if end.Sub(begin) < 0 {
        err = errors.New("End time should later than begin time")
    }
    if err == nil{
        completed := c.TotalTasks - c.ActiveTasks
        duration := float64(end.Sub(begin).Nanoseconds())
        Parallel := runtime.NumCPU()

        Pinfo(c.Logger, "%s: %d %s %.9f %s\n", taskname, completed, "Tasks Completed! Elapsed: ", duration/1E9, "seconds")
        fmt.Printf("Concurrency: %.6f, Parallel: %d\n", float64(c.TotalTasks)*1E9/duration, Parallel)
        c.Logger.Printf("Concurrency: %.6f, Parallel: %d\n", float64(c.TotalTasks)*1E9/duration, Parallel)
    }

    return 
}

func (c *PolarisClient) Info() {
    c.Logger.Println("clientId: ", c.ClientId)
    c.Logger.Println("users ", c.Users)
    c.Logger.Printf("log level: %s\n", c.TraceLevel)
    c.Logger.Printf("total tasks: %d\n", c.TotalTasks)
    c.Logger.Printf("active tasks: %d\n", c.ActiveTasks)
    c.Logger.Printf("command: {%s:%d}\n", c.Command.Command, c.Command.Status)
    c.Logger.Println("storage service: ", c.StorageServiceURL)
    c.Logger.Println("metadata service: ", c.MetadataServiceURL)
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
func (c *PolarisClient)Init(cfg *config.PolarisConfig, cmd string, logger *log.Logger, tasks int, timeout int) (errs []error) {

    s, t1 := Trace(GetFunctionName(c.Init))
    defer Un(s, t1)
    
    //1. Update config from os env
    cfg.UpdateConfigFile("")

    //2. Update others
    testCmd := new(PolarisCommand)
    testCmd = &PolarisCommand{cmd, WAITING}
    timeoutCh := make(chan int)
    
    if timeout > 0 {
        go func(){
            time.Sleep(time.Duration(timeout) * 1000 * time.Millisecond)
            timeoutCh <- 1
        }()
    }

    *c = PolarisClient{cfg.ClientId, cfg.Users, cfg.StorageServiceURL, cfg.MetadataServiceURL, strings.ToLower(cfg.TraceLevel), testCmd, logger, tasks, 0, timeoutCh}

    if c.Logger == nil {
        errs = append(errs, errors.New("logger of client is not set"))
    }
    if len(c.ClientId) == 0 {
        errs = append(errs, errors.New("client id is not set"))
    }
    if len(c.Users) == 0 {
        errs = append(errs, errors.New("user id or token is not set correctly"))
    }
    if len(c.TraceLevel) == 0 {
        c.Logger.Println("Tracelevle is default: info")
    }
    if len(c.StorageServiceURL) == 0 && len(c.MetadataServiceURL) == 0 {
        errs = append(errs, errors.New("please set any of services:[storage,metadata]"))
    }

    if c.Command == nil {
        errs = append(errs, errors.New("No commands set!"))
    } else {
        _, ok := FindElementInArray(Commands, cmd)
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
    default:
        return fmt.Errorf("Error: response == %d %s", resp.StatusCode, resp.Status)
	}
}

/**Upload a given directory to storage
 * @param userch chan for pass user Info
 * @param ch chan for pass *http.Response
 * @user user id
 * @token token for user
 * @args... variables
 */
 func (c *PolarisClient) UploadDir(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error) {
     dir := args[0].(string)
     fileInfo, err := os.Stat(dir)
     Perr(c.Logger, err, true)
     if fileInfo.IsDir() == false {
         c.Logger.Printf("%s is not a directory", dir)
         os.Exit(1)
     } 

     // Upload a directory
     walker, err := GetDirAndFileList(dir)
     Perr(c.Logger, err,true)
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
         go FileTask(c.UploadFile, nil, ch, user, token, filename)
     }
     t2 := time.Now()
     c.Stat(GetFunctionName(c.UploadDir), t1, t2)
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
         if userch != nil {
             userch  <- user
         }
     }

     return
}

/**Upload File(s) to polaris storage
 * @param userch chan for pass user Info
 * @param ch chan for pass *http.Response
 * @user user id
 * @token token for user
 * @args... variables
 */
 func (c *PolarisClient) UploadFile(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error) {

     method := "PUT"
     var ok bool = false
     var path, url string
     var headers map[string]string
     headers = make(map[string]string)
     headers["Authorization"] = "Bearer " + token
     headers["Content-type"] = "text/plain"

     for _, arg := range args {
         if path, ok = arg.(string); ok {
             url = c.StorageServiceURL + "/" + user + "/files/" + filepath.Base(path) + "?previous="
         }
     }
     fContent, err := ioutil.ReadFile(path)
     Perr(c.Logger, err, true)
     c.Logger.Printf("url: %s\nmethod: %s\n", url, method)
     if strings.ToLower(c.TraceLevel) == "debug" {
         for k, v := range headers {
             fmt.Printf("%s: %s\n", k, v)
             c.Logger.Printf("%s: %s\n", k, v)
         }
     }

     c.ActiveTasks++
     r, err := CallAPI(method, url, &fContent, headers)
     Perr(c.Logger, err, false)

     err = CheckHttpResponseStatusCode(r)
     Perr(c.Logger, err, false)
     ch <- r

     if c.Command.Status != UNKOWN {
         c.Command.Status = DONE
         if userch != nil {
             userch  <- user
         }
     }
     return
 }

/**Delete a file from Storage
 * @param userch chan for pass user Info
 * @param ch chan for pass *http.Response
 * @user user id
 * @token token for user
 * @args... variables [filepath, block?]
 */
func (c *PolarisClient) DeleteFile(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error) {

    method := "DELETE"
    var headers map[string]string
    var ok, block bool = false, false
    headers = make(map[string]string)

    headers["Authorization"] = "Bearer " + token
    url := c.StorageServiceURL + "/" + user + "/files/"
    // args[0] pass in the file name to delete
    // args[1] pass in the block directive for all files deletion
    if args != nil {
        if _, ok = args[0].(string); ok {
            url = url + args[0].(string)
        }
        if block, ok = args[1].(bool); ok {
            block = args[1].(bool)
        }
    }
    c.ActiveTasks++
    r, err := CallAPI(method, url, nil, headers)
    Perr(c.Logger, err, false)
    err = CheckHttpResponseStatusCode(r)
    Perr(c.Logger, err, false)
    ch <- r

    if c.Command.Status != UNKOWN {
        c.Command.Status = DONE
        if userch != nil && block == false {
            userch  <- user
        }
    }
    return
}

/**Delete all files of a user from Storage
 * @param userch chan for pass user Info
 * @param ch chan for pass *http.Response
 * @user user id
 * @token token for user
 * @args... variables [filelist]
 */
func(c *PolarisClient) DeleteAllFiles(userch chan string, ch chan *http.Response, user, token string, args... interface{}) (err error) {

    var fileList []string
    // args[0] pass in the file list to delete
    if args != nil {
        if l, ok := args[0].([]string); ok {
            fileList = l
        }
    }
    t1 := time.Now()
    for _, f := range fileList {
        go c.DeleteFile(userch, ch, user, token, strings.TrimPrefix(f, "/"), true)
    }
    t2 := time.Now()
    defer c.Stat(GetFunctionName(c.DeleteFile), t1, t2)
    for j := 0; j < len(fileList); j++ {
        select {
        case r := <- ch:
            c.ActiveTasks--
            if c.TraceLevel == "debug" {
                fmt.Println(r)
                c.Logger.Println(r)
            }
        }
    }
    userch <- user
    return
}

/**List all files of a user from Storage
 * @param userch chan for pass user Info
 * @param ch chan for pass *http.Response
 * @user user id
 * @token token for user
 * @args... variables
 */
func (c *PolarisClient) ListFile(userch chan string, ch chan *http.Response, user, token string, args... interface{})(err error) {

    var limit int
    var marker string
    var ok bool
    method := "GET"
    var headers map[string]string
    headers = make(map[string]string)


    headers["Authorization"] = "Bearer " + token
    url := c.StorageServiceURL + "/" + user + "/files"

    for _, arg := range args {
        if limit, ok = arg.(int); ok {
            if limit > 0 {
                url = url + "?limit=" + strconv.Itoa(limit)
            }
        } else if marker, ok = arg.(string); ok {
            url = url + "&marker=" + marker
        }
    }
    c.ActiveTasks++
    r, err := CallAPI(method, url, nil, headers)
    Perr(c.Logger, err, false)

    err = CheckHttpResponseStatusCode(r)
    Perr(c.Logger, err, false)
    ch <- r

    if c.Command.Status != UNKOWN {
        c.Command.Status = DONE
        if userch != nil {
            userch  <- user
        }
    }

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

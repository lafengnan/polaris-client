package utils 

import (
    "os"
    "fmt"
    "log"
    "errors"
    "bytes"
    "time"
    "runtime"
    "reflect"
    "net/http"
    "net/url"
    "io/ioutil"
    "path/filepath"
    "strings"
    "github.com/belogik/goes"
)

var Commands []string  = []string {
    "UploadFile",
    "DeleteFile",
    "ListFile",
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
    TaskCount int
    Timeout chan int
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
func (c *PolarisClient)Init(clientId, UserId, token, stVC, mdVC, traceLevel string, cmd *PolarisCommand, logger *log.Logger, timeoutCh chan int) (errs []error) {

    s, t1 := Trace(GetFunctionName(c.Init))
    defer Un(s, t1)

    *c = PolarisClient{clientId, UserId, token, stVC, mdVC, strings.ToLower(traceLevel), cmd, logger, 0, timeoutCh}

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
                c.TaskCount--
                if c.TraceLevel == "debug" {
                    fmt.Println(r)
                    c.Logger.Println(r)
                }
            }
        }
        if c.Command.Status != UNKOWN {
            c.Command.Status = DONE
        }

        defer fmt.Println(len(walker.Files) - c.TaskCount, "Files Uploaded")
        defer c.Logger.Println(len(walker.Files) - c.TaskCount, "Files Uploaded")
        defer fmt.Printf("Concurrency: %d, Paralell: %d\n", int64(len(walker.Files))*1E9/(t2.Sub(t1).Nanoseconds()), runtime.NumCPU())
        defer c.Logger.Printf("Concurrency: %d, Paralell: %d\n", int64(len(walker.Files))*1E9/(t2.Sub(t1).Nanoseconds()), runtime.NumCPU())
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

     c.TaskCount++
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
         c.Logger.Fatal(err)
     }

     err = CheckHttpResponseStatusCode(r)
     if err != nil {
         if 401 == r.StatusCode {
             fmt.Println(err)
             c.Logger.Fatal(err)
         } else {
             c.Logger.Fatal(err)
         }
     }
     ch <- r
     return
 }

func (c *PolarisClient) DeleteFile(path string, ch chan *http.Response) (err error) {
    return
}

func (c *PolarisClient) ListFile(path string, ch chan *http.Response)(err error) {
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

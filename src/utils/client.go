package utils 

import (
    "fmt"
    "log"
    "errors"
    "net/http"
    "net/url"
    "io/ioutil"
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

type PolarisClient struct {
    ClientId string
    UserId string
    Token string
    StorageServiceURL string
    MetadataServiceURL string
    TraceLevel string
    Commands []string
    Logger *log.Logger
}

type FileOps interface {
    UploadFile(path string, url string) (ch chan http.Response, err error)
    DeleteFile(path string, url string) (ch chan http.Response, err error)
    ListFile(url string) (ch chan http.Response, err error)
}

type MetadataOps interface {
    IndexDocument(esConn goes.Connection, d goes.Document, extraArgs url.Values) (ch chan goes.Response, err error)
    DeleteDocument(esConn goes.Connection, d goes.Document, extraArgs url.Values) (ch chan goes.Response, err error)
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
func (c *PolarisClient)Init(clientId, UserId, token, stVC, mdVC, traceLevel string, cmds []string, logger *log.Logger) (errs []error) {

    s, t1 := Trace(GetFunctionName(c.Init))
    defer Un(s, t1)

    *c = PolarisClient{clientId, UserId, token, stVC, mdVC, strings.ToLower(traceLevel), cmds, logger}

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

    if len(c.Commands) == 0 {
        errs = append(errs, errors.New("No commands set!"))
    } else {
        for _, cmd := range c.Commands {
            _, ok := FindElementInArray(Commands, cmd)
            if ok == false {
                errs = append(errs, errors.New("Wrong Command"))
            }
        }
    }

    return  
}

/**Upload File(s) to polaris storage
 * @param path the file(s) to upload 
 * @param url the API uri
 * @param traceLevel the log level
 * @param ch the chan to transit http.Response
 */
 func (c *PolarisClient) UploadFile(path string, url string) (ch chan *http.Response, err error) {

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

func (c *PolarisClient) DeleteFile(path string, url string) (ch chan http.Response, err error) {
    return
}

func (c *PolarisClient) ListFile(url string)(ch chan http.Response, err error) {
    return 
}

func (c *PolarisClient) IndexDocument(es goes.Connection, d goes.Document, extraArgs url.Values) (ch chan goes.Response, err error) {
    return
}

func (c *PolarisClient)DeleteDocument(es goes.Connection, d goes.Document, extraArgs url.Values) (ch chan goes.Response, err error) {
    return
}



package main

import (
    "os"
    "log"
    "io/ioutil"
    "runtime"
    "flag"
    "net/http"
    "path/filepath"
    "polaris/util"
)

const (
    MAXFILE = 5
)

var (
    logFileName = flag.String("log", "client.log","log file name" )
)


var ch chan http.Response 

type Walker struct {
    dirs []string
    files []string
}

func UploadFile(path string, url string) error {
    var headers map[string] string
    headers = make(map[string] string)
    headers["Authorization"] = "Bearer " + os.Getenv("TOKEN")
    headers["Content-type"] = "text/plain"

    fContent, err := ioutil.ReadFile(path)
    if err != nil {
        log.Fatal(err)
    }
    

    log.Println("url: ", url)
    for k, v := range headers {
        log.Printf("%s: %s", k, v)
    }
    res, err := util.CallAPI("PUT", url, &fContent, headers)
    
    if  err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Upload %s, Response: %s\n", filepath.Base(path), res.Status)
    ch <- *res
    
    return err
}

func main() {

    runtime.GOMAXPROCS(runtime.NumCPU())
    flag.Parse()

    logFile, logErr := os.OpenFile(*logFileName, os.O_CREATE|os.O_RDWR, 0666)
    if logErr != nil {
        log.Fatal("Fail to open log file")
    }
    log.SetOutput(logFile)
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

    userId := os.Getenv("USER_ID")
    token := os.Getenv("TOKEN")
    clientId := os.Getenv("CLIENT_ID")
    storageService := os.Getenv("STORAGE_SVC")
    metadataService := os.Getenv("MD_SVC")

    log.Println("user_id: ", userId)
    log.Println("clientId: ", clientId)
    log.Println("storage service: ", storageService)
    log.Println("metadata service: ", metadataService)
    log.Println("token: ", token)

    dir := "/home/panzhongbin/Beta/go_source/src/polaris/"
    walker := new(Walker)
    filepath.Walk(dir,
    func(path string, fi os.FileInfo, err error) error {
        if fi == nil {
            return err
        }
        if fi.IsDir() {
            log.Printf("Found directory: %s", path)
            walker.dirs = append(walker.dirs, path)

        } else {
            log.Printf("Found Files: %s", path)
            walker.files = append(walker.files, path)
        }
        return nil
    })

    ch = make(chan http.Response, len(walker.files))
    for _, filename := range walker.files {
        go UploadFile(filename, storageService+"/"+userId+"/files/"+filepath.Base(filename)+"?previous=")
    }

    i := 0
    LOOP: for {
        select {
        case <-ch:
                i++
                if i == len(walker.files){
                    break LOOP
                }
                
        }
    }
}

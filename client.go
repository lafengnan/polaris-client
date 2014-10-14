package main

import (
    "os"
    "fmt"
    "log"
    "runtime"
    "flag"
    "time"
    "net/http"
    "runtime/pprof"
    "path/filepath"
    "polaris/util"
)

var (
    logFileName = flag.String("log", "client.log", "log file name" )
    dir = flag.String("d", "", "source directory")
    traceLevel = flag.String("level", "Info", "trace level")
    cpuProfile = flag.String("cpuprofile", "", "write profile to file")
)


var ch chan http.Response 


func main() {

    runtime.GOMAXPROCS(runtime.NumCPU())
    flag.Parse()
    logFile, logErr := os.OpenFile(*logFileName, os.O_CREATE|os.O_RDWR, 0666)
    if logErr != nil {
        log.Fatal("Fail to open log file")
    }
    log.SetOutput(logFile)
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

    if *cpuProfile != "" {
        f, err := os.Create(*cpuProfile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }
    log.Printf("source directory: %s\n", *dir)
    log.Printf("log file: %s\n", *logFileName)
    log.Printf("log level: %s\n", *traceLevel)

    userId := os.Getenv("USER_ID")
    token := os.Getenv("TOKEN")
    clientId := os.Getenv("CLIENT_ID")
    storageService := os.Getenv("STORAGE_SVC")
    metadataService := os.Getenv("MD_SVC")

    if len(storageService) == 0 || len(token) == 0 || len(userId) == 0 {
        fmt.Println("Service error!")
        log.Fatal("Service error!")
    }

    log.Println("user_id: ", userId)
    log.Println("clientId: ", clientId)
    log.Println("storage service: ", storageService)
    log.Println("metadata service: ", metadataService)
    log.Println("token: ", token)

    walker := new(util.Walker)
    err := filepath.Walk(*dir,
    func(path string, fi os.FileInfo, err error) error {
        if fi == nil {
            fmt.Println(err)
            log.Println(err)
            return err
        }
        if fi.IsDir() {
            log.Printf("Found directory: %s\n", path)
            walker.Dirs = append(walker.Dirs, path)

        } else {
            log.Printf("Found Files: %s\n", path)
            walker.Files = append(walker.Files, path)
        }
        return nil
    })

    if err != nil {
        return 
    }

    ch = make(chan http.Response, len(walker.Files))
    fmt.Printf("Preapare to upload %d files\n", len(walker.Files))
    log.Printf("Preapare to upload %d files\n", len(walker.Files))
    
    t1 := time.Now()
    for _, filename := range walker.Files {
        go util.UploadFile(filename, storageService+"/"+userId+"/files/"+filepath.Base(filename)+"?previous=", *traceLevel, ch)
    }

    t2 := time.Now()
    fmt.Printf("Concurrency: %d\n", int64(len(walker.Files))*1E9/(t2.Sub(t1).Nanoseconds()))
    log.Printf("Concurrency: %d\n", int64(len(walker.Files))*1E9/(t2.Sub(t1).Nanoseconds()))

    i := 0
    LOOP: for {
        select {
        case <-ch:
                i++
                if i == len(walker.Files){
                    break LOOP
                }
                
        }
    }
}

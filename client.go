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
    dir = flag.String("source", "", "source directory")
    concurrencyNum = flag.Int("n", 1, "Concurrency number")
    traceLevel = flag.String("level", "Info", "trace level")
    cpuProfile = flag.String("cpuprofile", "", "write profile to file")
)



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
    

    userId := os.Getenv("USER_ID")
    token := os.Getenv("TOKEN")
    clientId := os.Getenv("CLIENT_ID")
    storageService := os.Getenv("STORAGE_SVC")
    metadataService := os.Getenv("MD_SVC")

    if len(storageService) == 0 || len(token) == 0 || len(userId) == 0 {
        fmt.Println("Fatal: Service error!")
        log.Fatal("Fatal: Service error!")
    }

    log.Printf("source directory: %s\n", *dir)
    log.Printf("log file: %s\n", *logFileName)
    log.Printf("log level: %s\n", *traceLevel)
    log.Println("user_id: ", userId)
    log.Println("clientId: ", clientId)
    log.Println("storage service: ", storageService)
    log.Println("metadata service: ", metadataService)
    log.Println("token: ", token)
    log.Println("-----\n\n")

   
    walker, err := util.GetDirAndFileList(*dir)

    if err != nil {
        return 
    }
    if *traceLevel == "debug" {
        for _, f := range walker.Files {
            fmt.Println(f)
            log.Println(f)
        }
    }

    ch := make(chan http.Response, len(walker.Files))
    fmt.Printf("Preapare to upload %d files\n", len(walker.Files))
    log.Printf("Preapare to upload %d files\n", len(walker.Files))
    
    t1 := time.Now()
    if *concurrencyNum > 1 && len(walker.Files) == 1 {
            filename := walker.Files[0]
            url := storageService + "/" + userId + "/files/" + filepath.Base(filename) + "?previous="
        for j := 0; j < *concurrencyNum; j++ {
            go util.Task(util.UploadFile, filename, url, *traceLevel, ch)
        }
    } else {
        for _, filename := range walker.Files {
            url := storageService + "/" + userId + "/files/" + filepath.Base(filename) + "?previous="
            go util.Task(util.UploadFile, filename, url, *traceLevel, ch)
        }
    }
    
    t2 := time.Now()
    fmt.Printf("Concurrency: %d, Paralell: %d\n", int64(len(walker.Files))*1E9/(t2.Sub(t1).Nanoseconds()), runtime.NumCPU())
    log.Printf("Concurrency: %d, Paralell: %d\n", int64(len(walker.Files))*1E9/(t2.Sub(t1).Nanoseconds()), runtime.NumCPU())


    completeCount := 0
    waitNum := *concurrencyNum
    if waitNum == 1 {
        waitNum = len(walker.Files)
    }
    for i := 0; i < waitNum; i++ {
        select {
        case r := <-ch:
            completeCount++
            if *traceLevel == "debug" {
                fmt.Println(r)
                log.Println(r)
            }
        }
    }
    fmt.Println(completeCount, "File Uploaded")
    log.Println(completeCount, "File Uploaded")
    
}

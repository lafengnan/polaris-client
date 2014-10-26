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
    "utils"
)

var (
    dirToUpload = flag.String("f", "", "Files/Dircs to upload")
    concurrencyNum = flag.Int("n", 1, "Concurrency number")
    logFileName = flag.String("log", "client.log", "log file name" )
    traceLevel = flag.String("level", "Info", "trace level")
    cpuProfile = flag.String("cpuprofile", "", "write profile to file")
    timeout = flag.Int("t", 0, "timeout value for waiting")
    cmd = flag.String("c", "help", "command to execute")
)



func main() {

    runtime.GOMAXPROCS(runtime.NumCPU())
    flag.Parse()
    if *cmd == "help" {
        flag.PrintDefaults()
        os.Exit(1)
    }
    logFile, logErr := os.OpenFile(*logFileName, os.O_CREATE|os.O_RDWR, 0666)
    if logErr != nil {
        log.Fatal("Fail to open log file")
    }
    log.SetOutput(logFile)
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
    logger := log.New(logFile, "polaris-client ", log.Flags())

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

    client := new(utils.PolarisClient)
    errs := client.Init(clientId, userId, token, storageService, metadataService, *traceLevel, *cmd, logger)

    if len(errs) > 0 {
        for i, err := range errs {
            if i == len(errs) - 1 {
                log.Fatal(err)
            } else {
                log.Println(err)
            }
        }
    }

    client.Logger.Printf("log file: %s\n", *logFileName)
    client.Logger.Printf("log level: %s\n", *traceLevel)
    if *concurrencyNum > 1 {
        client.Logger.Printf("concurrency: %d\n", *concurrencyNum)
    } else if *concurrencyNum == 1 {
        client.Logger.Println("concurrency: To be calculated!")
    }
    client.Logger.Printf("Timeout : %d seconds\n", *timeout)
    client.Logger.Println("user_id: ", userId)
    client.Logger.Println("clientId: ", clientId)
    client.Logger.Println("storage service: ", storageService)
    client.Logger.Println("metadata service: ", metadataService)
    client.Logger.Println("token: ", token)
    client.Logger.Println("-----\n\n")


    timeoutCh := make(chan int)

    if *timeout > 0 {
        go func(){
            time.Sleep(time.Duration(*timeout) * 1000 * time.Millisecond)
            timeoutCh <- 1
        }()
    }
    switch client.Command {
    case "UploadFile":
        client.Logger.Printf("source directory/file: %s\n", *dirToUpload)
        var t1 time.Time
        var ch chan http.Response
        fileInfo, err := os.Stat(*dirToUpload)
        if err != nil {
            client.Logger.Fatal(err)
        }
        if fileInfo.IsDir() {
            client.UploadDir(*dirToUpload, timeoutCh)
        } else {
            url := client.StorageServiceURL + "/" + userId + "/files/" + filepath.Base(*dirToUpload) + "?previous="
            ch = make(chan http.Response, *concurrencyNum)
            
            t1 = time.Now()
            for j := 0; j < *concurrencyNum; j++ {
                go utils.Task(utils.UploadFile, *dirToUpload, url, *traceLevel, ch)
            }
            t2 := time.Now()
            waitNum := *concurrencyNum
            completeCount := 0
            for i := 0; i < waitNum; i++ {
                select {
                case <- timeoutCh:
                    fmt.Println("Timeout!")
                    client.Logger.Println("Timeout!")
                    break
                case r := <- ch:
                    completeCount++
                    if client.TraceLevel == "debug" {
                        fmt.Println(r)
                        client.Logger.Println(r)
                    }
                }
            }

            defer fmt.Println(completeCount, "Files Uploaded")
            defer client.Logger.Println(completeCount, "Files Uploaded")
            defer fmt.Printf("Concurrency: %d, Paralell: %d\n", int64(*concurrencyNum)*1E9/(t2.Sub(t1).Nanoseconds()), runtime.NumCPU())
            defer client.Logger.Printf("Concurrency: %d, Paralell: %d\n", int64(*concurrencyNum)*1E9/(t2.Sub(t1).Nanoseconds()), runtime.NumCPU())
        }

    }
}

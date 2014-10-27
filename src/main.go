package main

import (
    "os"
    "fmt"
    "log"
    "runtime"
    "flag"
    "time"
    "io/ioutil"
    "net/http"
    "encoding/json"
    "runtime/pprof"
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
    testCmd := new(utils.PolarisCommand)
    testCmd.Command = *cmd
    testCmd.Status = utils.WAITTING
    timeoutCh := make(chan int)
    totalTasks := 0


    if len(*dirToUpload) != 0 {
        w, err := utils.GetDirAndFileList(*dirToUpload)
        if err != nil {
            fmt.Println(err)
            client.Logger.Fatal(err)
        }

        if len(w.Files) > 0 && *concurrencyNum > 1 {
            log.Fatal("Error config! directory upload and concurrencyNum should not be configure together")
        }
        if len(w.Files) > 0 {
            totalTasks = len(w.Files)
        } else {
            totalTasks = *concurrencyNum
        }
    } else if *cmd == "ListFiles" {
        totalTasks = 1
    }
    
    
    errs := client.Init(clientId, userId, token, storageService, metadataService, *traceLevel, testCmd, logger, totalTasks, timeoutCh)

    if len(errs) > 0 {
        for i, err := range errs {
            fmt.Println(err)
            if i == len(errs) - 1 {
                log.Fatal(err)
            } else {
                log.Println(err)
            }
        }
    }

    client.Logger.Printf("log file: %s\n", *logFileName)
    client.Logger.Printf("timeout : %d seconds\n", *timeout)
    client.Info()

    if *timeout > 0 {
        go func(){
            time.Sleep(time.Duration(*timeout) * 1000 * time.Millisecond)
            timeoutCh <- 1
        }()
    }
    switch client.Command.Command {
    case "ListFiles":
        var ch chan *http.Response
        ch = make(chan *http.Response)
        go utils.FileTask(client.ListFile, "", ch)
        select {
        case <- timeoutCh:
            fmt.Println("Timeout!")
            client.Logger.Println("Timeout!")
            client.Command.Status = utils.UNKOWN
        case r := <- ch:
            client.ActiveTasks--
            var flist  []utils.ReturnedFile
            defer r.Body.Close()
            body, err := ioutil.ReadAll(r.Body)
            if err != nil {
                fmt.Println(err)
                client.Logger.Println(err)
            }
            err = json.Unmarshal(body, &flist) 
            if err != nil {
                fmt.Println(err)
                client.Logger.Println(err)
            }
            for _, f := range flist {
                fmt.Println(f.Path)
                client.Logger.Println(f.Path)
            }
        }
    case "UploadFile":
        client.Logger.Printf("source directory/file: %s\n", *dirToUpload)
        var ch chan *http.Response
        fileInfo, err := os.Stat(*dirToUpload)
        if err != nil {
            client.Logger.Fatal(err)
        }
        client.Command.Status = utils.RUNNING
        if fileInfo.IsDir() {
            utils.FileTask(client.UploadDir, *dirToUpload, ch)
        } else {
            ch = make(chan *http.Response, *concurrencyNum)
            t1 := time.Now()
            for j := 0; j < *concurrencyNum; j++ {
                go utils.FileTask(client.UploadFile, *dirToUpload, ch)
            }
            t2 := time.Now()
            waitNum := *concurrencyNum
            for i := 0; i < waitNum; i++ {
                select {
                case <- timeoutCh:
                    fmt.Println("Timeout!")
                    client.Logger.Println("Timeout!")
                    client.Command.Status = utils.UNKOWN
                    break
                case r := <- ch:
                    client.ActiveTasks--
                    if client.TraceLevel == "debug" {
                        fmt.Println(r)
                        client.Logger.Println(r)
                    }
                }
            }
            if client.Command.Status == utils.RUNNING{
                client.Command.Status = utils.DONE
            }

            defer client.Stat(t1, t2)
        }

    }
}

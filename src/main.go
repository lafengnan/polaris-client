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
    utils.Perr(nil, logErr, true)
    log.SetOutput(logFile)
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
    logger := log.New(logFile, "polaris-client ", log.Flags())

    if *cpuProfile != "" {
        f, err := os.Create(*cpuProfile)
        utils.Perr(nil, err, true)
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }


    client := new(utils.PolarisClient)
    totalTasks := 0

    if len(*dirToUpload) != 0 {
        w, err := utils.GetDirAndFileList(*dirToUpload)
        utils.Perr(client.Logger, err, true)
        if len(w.Files) > 0 && *concurrencyNum > 1 {
            log.Fatal("Error config! directory upload and concurrencyNum should not be configure together")
        }
        if len(w.Files) > 0 {
            totalTasks = len(w.Files)
        } else {
            totalTasks = *concurrencyNum
        }
    } else {
        totalTasks = *concurrencyNum
    }
    
    
    errs := client.Init(*traceLevel, *cmd, logger, totalTasks, *timeout)

    if len(errs) > 0 {
        fatal := false
        for i, err := range errs {
            if i == len(errs) - 1 {
                fatal = true
            }
            utils.Perr(nil, err, fatal)
        }
    }

    client.Logger.Printf("log file: %s\n", *logFileName)
    client.Logger.Printf("timeout : %d seconds\n", *timeout)
    client.Info()

    
    switch client.Command.Command {
    case "ListFiles":
        var ch chan *http.Response
        ch = make(chan *http.Response)
        for i := 0; i < client.TotalTasks; i++ {
            go utils.FileTask(client.ListFile, ch, 210)
        }
        for i := 0; i <client.TotalTasks; i++ {
            select {
            case <- client.Timeout:
                fmt.Println("Timeout!")
                client.Logger.Println("Timeout!")
                client.Command.Status = utils.UNKOWN
            case r := <- ch:
                var flist  []utils.PolarisFile
                defer r.Body.Close()
                body, err := ioutil.ReadAll(r.Body)
                utils.Perr(client.Logger, err, false)
                err = json.Unmarshal(body, &flist) 
                utils.Perr(client.Logger, err, false)
                utils.Pinfo(client.Logger, "%d %s", len(flist), "files")
                for _, f := range flist {
                    utils.Pinfo(client.Logger, "%s %s %s %s", f.Path, f.Etag, f.UUID, f.LastModified)
                }
            }
        }
    case "UploadFile":
        client.Logger.Printf("source directory/file: %s\n", *dirToUpload)
        var ch chan *http.Response
        fileInfo, err := os.Stat(*dirToUpload)
        utils.Perr(client.Logger, err, true)
        client.Command.Status = utils.RUNNING
        if fileInfo.IsDir() {
            utils.FileTask(client.UploadDir, ch, *dirToUpload)
        } else {
            ch = make(chan *http.Response, *concurrencyNum)
            t1 := time.Now()
            for j := 0; j < *concurrencyNum; j++ {
                go utils.FileTask(client.UploadFile, ch, *dirToUpload)
            }
            t2 := time.Now()
            waitNum := *concurrencyNum
            for i := 0; i < waitNum; i++ {
                select {
                case <- client.Timeout:
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

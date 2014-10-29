package main

import (
    "os"
    "fmt"
    "log"
    "time"
    "flag"
    "errors"
    "runtime"
    "strings"
    "io/ioutil"
    "net/http"
    "encoding/json"
    "runtime/pprof"
    "utils"
    "config"
)

var (
    file = flag.String("f", "", "Files/Dircs to upload")
    concurrencyNum = flag.Int("n", 1, "Concurrency number")
    logFileName = flag.String("log", "client.log", "log file name" )
    ConfigFile = flag.String("config", "./.config", "config file")
    cpuProfile = flag.String("cpuprofile", "", "write profile to file")
    timeout = flag.Int("t", 0, "timeout value for waiting(seconds)")
    cmd = flag.String("c", "help", "command to execute")
)



func main() {

    runtime.GOMAXPROCS(runtime.NumCPU())
    flag.Parse()
    cfg := new(config.PolarisConfig)
    cfg.ReadConfig(*ConfigFile)

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

    if len(*file) != 0  && strings.ToLower(*cmd) == "uploadfile"{
        w, err := utils.GetDirAndFileList(*file)
        utils.Perr(client.Logger, err, true)
        if len(w.Files) > 0 && *concurrencyNum > 1 {
            log.Fatal("Error config! directory upload and concurrencyNum should not be configure together")
        }
        if len(w.Files) > 0 {
            totalTasks = len(w.Files) * len(cfg.Users)
        } else {
            totalTasks = *concurrencyNum * len(cfg.Users)
        }
    } else {
        totalTasks = *concurrencyNum * len(cfg.Users)
    }

    errs := client.Init(cfg, *cmd, logger, totalTasks, *timeout)

    if len(errs) > 0 {
        fatal := false
        for i, err := range errs {
            if i == len(errs) - 1 {
                fatal = true
            }
            utils.Perr(nil, err, fatal)
        }
    }

    utils.Pinfo(client.Logger, "%s %s\n", "log files: ", *logFileName)
    utils.Pinfo(client.Logger, "%s %d %s\n", "timeout :", *timeout, "seconds")
    client.Info()

    
    var t1, t2 time.Time
    var taskname string
    userch := make(chan string, len(client.Users))
    var ch chan *http.Response
    switch client.Command.Command {
    case "DeleteFile":
        if len(*file) == 0 {
            err := errors.New("No source file to delete!")
            utils.Perr(client.Logger, err, true)
        }
        ch = make(chan *http.Response, *concurrencyNum)
        taskname = utils.GetFunctionName(client.DeleteFile)
        t1 = time.Now()
        for u, t := range client.Users {
            for i := 0; i < *concurrencyNum; i++ {
                go utils.FileTask(client.DeleteFile, userch, ch, u, t, *file)
            }
        }
        t2 = time.Now()
        for i := 0; i < len(client.Users); i++ {
            for j := 0; j < *concurrencyNum; j++ {
                select {
                case r := <- ch:
                    client.ActiveTasks--
                    fmt.Println(r)
                    client.Logger.Println(r)
                }
            }
            select {
            case <- client.Timeout:
                client.Command.Status = utils.UNKOWN
                utils.Pinfo(client.Logger, "%s\n", "Timeout!")
                break
            case u := <- userch:
                utils.Pinfo(client.Logger, "%s: %s\n", u, "task completed!")    
            }
        }

    case "ListFiles":
        ch = make(chan *http.Response, client.TotalTasks)
        taskname = utils.GetFunctionName(client.ListFile)
        t1 = time.Now()
        for u, t := range client.Users {
            for i := 0; i < *concurrencyNum; i++ {
                go utils.FileTask(client.ListFile, userch, ch, u, t, 210)
            }
        }
        t2 = time.Now()
        for i := 0; i < len(client.Users); i++ {
            for j := 0; j < *concurrencyNum; j++ {
                select {
                case r := <- ch:
                    client.ActiveTasks--
                    var flist  []utils.PolarisFile
                    defer r.Body.Close()
                    body, err := ioutil.ReadAll(r.Body)
                    utils.Perr(client.Logger, err, true)
                    err = json.Unmarshal(body, &flist) 
                    utils.Perr(client.Logger, err, false)
                    utils.Pinfo(client.Logger, "%d %s\n", len(flist), "files")
                    for _, f := range flist {
                        utils.Pinfo(client.Logger, "%s %s %s %s\n", f.Path, f.Etag, f.UUID, f.LastModified)
                    }
                }
            }
            select {
            case <- client.Timeout:
                client.Command.Status = utils.UNKOWN
                utils.Pinfo(client.Logger, "%s\n", "Timeout!")
                break
            case u := <- userch:
                utils.Pinfo(client.Logger, "%s: %s", u, "task completed!\n")
            }
        }
    case "UploadFile":
        utils.Pinfo(client.Logger, "%s %s\n", "upload file(s) for", client.Users)
        utils.Pinfo(client.Logger, "%s %s\n", "files source: ", *file)
        ch = make(chan *http.Response, client.TotalTasks/len(client.Users))
        fileInfo, err := os.Stat(*file)
        utils.Perr(client.Logger, err, true)
        client.Command.Status = utils.RUNNING
        if fileInfo.IsDir() {
            taskname = utils.GetFunctionName(client.UploadDir)
            t1 = time.Now()
            for u, t := range client.Users {
                go utils.FileTask(client.UploadDir, userch, ch, u, t, *file)
            }
            t2 = time.Now()
            for i := 0; i < len(client.Users); i++ {
                select {
                    case <- client.Timeout:
                        client.Command.Status = utils.UNKOWN
                        utils.Pinfo(client.Logger, "%s\n", "Timeout!")
                        break
                    case u := <- userch:
                        utils.Pinfo(client.Logger, "%s: %s", u, "task completed!\n")
                }
            }
        } else {
            taskname = utils.GetFunctionName(client.UploadFile)
            t1 = time.Now()
            for u, t := range client.Users {
                for j := 0; j < *concurrencyNum; j++ {
                    go utils.FileTask(client.UploadFile, userch, ch, u, t, *file)
                }
            }
            t2 = time.Now()
            for i := 0; i < client.TotalTasks; i++ {
                select {
                case <- client.Timeout:
                    client.Command.Status = utils.UNKOWN
                    utils.Pinfo(client.Logger, "%s\n", "Timeout!")
                    break
                case r := <- ch:
                    client.ActiveTasks--
                    if client.TraceLevel == "debug" {
                        fmt.Println(r)
                        client.Logger.Println(r)
                    }
                case u := <- userch:
                    utils.Pinfo(client.Logger, "%s: %s", u, "task completed!\n")
                }
            }
            if client.Command.Status == utils.RUNNING{
                client.Command.Status = utils.DONE
            }
        }
    }
    defer client.Stat(taskname, t1,t2)
}

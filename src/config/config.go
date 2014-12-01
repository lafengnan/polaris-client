package config

import (
    "fmt"
    "os"
    "log"
    "encoding/json"
)

type PolarisConfig struct {
    ClientId string
    Users map[string]string
    StorageServiceURL string
    MetadataServiceURL string
    TraceLevel string
    Path string
}


func (cfg *PolarisConfig) ReadConfig(path string) {

    //default values
    cfg.ClientId = "go-client"
    cfg.TraceLevel = "info"
    cfg.Path = path
    if cfg.Users == nil {
        cfg.Users = make(map[string]string)
    }
    
    file, err := os.Open(path)
    if err != nil {
        fmt.Println(err)
    }
    decoder := json.NewDecoder(file)
    err = decoder.Decode(cfg)
    if err != nil {
        fmt.Println(err)
    }
    file.Close()
}

func (cfg *PolarisConfig) UpdateConfigFile(path string) {

    if len(path) == 0 {
        path = cfg.Path
    }
    file, err := os.OpenFile(path, os.O_WRONLY, 0666)
    if err != nil {
        fmt.Println(err)
    }

    clientId := os.Getenv("CLIENT_ID")
    userId := os.Getenv("USER_ID")
    token := os.Getenv("TOKEN")
    stVC := os.Getenv("STORAGE_SVC")
    mdVC := os.Getenv("MD_SVC")

    if len(clientId) != 0 && cfg.ClientId != clientId {
        cfg.ClientId = clientId
    } 
    if _, ok := cfg.Users[userId]; ok {
        fmt.Println(userId, "exisits, Update its token!")
        log.Println(userId, "exisits, Update its token!")
    } 
    if len(stVC) != 0 && cfg.StorageServiceURL != stVC {
        cfg.StorageServiceURL = stVC
    } 
    if len(mdVC) != 0 && cfg.MetadataServiceURL != mdVC {
        cfg.MetadataServiceURL = mdVC
    }
    if len(token) != 0 && len(userId) != 0 {
        cfg.Users[userId] = token
    }

    encoder := json.NewEncoder(file)
    err = encoder.Encode(cfg)
    if err != nil {
        fmt.Println(err)
    }
    file.Close()
}

package main 

import (
    "os"
    "fmt"
    "testing"
    "net/http"
    "path/filepath"
    "polaris/util"
)

var ch chan http.Response
var walker util.Walker

func BenchmarkFunction(b *testing.B) {

    userId := os.Getenv("USER_ID")
    storageService := os.Getenv("STORAGE_SVC")
    for i:= 0; i < b.N; i++ {
        url := storageService + "/" + userId + "/files/" + filepath.Base(walker.Files[i]) + "?previous="
        util.UploadFile(walker.Files[i], url, "info", ch)
    }
}


func main(){
    
    dir := "/home/panzhongbin/Beta/go_source/src/polaris/"
    walker := new(util.Walker)
    err := filepath.Walk(dir,
    func(path string, fi os.FileInfo, err error) error {
        if fi == nil {
            fmt.Println(err)
            return err
        }
        if fi.IsDir() {
            walker.Dirs = append(walker.Dirs, path)

        } else {
            walker.Files = append(walker.Files, path)
        }
        return nil
    })

    if err != nil {
        return 
    }
    ch = make(chan http.Response, len(walker.Files))
    fmt.Printf("Preapare to upload %d files\n", len(walker.Files))


    br := testing.Benchmark(BenchmarkFunction)
    fmt.Println(br)

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

package main

import (
    "encoding/gob"
    "log"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type FSnode struct {
    Name string
    Size int64
    ModTime time.Time
    IsDir bool
    Depth int
    Children map[string]bool
}

type FStree struct {
    Tree map[string]*FSnode
}

func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
}

func FST_parse(fst *FStree, dirname string) {
    for child, _ := range fst.Tree[dirname].Children {
        spaces(fst.Tree[dirname].Depth)
        if fst.Tree[child].IsDir {
            fmt.Println(child, ":", fst.Tree[child].ModTime)
            FST_parse(fst, child)
        } else {
            fmt.Println(fst.Tree[child].Name, "size:", fst.Tree[child].Size, "mod:", fst.Tree[child].ModTime)
        }
    }
}

func main() {

    root_folder := "watch_folder"
    dirname := "." + string(filepath.Separator) + root_folder + string(filepath.Separator)

    f, err := os.Open("FST_watch")
    defer f.Close()
    if err != nil {
        log.Println("Error opening file:", err)
    }
    var fst1 FStree
    decoder := gob.NewDecoder(f)
    decoder.Decode(&fst1)

    FST_parse(&fst1, dirname)
}

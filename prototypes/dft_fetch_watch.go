package main

import (
    "code.google.com/p/go.exp/inotify"
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

func FST_parse_watch(fst *FStree, dirname string, watcher *inotify.Watcher) {
    err := watcher.Watch(dirname)
    if err != nil {
        log.Fatal(err)
    }
    for child, _ := range fst.Tree[dirname].Children {
        spaces(fst.Tree[dirname].Depth)
        if fst.Tree[child].IsDir {
            fmt.Println(child, ":", fst.Tree[child].ModTime)
            FST_parse_watch(fst, child, watcher)
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

    watcher, err := inotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }

    FST_parse_watch(&fst1, dirname, watcher)

    for {
        select {
        case ev := <-watcher.Event:
            log.Println("event:", ev)
        case err := <-watcher.Error:
            log.Println("error:", err)
        }
    }
}

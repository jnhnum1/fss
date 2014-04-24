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
    RModTime time.Time
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

func FST_create(dirname string, depth int, fst *FStree) {
    
    d, err := os.Open(dirname)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    defer d.Close()
    fi, err := d.Readdir(-1)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    for _, fi := range fi {
        spaces(depth)
        if fi.Mode().IsRegular() {
            child_name := dirname+fi.Name()
            fmt.Println(fi.Name(), "size:", fi.Size(), "mod:", fi.ModTime())

            //fsn := FSnode{Name:fi.Name(), Size:fi.Size(), ModTime:fi.ModTime(), IsDir:fi.IsDir()}
            fst.Tree[child_name] = new(FSnode)
            fst.Tree[child_name].Name = fi.Name()
            fst.Tree[child_name].Size = fi.Size()
            fst.Tree[child_name].ModTime = fi.ModTime()
            fst.Tree[child_name].RModTime = fi.ModTime()
            fst.Tree[child_name].IsDir = fi.IsDir()
            fst.Tree[child_name].Depth = depth+1
            fst.Tree[dirname].Children[child_name] = true

            if fst.Tree[dirname].RModTime.Before(fst.Tree[child_name].RModTime) {
                fst.Tree[dirname].RModTime = fst.Tree[child_name].RModTime
            }
        } else if fi.IsDir() {
            child_name := dirname+fi.Name()+string(filepath.Separator)
            fmt.Println(child_name, ":", fi.ModTime())

            //fsn := FSnode{Name:fi.Name(), Size:fi.Size(), ModTime:fi.ModTime(), IsDir:fi.IsDir()}
            fst.Tree[child_name] = new(FSnode)
            fst.Tree[child_name].Name = fi.Name()
            fst.Tree[child_name].Size = fi.Size()
            fst.Tree[child_name].ModTime = fi.ModTime()
            fst.Tree[child_name].IsDir = fi.IsDir()
            fst.Tree[child_name].Depth = depth+1
            fst.Tree[child_name].Children = make(map[string]bool)
            fst.Tree[dirname].Children[child_name] = true
            FST_create(child_name, depth+1, fst)
            if fst.Tree[dirname].RModTime.Before(fst.Tree[child_name].RModTime) {
                fst.Tree[dirname].RModTime = fst.Tree[child_name].RModTime
            }
        }
    }
}

func main() {

    root_folder := "watch_folder"
    dirname := "." + string(filepath.Separator) + root_folder + string(filepath.Separator)

    fst := new(FStree)
    fst.Tree = make(map[string]*FSnode)
    fst.Tree[dirname] = new(FSnode)
    fst.Tree[dirname].Name = root_folder
    fst.Tree[dirname].Depth = 0
    fst.Tree[dirname].ModTime = time.Now()
    fst.Tree[dirname].RModTime = time.Now()
    fst.Tree[dirname].Children = make(map[string]bool)

    FST_create(dirname, 0, fst)

    /*
    for k,v := range fst.Tree {
        fmt.Println(k, v)
    }
    */

    f, err := os.OpenFile("FST_watch", os.O_WRONLY | os.O_CREATE, 0777)
    if err != nil {
        log.Println("Error opening file:", err)
    }

    encoder := gob.NewEncoder(f)
    encoder.Encode(fst)
    f.Close()
    fmt.Println("FST_watch dumped!")
}

package main

import (
    //"encoding/gob"
    //"log"
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

func FST_create(dirname string, depth int, fst *FStree) {

    //fst.Tree[dirname].Children = make(map[string]bool)
    
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
            fst.Tree[child_name].IsDir = fi.IsDir()
            fst.Tree[child_name].Depth = depth+1
            fst.Tree[dirname].Children[child_name] = true
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
        }
    }
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

    fst := new(FStree)
    fst.Tree = make(map[string]*FSnode)
    fst.Tree[dirname] = new(FSnode)
    fst.Tree[dirname].Name = root_folder
    fst.Tree[dirname].Depth = 0
    fst.Tree[dirname].Children = make(map[string]bool)

    FST_create(dirname, 0, fst)

    for k,v := range fst.Tree {
        fmt.Println(k, v)
    }

    fmt.Println("-----")
    FST_parse(fst, dirname)
}

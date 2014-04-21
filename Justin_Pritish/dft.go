package main

import (
    "fmt"
    "os"
    "path/filepath"
)

func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf(" ")
    }
    fmt.Printf("|- ")
}

func DFT(dirname string, depth int) {
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
        if fi.Mode().IsRegular() {
            spaces(depth)
            fmt.Println(fi.Name(), "size:", fi.Size(), "modified:", fi.ModTime())
        }
        if fi.IsDir() {
            spaces(depth)
            fmt.Println(fi.Name(), ":")
            //fmt.Println(dirname+fi.Name()+string(filepath.Separator), ":")
            DFT(dirname+fi.Name()+string(filepath.Separator), depth+1)
        }
    }
}

func main() {
    dirname := "." + string(filepath.Separator)
    DFT(dirname, 0)
}
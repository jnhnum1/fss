package main

import (
    "fmt"
    "os"
    "path/filepath"
)

func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
}


func DFT(dirname string, depth int ,str string)string {
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
//            spaces(depth)
           // fmt.Println(fi.Name(), "size:", fi.Size(), "modified:", fi.ModTime())
           str=str+fi.Name()
        }
        if fi.IsDir() {
//            spaces(depth)
            //fmt.Println(fi.Name(), ":")
  //          fmt.Println(dirname+fi.Name()+string(filepath.Separator), ":", fi.ModTime())
  			str=str+fi.Name()
            str=str+DFT(dirname+fi.Name()+string(filepath.Separator), depth+1)
        }else{
        	str=str+fi.Contents()
        }
    }
    return str
}

func main() {
    root_folder := "watch_folder"
    dirname := "." + string(filepath.Separator) + root_folder + string(filepath.Separator)
    DFT(dirname, 0)
}

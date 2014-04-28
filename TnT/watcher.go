package TnT

import (
    "code.google.com/p/go.exp/inotify"
    "log"
    "fmt"
    "time"
    "path/filepath"
    "os"
)

type FSnode struct {
    Name string
    Size int64
    FD uintptr
    ModTime time.Time
    IsDir bool
    Depth int
    Children map[string]bool
}

type FStree struct {
    Tree map[string]*FSnode
}

func file_watcher() {
  
  watcher, err := inotify.NewWatcher()
  if err != nil {
    log.Fatal(err)
  }
  err = watcher.Watch("./watch_folder")
  if err != nil {
    log.Fatal(err)
  }
  err = watcher.Watch("./watch_folder/nest1")
  if err != nil {
    log.Fatal(err)
  }
  
  for {
    select {
    case ev := <-watcher.Event:
        log.Println("event:", ev)
    case err := <-watcher.Error:
        log.Println("error:", err)
    }
  }
}



func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
}

//Creates FST_Watch with data on every file in the seached folder which gets used by FST_parse_watch below
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
            //fmt.Println(fi.Name(), "size:", fi.Size(), "mod:", fi.ModTime())
            fmt.Println(fi)

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

func FST_parse_watch(fst *FStree, dirname string, watcher *inotify.Watcher) {
    fmt.Println("in fst_parse_watch")
    err := watcher.Watch(dirname)
    if err != nil {
        log.Fatal(err)
    }
    for child, _ := range fst.Tree[dirname].Children {
        //fmt.Println("start of loop", fst.Tree[child])
        spaces(fst.Tree[dirname].Depth)
        if fst.Tree[child].IsDir {
            fmt.Println(child, ":", fst.Tree[child].ModTime)
            FST_parse_watch(fst, child, watcher)
        } else {
            fmt.Println(fst.Tree[child].Name, "size:", fst.Tree[child].Size, "mod:", fst.Tree[child].ModTime)
        }
    }
}

func FST_watch_files(fst *FStree, dirname string, watcher *inotify.Watcher){
    fmt.Println("in FST_watch_files")
    fmt.Println(fst.Tree[dirname])
    for {
      select {
      case ev := <-watcher.Event:
          //log.Println("event:", ev)
          for child,_ := range fst.Tree[dirname].Children{
            //fmt.Println(dirname+"/"+fst.Tree[child].Name, ev.Name)
            if(dirname+"/"+fst.Tree[child].Name == ev.Name){
                d,_ := os.Open(ev.Name)
                fi, _ := d.Stat()
                fmt.Println(fi.ModTime(),fst.Tree[child].ModTime)
                //d.Close()
                //fmt.Println(fst.Tree[child], fst.Tree[child].fd)
            }
          }
          
      case err := <-watcher.Error:
          log.Println("error:", err)
      }
    } 
}


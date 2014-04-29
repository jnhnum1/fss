package TnT

import (
    "code.google.com/p/go.exp/inotify"
    "log"
    "fmt"
    //"time"
    //"path/filepath"
    "os"
    "encoding/gob"
)

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


/*
func spaces(depth int) {func SetupServers(tag string) ([]*TnTServer, func()) {

    for i:=0; i<depth; i++ {
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
}*/

//Creates FST_Watch with data on every file in the seached folder which gets used by FST_parse_watch below
func (tnt *TnTServer) FST_create(dirname string, depth int) {
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
        //spaces(depth)
        child_name := dirname+fi.Name()
        fmt.Println(child_name)
        tnt.Tree.MyTree[child_name] = new(FSnode)
        tnt.Tree.MyTree[child_name].Name = fi.Name()
        tnt.Tree.MyTree[child_name].Size = fi.Size()
        tnt.Tree.MyTree[child_name].ModTime = fi.ModTime()
        tnt.Tree.MyTree[child_name].IsDir = fi.IsDir()
        tnt.Tree.MyTree[child_name].Depth = depth+1
        tnt.Tree.MyTree[child_name].VerVect = 0
        tnt.Tree.MyTree[child_name].SyncVect = 0
        tnt.Tree.MyTree[dirname].Children[child_name] = true
        if fi.IsDir() {
            tnt.Tree.MyTree[child_name].Children = make(map[string]bool)
            tnt.FST_create(child_name, depth+1)
        }

    }
}

func WritetoDisk(dirname string, tnt *TnTServer) error {
    f, err := os.OpenFile(dirname+"FST_watch", os.O_WRONLY | os.O_CREATE, 0777)
    if err != nil {
        log.Println("Error opening file:", err)
    }

    encoder := gob.NewEncoder(f)
    encoder.Encode(tnt)
    f.Close()
    fmt.Println("FST_watch dumped!")
    return nil
}

func ReadFromDisk(dirname string, tnt *TnTServer) FStree {
    //Test the watch here
    fmt.Println(dirname)
    f, err := os.Open(dirname+"FST_watch")
    if err != nil {
        log.Println("Error opening file:", err)
    }
    var fst1 FStree
    decoder := gob.NewDecoder(f)
    decoder.Decode(&fst1)

    f.Close()
    return fst1
}

//This function sets watch on all files in the directory
func (tnt *TnTServer) FST_set_watch(dirname string, watcher *inotify.Watcher) {
    fmt.Println("in fst_set_watch")

    for child, _ := range tnt.Tree.MyTree[dirname].Children {
        fmt.Println("start of loop", child)

        err := watcher.AddWatch(child, IN_MODIFY | IN_CREATE | IN_DELETE)
        if err != nil {
            log.Fatal(err)
        }

        if tnt.Tree.MyTree[child].IsDir {
            tnt.FST_set_watch(child, watcher)
        } 
    }
}

//This function watches all of the files in the background and takes action accordingly
func (tnt *TnTServer) FST_watch_files(dirname string, watcher *inotify.Watcher){
    fmt.Println("in FST_watch_files")
    //fmt.Println(tnt.Tree.MyTree[dirname])
    for {
        select {
            case ev := <-watcher.Event:
                //fmt.Println(ev)
                file_node := tnt.Tree.MyTree[ev.Name]
                
                fmt.Println("ev: ", ev, "file node: ", file_node)
            case err := <-watcher.Error:
                log.Println("error:", err)
        }
    } 
}

//This function is used to recursively parse the tree to find the file that set off an event in FST_watch_files
//And return its node in the tree
func (tnt *TnTServer) FST_parse_watch(dirname string, ev *inotify.Event) *FSnode {
    //fmt.Println("in fst_parse_watch")

    for child, _ := range tnt.Tree.MyTree[dirname].Children {
        if tnt.Tree.MyTree[child].IsDir {
            tnt.FST_parse_watch(child, ev)
        } else if(ev.Name == child){
            //fmt.Println("i found it", tnt.Tree.MyTree[child])
            return tnt.Tree.MyTree[child]
        }
    }
    return nil
}

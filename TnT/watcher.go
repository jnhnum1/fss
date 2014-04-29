package TnT

import (
    "code.google.com/p/go.exp/inotify"
    "log"
    "fmt"
    //"time"
    //"path/filepath"
    "strings"
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
    fmt.Println("in fst_create")
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
        if !strings.Contains(fi.Name(),"~") {
            child_name := dirname+fi.Name()
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

    err := watcher.Watch(dirname)
    if err != nil {
        log.Fatal(err)
    }

    /*
    for child, _ := range tnt.Tree.MyTree[dirname].Children {
        fmt.Println("start of loop", child)

        //err := watcher.Watch(child)
        //err := watcher.AddWatch(child, IN_MODIFY | IN_CREATE | IN_DELETE)
        if err != nil {
            log.Fatal(err)
        }

        if tnt.Tree.MyTree[child].IsDir {
            watcher.Watch(child)
            tnt.FST_set_watch(child, watcher)
        } 
    } */
}

//This function watches all of the files in the background and takes action accordingly
func (tnt *TnTServer) FST_watch_files(dirname string, watcher *inotify.Watcher){
    fmt.Println("in FST_watch_files")
    //fmt.Println(tnt.Tree.MyTree[dirname])
    var cur_file string
    var seq_count int = 0
    var break_bool bool
    for {
        select {
            case ev := <-watcher.Event:
                
                //This if statement causes us to avoid taking into account swap files used to keep 
                //track of file modifications
                if(!strings.Contains(ev.Name, ".swp") && !strings.Contains(ev.Name, ".swx") && !strings.Contains(ev.Name, "~")){
                
                    //fmt.Println("ev: ", ev, "file node: ", tnt.Tree.MyTree[ev.Name])

                    //there are 3 cases I need to take care of
                    // 1) Create a file/folder - add it to tree
                    //Folder only command is IN_CREATE with name as path
                    if(ev.Mask == IN_CREATE_ISDIR){
                        fmt.Println("new folder")
                        tnt.Tree.MyTree[ev.Name] = new(FSnode)
                        tnt.Tree.MyTree[ev.Name].IsDir = fi.IsDir()
                        tnt.Tree.MyTree[ev.Name].VerVect = 0
                        tnt.Tree.MyTree[ev.Name].SyncVect = 0
                    }

                    //This is the sequence of commands when a file is created or modified
                    if(ev.Mask == IN_CREATE && seq_count == 0){
                        cur_file = ev.Name
                        seq_count = 1
                    }else if(ev.Mask == IN_OPEN && seq_count == 1){

                        seq_count = 2
                    } else if(ev.Mask == IN_MODIFY && seq_count == 2){

                        seq_count = 3
                    }else if(ev.Mask == IN_CLOSE && cur_file == ev.Name && seq_count == 3){
                        if(tnt.Tree.MyTree[ev.Name] == nil){
                            fmt.Println("new file was created")
                            tnt.Tree.MyTree[ev.Name] = new(FSnode)
                            tnt.Tree.MyTree[ev.Name].IsDir = fi.IsDir()
                            tnt.Tree.MyTree[ev.Name].VerVect = 0
                            tnt.Tree.MyTree[ev.Name].SyncVect = 0
                        }else{
                            fmt.Println("file has been modified")
                            if(tnt.Tree.MyTree[ev.Name].VerVect == tnt.Tree.MyTree[ev.Name].SyncVect){
                                tnt.Tree.MyTree[ev.Name].VerVect++
                            }
                        }
                    }else {
                        seq_count = 0
                    }


                    // 2) Modify a file - increment its modified vector by 1


                    // 3) Delete a file - indicate it has been removed, don't necessarily remove it from tree
                    if(ev.Mask == IN_DELETE && tnt.Tree.MyTree[ev.Name] != nil){
                        fmt.Println("file has been deleted")
                    }
                }

            case err := <-watcher.Error:
                log.Println("error:", err)
        }
    }
    if break_bool {
        //tnt.FST_watch_files(dirname, watcher)
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

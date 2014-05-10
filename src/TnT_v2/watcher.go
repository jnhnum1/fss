package TnT_v2

import (
    "code.google.com/p/go.exp/inotify"
    "log"
    "fmt"
    //"time"
    //"path/filepath"
    "strings"
    "os"
    //"encoding/gob"
)

/*
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
        var child_name string
        if (!strings.Contains(fi.Name(),"~")) {
            if(fi.IsDir()){
                child_name = dirname+fi.Name() + "/"
                fmt.Println(child_name)
            }else {
                child_name = dirname+fi.Name()
            }
            if(tnt.Tree.MyTree[child_name] == nil){
                fmt.Println(child_name)
                tnt.Tree.MyTree[child_name] = new(FSnode)
                tnt.Tree.MyTree[child_name].Name = fi.Name()
                tnt.Tree.MyTree[child_name].Size = fi.Size()
                tnt.Tree.MyTree[child_name].LastModTime = fi.ModTime()
                tnt.Tree.MyTree[child_name].IsDir = fi.IsDir()
                tnt.Tree.MyTree[child_name].Depth = depth+1
                tnt.Tree.MyTree[child_name].VerVect = make(map[int]int)
                tnt.Tree.MyTree[child_name].VerVect[tnt.me] = 1
                tnt.Tree.MyTree[child_name].SyncVect = make(map[int]int)
                tnt.Tree.MyTree[child_name].SyncVect[tnt.me] = 1
                tnt.Tree.MyTree[child_name].Parent = dirname
                tnt.Tree.MyTree[child_name].Exists = true
                tnt.Tree.MyTree[dirname].Children[child_name] = true
                if fi.IsDir() {
                    tnt.Tree.MyTree[child_name].Children = make(map[string]bool)
                    tnt.FST_create(child_name, depth+1)
                }
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
*/

//This function sets watch on folders in directory
func (tnt *TnTServer) FST_set_watch(dirname string, watcher *inotify.Watcher) {
    
    new_dirname := strings.TrimSuffix(dirname, "/")

    err := watcher.Watch(new_dirname)
    //fmt.Println("in fst_set_watch", dirname, new_dirname)
    if err != nil {
        log.Fatal(err)
    }
    //fmt.Println(dirname)
    for name, fi := range tnt.Tree.MyTree {
        if(fi.IsDir == true && name != "./"){
            new_name := strings.TrimPrefix(strings.TrimSuffix(name, "/"), "./")

            //fmt.Println("in fst_set_watch",dirname, name, tnt.root+new_name)
            err := watcher.Watch(tnt.root+new_name)
            if err != nil {
                log.Fatal(err)
            }
        }
    }


}

//This function watches all of the files in the background and takes action accordingly
func (tnt *TnTServer) FST_watch_files(dirname string){
    //First initialize the watch

    watcher, err := inotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }

    tnt.FST_set_watch(dirname, watcher)

    //fmt.Println("in FST_watch_files", dirname)
    //fmt.Println(tnt.Tree.MyTree[dirname])
    var cur_file string
    var seq_count int = 0 //This is for creation and mods from text editors
    var mod_count int = 0 //This is for tracking modifications
    var move_count int = 0
    var old_path string
    for {
        select {
            case ev := <-watcher.Event:
                
                //This if statement causes us to avoid taking into account swap files used to keep 
                //track of file modifications
                if(!strings.Contains(ev.Name, ".swp") && !strings.Contains(ev.Name, ".swx") && !strings.Contains(ev.Name, "~") && !strings.Contains(ev.Name, ".goutputstream")){                
                    fmt.Println("ev: ", ev, "file node: ", tnt.Tree.MyTree[ev.Name])
                    //fmt.Println("ev.Name: ", ev.Name)
                    fi, _ := os.Lstat(ev.Name)
                    key_path := "./"+strings.TrimPrefix(ev.Name,tnt.root)
                    //fmt.Println("./"+key_path)
                    //trim_name := strings.TrimPrefix(ev.Name, tnt.root)

                    //fmt.Println("did the files open?", dirname, tnt.root, ev.Name, fi, err)
                
                    // 1) Create a file/folder - add it to tree
                    //Folder only command is IN_CREATE with name as path
                    if(ev.Mask == IN_CREATE_ISDIR && tnt.Tree.MyTree[key_path] == nil){
                        fmt.Println("new folder", ev.Name)
                        err := watcher.Watch(ev.Name)
                        if err != nil {
                            log.Fatal(err)
                        }

                        tnt.Tree.MyTree[key_path] = new(FSnode)
                        tnt.Tree.MyTree[key_path].Name = fi.Name()
                        tnt.Tree.MyTree[key_path].Size = fi.Size()
                        tnt.Tree.MyTree[key_path].IsDir = fi.IsDir()
                        tnt.Tree.MyTree[key_path].LastModTime = fi.ModTime()
                        tnt.Tree.MyTree[key_path].Creator = tnt.me
                        tnt.Tree.MyTree[key_path].CreationTime = tnt.Tree.LogicalTime
                        tnt.Tree.MyTree[key_path].Children = make(map[string]bool)

                        tnt.Tree.MyTree[key_path].VerVect = make(map[int]int64)
                        tnt.Tree.MyTree[key_path].SyncVect = make(map[int]int64)
                        for i:=0; i<len(tnt.servers); i++ {
                            tnt.Tree.MyTree[key_path].VerVect[i] = 0
                            tnt.Tree.MyTree[key_path].SyncVect[i] = 0
                        }
                        tnt.Tree.MyTree[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        tnt.Tree.MyTree[key_path].Parent = parent(ev.Name)                      

                        //fmt.Println("parent is ", tnt.Tree.MyTree[ev.Name].Parent)
                        
                    }

                    //This is the sequence of commands when a file is created or modified in a text editor
                    //fmt.Println(seq_count)
                    if(ev.Mask == IN_CREATE && seq_count == 0 && !strings.Contains(ev.Name,"/tmp")){
                        cur_file = ev.Name
                        seq_count = 1
                    }else if(ev.Mask == IN_OPEN && seq_count == 1){

                        seq_count = 2
                    } else if(ev.Mask == IN_MODIFY && seq_count == 2){

                        seq_count = 3
                    }else if(ev.Mask == IN_CLOSE && cur_file == ev.Name && seq_count == 3){
                        seq_count = 0
                        if(tnt.Tree.MyTree[key_path] == nil){
                            fmt.Println("new file was created", ev.Name)
                            tnt.Tree.MyTree[key_path] = new(FSnode)
                            tnt.Tree.MyTree[key_path].Name = fi.Name()
                            tnt.Tree.MyTree[key_path].Size = fi.Size()
                            tnt.Tree.MyTree[key_path].IsDir = fi.IsDir()
                            tnt.Tree.MyTree[key_path].LastModTime = fi.ModTime()
                            tnt.Tree.MyTree[key_path].Creator = tnt.me
                            tnt.Tree.MyTree[key_path].CreationTime = tnt.Tree.LogicalTime

                            tnt.Tree.MyTree[key_path].VerVect = make(map[int]int64)
                            tnt.Tree.MyTree[key_path].SyncVect = make(map[int]int64)
                            for i:=0; i<len(tnt.servers); i++ {
                                tnt.Tree.MyTree[key_path].VerVect[i] = 0
                                tnt.Tree.MyTree[key_path].SyncVect[i] = 0
                            }
                            tnt.Tree.MyTree[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                            tnt.Tree.MyTree[key_path].Parent = parent(ev.Name)                 

                        }else{
                            // 2) Modify a file - increment its modified vector by 1
                            fmt.Println("file has been modified", key_path)
                            if(tnt.Tree.MyTree[key_path].VerVect[tnt.me] < tnt.Tree.MyTree[key_path].SyncVect[tnt.me]){

                            }
                        }
                    }else {
                        seq_count = 0
                    }

                    //This is the events that occur when files modified from the command line
                    if(ev.Mask == IN_MODIFY && mod_count == 0 && !strings.Contains(ev.Name,"/tmp")){
                        cur_file = ev.Name
                        mod_count = 1
                    }else if(ev.Mask == IN_OPEN && mod_count == 1){

                        mod_count = 2
                    } else if(ev.Mask == IN_MODIFY && mod_count == 2){

                        mod_count = 3
                    }else if(ev.Mask == IN_CLOSE && cur_file == ev.Name && mod_count == 3){
                        mod_count = 0

                        // 2) Modify a file - increment its modified vector by 1
                        fmt.Println("file has been modified", ev.Name)
                    }else {
                        mod_count = 0
                    }



                    // 3) Delete a file - indicate it has been removed, don't necessarily remove it from tree
                    if(ev.Mask == IN_DELETE && tnt.Tree.MyTree[ev.Name] != nil){
                        fmt.Println("file has been deleted", ev.Name)
                        tnt.DeleteTree(ev.Name)
                    }
                    // 6) Delete a directory, need to parse and remove children as well
                    if(ev.Mask == IN_DELETE_ISDIR){
                        fmt.Println("folder has been deleted", ev.Name)
                        tnt.DeleteTree(ev.Name)
                    }

                    // 5) Do nothing when transferring files from tmp/ to the rest of the directory
                    //fmt.Println(ev, move_count)
                    
                    if(ev.Mask == IN_MOVE_FROM && move_count == 0){
                        fmt.Println("This is a move")
                        old_path = ev.Name

                        move_count = 1
                    }else if( ev.Mask == IN_MOVE_TO && move_count == 1){
                        fmt.Println("file has been moved", old_path)

                        if(strings.Contains(old_path,"/tmp")){
                            fmt.Println("Moved thru transfer, do nothing")
                        }else{
                            fmt.Println("Actual Move, do something")
                        }

                        move_count = 0
                    }else if(ev.Mask == IN_MOVE_TO && tnt.Tree.MyTree[key_path] != nil && move_count == 0){
                        //This is when a file has been modified
                        fmt.Println("file has been modified")
                    }else if(ev.Mask == IN_MOVE_TO && tnt.Tree.MyTree[key_path] == nil && move_count == 0) {
                        fmt.Println("file has been moved from outside directory")
                    }else{
                        move_count = 0
                    }
                    
                }

            case err := <-watcher.Error:
                log.Println("error:", err)
        }
    }
}

/*
//This function is used to recursively parse the tree to find the file that set off an event in FST_watch_files
//And return its node in the tree
func (tnt *TnTServer) FST_find_parent(dirname string, ev *inotify.Event) string {
    var parent_folder string

    new_name := strings.TrimPrefix(ev.Name, dirname)
    fmt.Println("in FST_find_parent", dirname, ev.Name, strings.Contains(new_name, "/"))
    if(!strings.Contains(new_name, "/")){
        parent_folder = dirname
        //fmt.Println("found it", parent_folder)
    }else {
        //fmt.Println("not there yet")
        for child, _ := range tnt.Tree.MyTree {
            fmt.Println(child, strings.Contains(child,dirname))
            if (tnt.Tree.MyTree[child].IsDir && strings.Contains(child,dirname) && child != dirname) {
                parent_folder = ""
                parent_folder = tnt.FST_find_parent(child, ev)
                if(parent_folder != ""){
                    break
                }
            }
        }
    }
    //fmt.Println("do i return again?", parent_folder)
    return parent_folder
}
*/

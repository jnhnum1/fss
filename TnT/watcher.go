package TnT_v2

import (
    "code.google.com/p/go.exp/inotify"
    "log"
    "fmt"
    "strconv"
    //"path/filepath"
    "strings"
    "os"
    //"encoding/gob"
)

//This function sets watch on folders in directory
func (tnt *TnTServer) FST_set_watch(dirname string, watcher *inotify.Watcher) {
    
    new_dirname := strings.TrimSuffix(dirname, "/")

    err := watcher.Watch(new_dirname)
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

    //Set watch on /tmp folder for transfers
    tnt.FST_set_watch("../roots/tmp"+strconv.Itoa(tnt.me)+"/", watcher)
    tnt.FST_set_watch(dirname, watcher)

    fst := tnt.Tree.MyTree

    //fmt.Println("in FST_watch_files", dirname)
    //fmt.Println(fst[dirname])
    var cur_file string
    var seq_count int = 0 //This is for creation and mods from text editors
    var mod_count int = 0 //This is for tracking modifications
    var move_count int = 0
    var old_path string
    var old_path_key string
    for {
        select {
            case ev := <-watcher.Event:
                
                //This if statement causes us to avoid taking into account swap files used to keep 
                //track of file modifications
                if(!strings.Contains(ev.Name, ".swp") && !strings.Contains(ev.Name, ".swx") && !strings.Contains(ev.Name, "~") && !strings.Contains(ev.Name, ".goutputstream") && !strings.Contains(ev.Name,"roots/tmp")){                
                    fmt.Println("ev: ", ev)
                    //fmt.Println("ev.Name: ", ev.Name)
                    fi, _ := os.Lstat(ev.Name)
                    key_path := "./"+strings.TrimPrefix(ev.Name,tnt.root)

                    tnt.Tree.LogicalTime++

                    tnt.UpdateTree(key_path)
                    //fmt.Println("./"+key_path)
                    //trim_name := strings.TrimPrefix(ev.Name, tnt.root)
                
                    // 1) Create a file/folder - add it to tree
                    //Folder only command is IN_CREATE with name as path
                    /*
                    if(ev.Mask == IN_CREATE_ISDIR && fst[key_path] == nil && !strings.Contains(ev.Name,"roots/tmp")){
                        fmt.Println("new folder", ev.Name)
                        err := watcher.Watch(ev.Name)
                        if err != nil {
                            log.Fatal(err)
                        }

                        tnt.Tree.LogicalTime++

                        tnt.UpdateTree(key_path)
                        /*
                        fst[key_path] = new(FSnode)
                        fst[key_path].Name = fi.Name()
                        fst[key_path].Size = fi.Size()
                        fst[key_path].IsDir = fi.IsDir()
                        fst[key_path].LastModTime = fi.ModTime()
                        fst[key_path].Creator = tnt.me
                        fst[key_path].CreationTime = tnt.Tree.LogicalTime
                        fst[key_path].Children = make(map[string]bool)

                        fst[key_path].VerVect = make(map[int]int64)
                        fst[key_path].SyncVect = make(map[int]int64)
                        for i:=0; i<len(tnt.servers); i++ {
                            fst[key_path].VerVect[i] = 0
                            fst[key_path].SyncVect[i] = 0
                        }
                        fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].Parent = parent(ev.Name)
                                             

                        //fmt.Println("parent is ", fst[ev.Name].Parent)
                        
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
                        if(fst[key_path] == nil){
                            fmt.Println("new file was created", ev.Name)
                            tnt.Tree.LogicalTime++
                            fst[key_path] = new(FSnode)
                            fst[key_path].Name = fi.Name()
                            fst[key_path].Size = fi.Size()
                            fst[key_path].IsDir = fi.IsDir()
                            fst[key_path].LastModTime = fi.ModTime()
                            fst[key_path].Creator = tnt.me
                            fst[key_path].CreationTime = tnt.Tree.LogicalTime

                            fst[key_path].VerVect = make(map[int]int64)
                            fst[key_path].SyncVect = make(map[int]int64)
                            for i:=0; i<len(tnt.servers); i++ {
                                fst[key_path].VerVect[i] = 0
                                fst[key_path].SyncVect[i] = 0
                            }
                            fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                            fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                            fst[key_path].Parent = parent(ev.Name)                 

                        }else{
                            // 2) Modify a file - increment its modified vector by 1
                            fmt.Println("file has been modified", fst[key_path])
                            tnt.Tree.LogicalTime++
                            fst[key_path].LastModTime = fi.ModTime()
                            fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                            fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                            tnt.PropagateUp(fst[key_path].VerVect,fst[key_path].SyncVect,fst[key_path].Parent)

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
                        //fmt.Println("file has been modified", fst[key_path])
                        tnt.Tree.LogicalTime++
                        fst[key_path].LastModTime = fi.ModTime()
                        fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                        tnt.PropagateUp(fst[key_path].VerVect,fst[key_path].SyncVect,fst[key_path].Parent)
                        fmt.Println("file has been modified", fst[key_path])

                    }else {
                        mod_count = 0
                    }


                    // 3) Delete a file - indicate it has been removed, don't necessarily remove it from tree
                    if(ev.Mask == IN_DELETE && fst[key_path] != nil){
                        fmt.Println("file has been deleted", fst[key_path])
                        tnt.Tree.LogicalTime++

                        fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                        tnt.PropagateUp(fst[key_path].VerVect,fst[key_path].SyncVect,fst[key_path].Parent)

                        tnt.DeleteTree(ev.Name)
                    }
                    // 6) Delete a directory, need to parse and remove children as well
                    if(ev.Mask == IN_DELETE_ISDIR && fst[key_path] != nil){
                        fmt.Println("folder has been deleted", fst[key_path])
                        tnt.Tree.LogicalTime++

                        fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                        tnt.PropagateUp(fst[key_path].VerVect,fst[key_path].SyncVect,fst[key_path].Parent)

                        tnt.DeleteTree(ev.Name)
                    }

                    // 5) Do nothing when transferring files from tmp/ to the rest of the directory
                    //fmt.Println(ev, move_count)
                    
                    if(ev.Mask == IN_MOVE_FROM && move_count == 0){
                        fmt.Println("This is a move")
                        old_path = ev.Name
                        old_path_key = "./"+strings.TrimPrefix(ev.Name,tnt.root)

                        move_count = 1
                    }else if( ev.Mask == IN_MOVE_TO && move_count == 1){
                        fmt.Println("file has been moved", old_path)

                        if(strings.Contains(old_path,"/tmp")){
                            fmt.Println("Moved thru transfer, do nothing")
                        }else{
                            fmt.Println("Actual Move, do something")
                            //Need to delete previous path and create new path
                            tnt.Tree.LogicalTime++

                            fst[old_path_key].VerVect[tnt.me] = tnt.Tree.LogicalTime
                            fst[old_path_key].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                            tnt.PropagateUp(fst[old_path_key].VerVect,fst[old_path_key].SyncVect,fst[old_path_key].Parent)

                            tnt.DeleteTree(old_path)

                            fst[key_path] = new(FSnode)
                            fst[key_path].Name = fi.Name()
                            fst[key_path].Size = fi.Size()
                            fst[key_path].IsDir = fi.IsDir()
                            fst[key_path].LastModTime = fi.ModTime()
                            fst[key_path].Creator = tnt.me
                            fst[key_path].CreationTime = tnt.Tree.LogicalTime

                            fst[key_path].VerVect = make(map[int]int64)
                            fst[key_path].SyncVect = make(map[int]int64)
                            for i:=0; i<len(tnt.servers); i++ {
                                fst[key_path].VerVect[i] = 0
                                fst[key_path].SyncVect[i] = 0
                            }
                            fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                            fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                            fst[key_path].Parent = parent(ev.Name)
                        }

                        move_count = 0
                    }else if(ev.Mask == IN_MOVE_TO && fst[key_path] != nil && move_count == 0){
                        //This is when a file has been modified
                        fmt.Println("file has been modified")
                        tnt.Tree.LogicalTime++
                        fst[key_path].LastModTime = fi.ModTime()
                        fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                        tnt.PropagateUp(fst[key_path].VerVect,fst[key_path].SyncVect,fst[key_path].Parent)

                    }else if(ev.Mask == IN_MOVE_TO && fst[key_path] == nil && move_count == 0) {
                        fmt.Println("file has been moved from outside directory, treat like new file")
                        tnt.Tree.LogicalTime++
                        fst[key_path] = new(FSnode)
                        fst[key_path].Name = fi.Name()
                        fst[key_path].Size = fi.Size()
                        fst[key_path].IsDir = fi.IsDir()
                        fst[key_path].LastModTime = fi.ModTime()
                        fst[key_path].Creator = tnt.me
                        fst[key_path].CreationTime = tnt.Tree.LogicalTime

                        fst[key_path].VerVect = make(map[int]int64)
                        fst[key_path].SyncVect = make(map[int]int64)
                        for i:=0; i<len(tnt.servers); i++ {
                            fst[key_path].VerVect[i] = 0
                            fst[key_path].SyncVect[i] = 0
                        }
                        fst[key_path].VerVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                        fst[key_path].Parent = parent(ev.Name)

                    }else{
                        move_count = 0
                    }
                */                    
                }

            case err := <-watcher.Error:
                log.Println("error:", err)
        }
    }
}


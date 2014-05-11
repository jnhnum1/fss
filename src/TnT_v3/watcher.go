package TnT_v2

import (
    "code.google.com/p/go.exp/inotify"
    "log"
    "fmt"
    "strconv"
    "os"
    "strings"
    //"os"
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
    fmt.Println(dirname)
    //Set watch on /tmp folder for transfers
    tnt.FST_set_watch("../roots/tmp"+strconv.Itoa(tnt.me)+"/", watcher)
    tnt.FST_set_watch(dirname, watcher)

    fst := tnt.Tree.MyTree

    for {
        select {
            case ev := <-watcher.Event:
                //fmt.Println("I see event: ", ev)
                //This if statement causes us to avoid taking into account swap files used to keep 
                //track of file modifications
                if(!strings.Contains(ev.Name, ".swp") && !strings.Contains(ev.Name, ".swx") && !strings.Contains(ev.Name, "~") && !strings.Contains(ev.Name, ".goutputstream") && !strings.Contains(ev.Name,tnt.tmp)) {                
                    if(ev.Mask != IN_CLOSE && ev.Mask != IN_OPEN && ev.Mask != IN_OPEN_ISDIR && ev.Mask != IN_CLOSE_ISDIR){
                    //fmt.Println("ev.Name: ", ev.Name)
                    fi, err := os.Lstat(ev.Name)
                    key_path := "./"+strings.TrimPrefix(ev.Name,tnt.root)

                    //fmt.Println("ev: ", ev, key_path)

                    if err == nil {
                        if fi.IsDir(){
                            tnt.FST_set_watch(ev.Name, watcher)
                            key_path = key_path + "/"
                        }
                    } else if fst[key_path + "/"] != nil {
                        key_path = key_path + "/"
                    } else if fst[key_path] != nil{
                        fmt.Println("this is a file")
                    } else {
                        fmt.Println("what am i doing", err, fst[key_path])
                    }

                    //fmt.Println("key to update", key_path)
                    //if(fst[key_path] != nil || err == nil){
					tnt.mu.Lock()
					tnt.UpdateTreeWrapper(key_path)
					tnt.mu.Unlock()
                    //}   

                    
                    }
                    
                }

            case err := <-watcher.Error:
                log.Println("error:", err)

        }
    }
}


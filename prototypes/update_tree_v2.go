package main

import (
    "time"
    "fmt"
    "log"
    "sync"
    "net"
    "os"
    "path/filepath"
    "encoding/gob"
)

type FSnode struct {
    Name string
    Size int64
    IsDir bool
    Children map[string]bool
    LastModTime time.Time
    Creator int
    CreationTime int64
    VerVect map[int]int64	//ask Zach to modify this
    SyncVect map[int]int64	//ask Zach to modify this
    Parent string		//ask Zach to add this
}

type FStree struct {
    LogicalTime int64
    MyTree map[string] *FSnode
}

type TnTServer struct {
    mu sync.Mutex
    l net.Listener
    me int
    dead bool
    servers []string
    root string
    Tree *FStree
}

func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf(" |")
    }
    fmt.Printf(" |---- ")
}

func setMinVersionVect(hA map[int]int64, hB map[int]int64) {
  /* For all k, sets hA[k] = max(hA[k], hB[k]) */
  for k, v := range hB {
    if hA[k] > v {
        hA[k] = v
    }
  }
}

func setMaxVersionVect(hA map[int]int64, hB map[int]int64) {
  /* For all k, sets hA[k] = max(hA[k], hB[k]) */
  for k, v := range hB {
    if hA[k] < v {
        hA[k] = v
    }
  }
}

/*
Non-trivial remarks: 
(1) In our notation, directory path names always end in "/", whereas file path names never end in "/"
    For example, "./root/foo/" is the path name of a folder, but "./root/foo" is path name of a file
    This way, if a folder named "foo" existed in the past under "root" and a file named "foo" exists under root now,
    the FStree.Tree will have entries for both "./root/foo/" and "./root/foo" and they won't interfere! :)

(2) fst.Tree[dir].Children is being used for some clever book-keeping, and not just as a "set".
    So don't expect the truth values stored in the map to always be "true".
*/

func (tnt *TnTServer) UpdateTreeWrapper(dir string) {
    tnt.Tree.LogicalTime += 1
    tnt.UpdateTree(dir)
}

/*
func (tnt *TnTServer) SyncMeDown(path string) {

    fst := tnt.Tree.MyTree

    if fst[path].IsDir {
        for child, _ := range fst[path].Children {
            tnt.SyncMeDown(child)
        }
    }
    fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
}
*/

func (tnt *TnTServer) DeleteTree(path string) {
    // Deletes entire sub-tree under 'path' from FStree

    fst := tnt.Tree.MyTree

    if _, present := fst[path]; present {
        // Delete all children; recursively delete if child is a directory
        for child, _ := range fst[path].Children {
            tnt.DeleteTree(child)
        }
        delete(fst, path)
    }
}

func (tnt *TnTServer) UpdateTree(path string) {

    /* Remark: 'path' can be either a file or directory.

    (1) explores the file system, and make appropriate changes in FStree
    (2) deletes nodes in FStree which are not in the file system

    Caution: Should ensure that the first time UpdateTree is called, then
    'path' should exist in 'fst', and it's 'Parent' should be set appropriately.
    Otherwise, UpdateTree will create a spurious sub-tree which will not be
    reachable from the root. It is fine if stuff under 'path' are not in FST already.
    */

    fst := tnt.Tree.MyTree

    fi, err := os.Lstat(tnt.root + path)

    if err != nil {
        tnt.DeleteTree(path)
        return
    }

    if _, present := fst[path]; present == false {
        fst[path] = new(FSnode)
        fst[path].Name = fi.Name()
        fst[path].Size = fi.Size()
        fst[path].IsDir = fi.IsDir()
        fst[path].LastModTime = fi.ModTime()
        fst[path].Creator = tnt.me
        fst[path].CreationTime = tnt.Tree.LogicalTime

        if fi.IsDir() {
            fst[path].Children = make(map[string]bool)
        }

        fst[path].VerVect = make(map[int]int64)
        fst[path].SyncVect = make(map[int]int64)
        for i:=0; i<len(tnt.servers); i++ {
            fst[path].VerVect[i] = 0
            fst[path].SyncVect[i] = 0
        }

        fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
        // fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime // set outside - unconditionally
    }

    fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime

    if fi.IsDir() == false {
        if fst[path].LastModTime.Equal(fi.ModTime()) == false {
            fst[path].LastModTime = fi.ModTime()
            fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
        }
    } else {

        d, err := os.Open(tnt.root + path)
        defer d.Close()
        cfi, err := d.Readdir(-1)
        if err != nil {
            log.Println("LOL Error in UpdateTree:", err)
            os.Exit(1)
        }

        // Book-keeping : if fst.Tree[dir].Children[child] remains false in the end,
        // then it means child does not exist in file system
        for child, _ := range fst[path].Children {
            fst[path].Children[child] = false
        }

        for _, cfi := range cfi {
            var child string
            if cfi.IsDir() {
                child = path + cfi.Name() + string(filepath.Separator)
            } else {
                child = path + cfi.Name()
            }
            tnt.UpdateTree(child)
            fst[path].Children[child] = true
            fst[child].Parent = path

            // Update LastModTime, VerVect and SyncVect for 'dir' :

            // set my last mod time to be latest among all my children
            if fst[path].LastModTime.Before(fst[child].LastModTime) {
                fst[path].LastModTime = fst[child].LastModTime
            }

            setMaxVersionVect(fst[path].VerVect, fst[child].VerVect)   // VerVect is element-wise maximum of children's VerVect
            // SyncVect is element-wise maximum of children's SyncVect - no need to set because everything will anyway be tnt.Tree.LogicalTime!
            //setMinVersionVect(fst[path].SyncVect, fst[child].SyncVect)
        }

        // Delete all my children, who were deleted since the last sync
        for child, exists := range fst[path].Children {
            if exists == false {
                tnt.DeleteTree(child)
                delete(fst[path].Children, child)
            }
        }
    }
}

func (tnt *TnTServer) ParseTree(path string, depth int) {
    fst := tnt.Tree.MyTree

    spaces(depth)
    fmt.Println(fst[path].Name, ":	", fst[path].LastModTime, "	", fst[path].VerVect, "	", fst[path].SyncVect)

    if fst[path].IsDir {
        for child, _ := range fst[path].Children {
            //fmt.Println("--", path, "-", child, "fst[path].Children[child]:", fst[path].Children[child])
            //fmt.Printf("%t : ", fst[path].Children[child])
            tnt.ParseTree(child, depth+1)
        }
    }
}

func main() {

    root_folder := "watch_folder"
    root := "." + string(filepath.Separator) + root_folder + string(filepath.Separator)
    dump := "FST_watch_new"

    tnt := new(TnTServer)
    tnt.me = 0
    tnt.root = root
    tnt.servers = []string {"serv1", "serv2", "serv3"}

    f, err := os.Open(dump)
    defer f.Close()
    if err != nil {
        fmt.Println(dump, "not found. Creating new tree...")
        fst := new(FStree)
        fst.LogicalTime = 0
        fst.MyTree = make(map[string]*FSnode)
        fst.MyTree["./"] = new(FSnode)
        fst.MyTree["./"].Name = root_folder
        fst.MyTree["./"].IsDir = true
        fst.MyTree["./"].Children = make(map[string]bool)
        fst.MyTree["./"].LastModTime = time.Now()
        // Assuming that the root will never be deleted, Creator and CreationTime are not needed!

        fst.MyTree["./"].VerVect = make(map[int]int64)
        fst.MyTree["./"].SyncVect = make(map[int]int64)
        fst.MyTree["./"].Parent = "./"

        // Initialize VecVect, SyncVect
        for i:=0; i<len(tnt.servers); i++ {
            fst.MyTree["./"].VerVect[i] = 0
            fst.MyTree["./"].SyncVect[i] = 0
        }

        tnt.Tree = fst
    } else {
        fmt.Println(dump, "found! Fetching tree...")
        var fst1 FStree
        decoder := gob.NewDecoder(f)
        decoder.Decode(&fst1)
        tnt.Tree = &fst1
    }

    tnt.UpdateTreeWrapper("./")

    tnt.ParseTree("./", 0)

    f, err = os.OpenFile(dump, os.O_WRONLY | os.O_CREATE, 0777)
    if err != nil {
        log.Println("Error opening file:", err)
    }

    encoder := gob.NewEncoder(f)
    encoder.Encode(tnt.Tree)
    f.Close()
    fmt.Println(dump + " dumped!")
}

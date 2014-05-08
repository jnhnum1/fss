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
    VerVect map[int]int64	//ask Zach to modify this
    SyncVect map[int]int64	//ask Zach to modify this
    Parent string		//ask Zach to add this
    Exists bool			//ask Zach to add this
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
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
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
    tnt.SyncMeDown(dir)
}

func (tnt *TnTServer) SyncMeDown(path string) {

    fst := tnt.Tree.MyTree

    if fst[path].IsDir {
        for child, _ := range fst[path].Children {
            tnt.SyncMeDown(child)
        }
    }
    fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
}


func (tnt *TnTServer) DeleteTree(dir string) {
    /* Remarks:
    (1) "Deletes" entire sub-tree under 'dir'
    (2) Should be used only when fst[dir].Exists = true
    */

    fst := tnt.Tree.MyTree

    _, present := fst[dir]
    if present {

        // Delete all children; recursively delete if child is a directory
        for child, exists := range fst[dir].Children {
            if exists {
                if fst[child].IsDir {
                    tnt.DeleteTree(child)
                } else {
                    fst[child].Exists = false
                    fst[child].LastModTime = time.Now()
                    fst[child].VerVect[tnt.me] = tnt.Tree.LogicalTime
                    //fst[child].SyncVect[tnt.me] = tnt.Tree.LogicalTime
                }
                fst[dir].Children[child] = false
            }
        }
        // Set my own state
        fst[dir].Exists = false
        fst[dir].LastModTime = time.Now()  // note: my LastModTime is higher than any of my children
        fst[dir].VerVect[tnt.me] = tnt.Tree.LogicalTime
        //fst[dir].SyncVect[tnt.me] = tnt.Tree.LogicalTime
    }
}

func (tnt *TnTServer) UpdateTree(dir string) {

    //'dir' must be a directory; if it is not then program will crash!!

    fst := tnt.Tree.MyTree

    // (1) explore the file system, and make appropriate changes in FStree
    // (2) "delete" nodes in FStree which are not in the file system

    d, err := os.Open(tnt.root+dir)
    defer d.Close()

    if err != nil {
        tnt.DeleteTree(dir)
        return
    } else {
        fst[dir].Exists = true
    }

    fi, err := d.Readdir(-1)
    if err != nil {
        log.Println("Error in UpdateTree:", err)
        os.Exit(1)
    }

    // Book-keeping : if fst.Tree[dir].Children[child] remains false in the end,
    // then it means child does not exist in file system
    for child, _ := range fst[dir].Children {
        fst[dir].Children[child] = false
    }

    for _, fi := range fi {
        if fi.Mode().IsRegular() {
            // Check if file is already present in FSTree
            child := dir + fi.Name()
            if fst[child] != nil {
                // File is present in FStree; so "update", if required
                fst[dir].Children[child] = true // for book-keeping: child really exists in file-system
                fst[child].Exists = true
                if fst[child].LastModTime.Before(fi.ModTime()) {
                    fst[child].LastModTime = fi.ModTime()
                    fst[child].VerVect[tnt.me] = tnt.Tree.LogicalTime
                    // fst[child].SyncVect[tnt.me] is edited later in SyncMeDown(...) irrespective of whether the file was edited or not
                }
            } else {
                // File is new; so add a new entry in FStree
                fst[child] = new(FSnode)
                fst[child].Name = fi.Name()
                fst[child].Size = fi.Size()
                fst[child].IsDir = fi.IsDir()
                fst[child].LastModTime = fi.ModTime()
                fst[child].VerVect = make(map[int]int64)
                fst[child].SyncVect = make(map[int]int64)
                fst[child].Parent = dir
                fst[child].Exists = true

                // Initialize VecVect, SyncVect
                for i:=0; i<len(tnt.servers); i++ {
                    fst[child].VerVect[i] = 0
                    fst[child].SyncVect[i] = 0
                }

                fst[child].VerVect[tnt.me] = tnt.Tree.LogicalTime
                // fst[child].SyncVect[tnt.me] is edited later in SyncMeDown(...) irrespective of whether the file was edited or not

                // Make an entry in parent directory
                fst[dir].Children[child] = true // also book-keeping: child really exists in file-system
            }
        } else if fi.IsDir() {
            child := dir + fi.Name() + string(filepath.Separator)

            if fst[child] != nil {
                // Directory is present in FStree; so "update" recursively
                fst[dir].Children[child] = true
                tnt.UpdateTree(child)
                // fst[child].SyncVect[tnt.me] is edited later in SyncMeDown(...) irrespective of whether the file was edited or not
            } else {
                // Directory is new; so add a new entry in FStree recursively
                fst[child] = new(FSnode)
                fst[child].Name = fi.Name()
                fst[child].Size = fi.Size()
                fst[child].IsDir = fi.IsDir()
                fst[child].Children = make(map[string]bool)
                fst[child].LastModTime = fi.ModTime()

                fst[child].VerVect = make(map[int]int64)
                fst[child].SyncVect = make(map[int]int64)
                fst[child].Parent = dir
                fst[child].Exists = true

                // Initialize VecVect, SyncVect
                for i:=0; i<len(tnt.servers); i++ {
                    fst[child].VerVect[i] = 0
                    fst[child].SyncVect[i] = 0
                }

                fst[child].VerVect[tnt.me] = tnt.Tree.LogicalTime
                // fst[child].SyncVect[tnt.me] is edited later in SyncMeDown(...) irrespective of whether the file was edited or not

                // Make an entry in parent directory
                fst[dir].Children[child] = true

                tnt.UpdateTree(child)
            }
        }
    }

    // Delete all my children, who were deleted since the last modification
    for child, exists := range fst[dir].Children {
        if exists == false {
            if fst[child].Exists == true {
                tnt.DeleteTree(child)
            }
        }
        // fst[child].SyncVect[tnt.me] is edited later in SyncMeDown(...) irrespective of whether the file was edited or not
    }

    // Update LastModTime, VerVect and SyncVect for 'dir' :
    for child, _ := range  fst[dir].Children {
        // set my last mod time to be latest among all.
        if fst[dir].LastModTime.Before(fst[child].LastModTime) {
            fst[dir].LastModTime = fst[child].LastModTime
        }

        // VerVect is element-wise maximum of children's VerVect
        for k, v := range fst[dir].VerVect {
            if v < fst[child].VerVect[k] {
                fst[dir].VerVect[k] = fst[child].VerVect[k]
            }
        }

        // SyncVect is element-wise minimum of children's SyncVect
        for k, v := range fst[dir].SyncVect {
            if v > fst[child].SyncVect[k] {
                fst[dir].SyncVect[k] = fst[child].SyncVect[k]
            }
        }
    }
}

func (tnt *TnTServer) ParseTree(path string, depth int) {
    fst := tnt.Tree.MyTree

    //spaces(depth)
    fmt.Println(path, fst[path].Name, ":", fst[path].LastModTime, fst[path].Exists, fst[path].VerVect, fst[path].SyncVect)

    if fst[path].IsDir {
        for child, _ := range fst[path].Children {
            spaces(depth)
            //fmt.Println("--", path, "-", child, "fst[path].Children[child]:", fst[path].Children[child])
            fmt.Printf("%t : ", fst[path].Children[child])
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
        fst.MyTree["./"].Name = root
        fst.MyTree["./"].IsDir = true
        fst.MyTree["./"].Children = make(map[string]bool)
        fst.MyTree["./"].LastModTime = time.Now()

        fst.MyTree["./"].VerVect = make(map[int]int64)
        fst.MyTree["./"].SyncVect = make(map[int]int64)
        fst.MyTree["./"].Parent = root
        fst.MyTree["./"].Exists = true

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

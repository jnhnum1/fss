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
    VerVect map[int]int		//ask Zach to modify this
    SyncVect map[int]int	//ask Zach to modify this
    Parent string		//ask Zach to add this
    Exists bool			//ask Zach to add this
}

type FStree struct {
    Tree map[string] *FSnode
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

func (tnt *TnTServer) DeleteTree(dir string) {

    fst := tnt.Tree

    // Delete all children; recursively delete if child is a directory
    for child, _ := range fst.Tree[dir].Children {
        if fst.Tree[child].IsDir {
            tnt.DeleteTree(child)
        } else {
            fst.Tree[child].Exists = false
            fst.Tree[child].LastModTime = time.Now()
            fst.Tree[child].VerVect[tnt.me] = fst.Tree[child].SyncVect[tnt.me] + 1
            fst.Tree[child].SyncVect[tnt.me] += 1
        }
    }
    // Set my own state
    fst.Tree[dir].Exists = false
    fst.Tree[dir].LastModTime = time.Now()  // note: my LastModTime is higher than any of my children
    fst.Tree[dir].VerVect[tnt.me] = fst.Tree[dir].SyncVect[tnt.me] + 1
    fst.Tree[dir].SyncVect[tnt.me] += 1
}

func (tnt *TnTServer) UpdateTree(dir string) {

    fst := tnt.Tree

    // (1) explore the file system, and make appropriate changes in FStree
    // (2) "delete" nodes in FStree which are not in the file system

    d, err := os.Open(dir)
    defer d.Close()

    if err != nil {
        tnt.DeleteTree(dir)        
        return
    }

    fi, err := d.Readdir(-1)
    if err != nil {
        log.Println("Error in UpdateTree:", err)
        os.Exit(1)
    }

    // Book-keeping : if fst.Tree[dir].Children[child] remains false in the end,
    // then it means child does not exist in file system
    for child, _ := range fst.Tree[dir].Children {
        fst.Tree[dir].Children[child] = false
    }

    for _, fi := range fi {
        if fi.Mode().IsRegular() {
            // Check if file is already present in FSTree
            child := dir + fi.Name()
            if fst.Tree[child] != nil {
                // File is present in FStree; so "update", if required
                fst.Tree[dir].Children[child] = true // for book-keeping: child really exists in file-system
                if fst.Tree[child].LastModTime.Before(fi.ModTime()) {
                    fst.Tree[child].Exists = true
                    fst.Tree[child].LastModTime = fi.ModTime()
                    fst.Tree[child].VerVect[tnt.me] = fst.Tree[child].SyncVect[tnt.me] + 1
                    fst.Tree[child].SyncVect[tnt.me] += 1
                }
            } else {
                // File is new; so add a new entry in FStree
                fst.Tree[child] = new(FSnode)
                fst.Tree[child].Name = fi.Name()
                fst.Tree[child].Size = fi.Size()
                fst.Tree[child].IsDir = fi.IsDir()
                fst.Tree[child].LastModTime = fi.ModTime()
                fst.Tree[child].VerVect = make(map[int]int)
                fst.Tree[child].SyncVect = make(map[int]int)
                fst.Tree[child].Parent = dir
                fst.Tree[child].Exists = true

                // Initialize VecVect, SyncVect
                for i:=0; i<len(tnt.servers); i++ {
                    fst.Tree[child].VerVect[i] = 0
                    fst.Tree[child].SyncVect[i] = 0
                }
                fst.Tree[child].VerVect[tnt.me] = 1
                fst.Tree[child].SyncVect[tnt.me] = 1

                // Make an entry in parent directory
                fst.Tree[dir].Children[child] = true // also book-keeping: child really exists in file-system
            }

        } else if fi.IsDir() {
            child := dir + fi.Name() + string(filepath.Separator)

            if fst.Tree[child] != nil {
                // Directory is present in FStree; so "update" recursively
                fst.Tree[dir].Children[child] = true
                tnt.UpdateTree(child)
            } else {
                // Directory is new; so add a new entry in FStree recursively
                fst.Tree[child] = new(FSnode)
                fst.Tree[child].Name = fi.Name()
                fst.Tree[child].Size = fi.Size()
                fst.Tree[child].IsDir = fi.IsDir()
                fst.Tree[child].Children = make(map[string]bool)
                fst.Tree[child].LastModTime = fi.ModTime()

                fst.Tree[child].VerVect = make(map[int]int)
                fst.Tree[child].SyncVect = make(map[int]int)
                fst.Tree[child].Parent = dir
                fst.Tree[child].Exists = true

                // Initialize VecVect, SyncVect
                for i:=0; i<len(tnt.servers); i++ {
                    fst.Tree[child].VerVect[i] = 0
                    fst.Tree[child].SyncVect[i] = 0
                }
                fst.Tree[child].VerVect[tnt.me] = 1
                fst.Tree[child].SyncVect[tnt.me] = 1

                // Make an entry in parent directory
                fst.Tree[dir].Children[child] = true

                tnt.UpdateTree(child)
            }
        }
    }

    // Delete all my children, who were deleted since the last modification
    for child, exists := range fst.Tree[dir].Children {
        if exists == false {
            if fst.Tree[child].Exists == true {
                tnt.DeleteTree(child)
            }
        }
    }

    // Update LastModTime, VerVect and SyncVect for 'dir' :
    for child, _ := range  fst.Tree[dir].Children {
        // set my last mod time to be latest among all.
        if fst.Tree[dir].LastModTime.Before(fst.Tree[child].LastModTime) {
            fst.Tree[dir].LastModTime = fst.Tree[child].LastModTime
        }

        // VerVect is element-wise maximum of children's VerVect
        for k, v := range fst.Tree[dir].VerVect {
            if v < fst.Tree[child].VerVect[k] {
                fst.Tree[dir].VerVect[k] = fst.Tree[child].VerVect[k]
            }
        }

        // SyncVect is element-wise minimum of children's SyncVect
        for k, v := range fst.Tree[dir].SyncVect {
            if v > fst.Tree[child].SyncVect[k] {
                fst.Tree[dir].SyncVect[k] = fst.Tree[child].SyncVect[k]
            }
        }
    }
}

func (tnt *TnTServer) ParseTree(path string, depth int) {
    fst := tnt.Tree

    spaces(depth)
    fmt.Println(fst.Tree[path].Name, ":", fst.Tree[path].LastModTime, fst.Tree[path].Exists, fst.Tree[path].VerVect, fst.Tree[path].SyncVect)

    if fst.Tree[path].IsDir {
        for child, _ := range fst.Tree[path].Children {
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
        fst.Tree = make(map[string]*FSnode)
        fst.Tree[root] = new(FSnode)
        fst.Tree[root].Name = root_folder
        fst.Tree[root].IsDir = true
        fst.Tree[root].Children = make(map[string]bool)
        fst.Tree[root].LastModTime = time.Now()

        fst.Tree[root].VerVect = make(map[int]int)
        fst.Tree[root].SyncVect = make(map[int]int)
        fst.Tree[root].Parent = root
        fst.Tree[root].Exists = true

        // Initialize VecVect, SyncVect
        for i:=0; i<len(tnt.servers); i++ {
            fst.Tree[root].VerVect[i] = 0
            fst.Tree[root].SyncVect[i] = 0
        }

        tnt.Tree = fst
    } else {
        fmt.Println(dump, "found! Fetching tree...")
        var fst1 FStree
        decoder := gob.NewDecoder(f)
        decoder.Decode(&fst1)
        tnt.Tree = &fst1
    }

    tnt.UpdateTree(root)

    tnt.ParseTree(root, 0)

    f, err = os.OpenFile(dump, os.O_WRONLY | os.O_CREATE, 0777)
    if err != nil {
        log.Println("Error opening file:", err)
    }

    encoder := gob.NewEncoder(f)
    encoder.Encode(tnt.Tree)
    f.Close()
    fmt.Println(dump + " dumped!")
}

package TnT

import (
    "fmt"
    "net"
    "net/rpc"
    "log"
    "os"
    "encoding/gob"
    "time"
    "path/filepath"
)

const (
    RPC_SLEEP_INTERVAL = 100*time.Millisecond
)

// ToDo: Handle the case when args.Path does not exist.
func (tnt *TnTServer) GetVersion(args *GetVersionArgs, reply *GetVersionReply) error {
    fmt.Println("Syncing ",args,tnt)
    fsn := tnt.Tree.MyTree[args.Path]
    reply.IsDir=make(map[string]bool)
	for k,_:=range fsn.Children{
		reply.IsDir[k]=tnt.Tree.MyTree[k].IsDir
	}
    // No need to copy stuff explicitly. Go RPC handles all this!
    // Pritish: added Exists bool to reply. This is needed for SyncSingle, isn't it?
    reply.Exists, reply.VerVect, reply.SyncVect, reply.Children = fsn.Exists, fsn.VerVect, fsn.SyncVect, fsn.Children

    return nil
}

func (tnt *TnTServer) PropagateUp(VersionVector map[int]int64, SyncVector map[int]int64, path string) {

    //path:path of the parent
    //Propagate the changes in a version vector upwards

    fst := tnt.Tree.MyTree // for ease of code

    tnt.mu.Lock()
    for k,v:=range fst[path].VerVect{
        if v < VersionVector[k] {
            fst[path].VerVect[k]=v
        }
    }
    for k,v:=range fst[path].SyncVect {
        if v > SyncVector[k] {
            fst[path].SyncVect[k]=v
        }
    }
    tnt.mu.Unlock()
    tnt.PropagateUp(fst[path].VerVect, fst[path].SyncVect, fst[path].Parent)
}


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

func (tnt *TnTServer) UpdateTree(dir string) {

    //'dir' must be a directory; if it is not then program will crash!!

    fst := tnt.Tree.MyTree

    // (1) explore the file system, and make appropriate changes in FStree
    // (2) "delete" nodes in FStree which are not in the file system

    d, err := os.Open(dir)
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

func (tnt *TnTServer) SyncWrapper(srv int, path string) {
    //Update tree and then sync
    tnt.mu.Lock()
    tnt.UpdateTree(tnt.root+path)
    tnt.mu.Unlock()
    tnt.SyncNow(srv, path, false)
    tnt.LogToFile()
}

func (tnt *TnTServer) SyncNow(srv int, path string, onlySync bool) {

    fst := tnt.Tree.MyTree // for ease of code

    if onlySync == true {
        parent := fst[path].Parent

        setMaxVersionVect(fst[path].SyncVect, fst[parent].SyncVect )
	return
    }

    //Sync Recursively
    args:=&GetVersionArgs{Path:path}
    var reply GetVersionReply
    for {
        ok:=call(tnt.servers[srv], "TnTServer.GetVersion", args, &reply)
        if ok {
            break
        }
        time.Sleep(RPC_SLEEP_INTERVAL) //Pritish: added some sleep between successive RPCs
    }

    mA_vs_sB := compareVersionVects(reply.VerVect, fst[path].SyncVect)
    //mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect)

    if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
        onlySync = true 	
    } else {
        //Check if the children are consistent
        for k, _ := range reply.Children {
            _, present := fst[path].Children[k]
            if !present {
                tnt.mu.Lock()
            	fst[path].Children[k]=true;
            	fs_node:=new(FSnode);
            	fs_node.VerVect=make(map[int]int64)
            	fs_node.SyncVect=make(map[int]int64)
            	fs_node.Exists=false
            	fs_node.Children=make(map[string]bool)
            	fs_node.Name=k
            	fs_node.Parent=path
            	fs_node.IsDir=reply.IsDir[k]
            	fst[k]=fs_node
            	tnt.mu.Unlock()
          }
        }
    }

    //Case of the leaf node
    if len(fst[path].Children) == 0 {
        tnt.mu.Lock()
        tnt.SyncSingle(srv, path, onlySync, &reply)
        tnt.mu.Unlock()
    } else { // Pritish: added else for clarity of code
        //Case of the non-leaf node
        for k, _ := range fst[path].Children {
            tnt.SyncNow(srv, k, onlySync)
        }
    }
}

func (tnt *TnTServer) SyncSingle(srv int, path string, onlySync bool, reply *GetVersionReply) {
    /*
    (1) Check for updates on local version of file: update modHist, syncHist if required
    (2) Get modHist and syncHist from 'srv'
    (3) Decide between:
        (a) do nothing
        (b) fetch the file
        (c) conflict
    (4) If there is conflict, 
        (a) Check if it's a delete-delete conflict. If yes, then update syncHist and ignore
        (b) If it is some other conflict, then ask the user for action.
        (c) Do appropriate action as specified by user.
        (d) In any case set syncHist appropriately
    */

    fst := tnt.Tree.MyTree

    // A : srv ; B : tnt.me
    mA_vs_sB := compareVersionVects(reply.VerVect, fst[path].SyncVect)
    mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect)

    if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {

        // Do nothing, but update sync history (Can this do anything wrong?!)
        fmt.Println(tnt.me, "has all updates already from", srv)
        setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)

    } else if  mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
        /*
        (1) If reply.Exists == true  : fetch file, set tnt.lastModTime
            If reply.Exists == false : delete local copy, set tnt.lastModTime = time.Now()
        (2) Update modHist and syncHist
        */
        if reply.Exists {
            fmt.Println(tnt.me, "is fetching file from", srv)
            // get file : it should exists on 'srv'
            tnt.CopyFileFromPeer(srv, path, path)
            // set tnt.lastModTime
            fi, err := os.Lstat(tnt.root + path)
            if err != nil {
                log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
            } else {
		fst[path].Exists=true
		fst[path].LastModTime=fi.ModTime()

            }
        } else /* reply.Exists == false */ {
            fmt.Println(tnt.me, "is deleting local copy due to", srv)
            // delete local copy, set tnt.lastModTime = time.Now()
            if(fst[path].IsDir){
              	tnt.DeleteTree(path);
                os.Remove(tnt.root + path)
            } else {
                os.Remove(tnt.root + path)
            }
            fst[path].LastModTime = time.Now()
            fst[path].Exists = false
        }

        // set modHist, syncHist
        setVersionVect(fst[path].VerVect, reply.VerVect)
        setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)

    } else {
        /*
        Four possible cases:
        (1) delete-delete conflict : just ignore
        (2) (a) delete-update conflict
            (b) update-delete conflict
        (3) update-update conflict
        */
          
        if reply.Exists == false && fst[path].Exists == false {
            // Delete-Delete conflict : update syncHist and ignore
            fmt.Println(tnt.me, "Delete-Delete conflict:", srv, "and", tnt.me, "deleted file independently : not really a conflict")
            setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)

        } else if reply.Exists == false && fst[path].Exists == true {

            // Ask user to choose:
            fmt.Println("Delete-Update conflict:", srv, "has deleted, but", tnt.me, "has updated")
            choice := -1
            for choice != tnt.me && choice != srv {
                fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                fmt.Scanf("%d", &choice)
            }

            if choice == tnt.me {
                // If my version is chosen, simply update syncHist
                setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
            } else {
                // Delete local copy, set tnt.exists, tnt.lastModTime and update modHist and syncHist :
                if(fst[path].IsDir){
                    tnt.DeleteTree(path);
                    os.Remove(tnt.root + path)
	        } else {
                    os.Remove(tnt.root + path)
	        }
                fst[path].LastModTime = time.Now()
                fst[path].Exists = false
                setVersionVect(fst[path].VerVect, reply.VerVect)
                setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
            }

        } else if reply.Exists == true {

            /* Update-Delete or Update-Update conflict */
            if fst[path].Exists == false {
                fmt.Println("Update-Delete conflict:", srv, "has update, but", tnt.me, "has deleted")
            } else {
                fmt.Println("Update-Update conflict:", srv, "and", tnt.me, "have updated independently")
            }
            choice := -1
            for choice != tnt.me && choice != srv {
                fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                fmt.Scanf("%d", &choice)
            }

            if choice == tnt.me {
                // If my version is chosen, simply update syncHist
                setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
            } else {
                // Fetch file, set tnt.lastModTime and update tnt.exists, modHist and syncHist :

                // get file
                tnt.CopyFileFromPeer(srv, path, path)
                // set tnt.lastModTime
                fi, err := os.Lstat(tnt.root + path)
                if err != nil {
                    log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
                } else {
                    fst[path].LastModTime = fi.ModTime()
                }
                // set exists, modHist, syncHist
                fst[path].Exists = true
                setVersionVect(fst[path].VerVect, reply.VerVect)
                setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
            }
        }
    }
}

func (tnt *TnTServer) Kill() {
  tnt.dead = true
  tnt.l.Close()
}

func StartServer(servers []string, me int, root string, fstpath string) *TnTServer {
  gob.Register(GetFileArgs{})
  gob.Register(GetDirArgs{})
  tnt := new(TnTServer)
  tnt.me = me
  tnt.servers = servers
  tnt.root = root
  fmt.Println(root+"*~")
  os.Remove(root+"*~")
  tnt.dump = fstpath //root+"FST_watch_new"
  
  f, err := os.Open(tnt.dump)
  defer f.Close()
  if err != nil {
      fmt.Println(tnt.dump, "not found. Creating new tree...")
      fst := new(FStree)
      fst.MyTree = make(map[string]*FSnode)
      fst.MyTree["./"] = new(FSnode)
      fst.MyTree["./"].Name = root
      fst.MyTree["./"].Children = make(map[string]bool)
      fst.MyTree["./"].LastModTime = time.Now()
      tnt.Tree = fst
  } else {
      fmt.Println(tnt.dump, "found! Fetching tree...")
      var fst1 FStree
      decoder := gob.NewDecoder(f)
      decoder.Decode(&fst1)
      tnt.Tree = &fst1
  }
  tnt.UpdateTree(root+"./")
  fmt.Println("in start server",tnt.Tree)

  // RPC set-up borrowed from Lab
  rpcs := rpc.NewServer()
  rpcs.Register(tnt)
  os.Remove(servers[me])
  l, e := net.Listen("unix", servers[me]);
  if e != nil {
      log.Fatal("listen error: ", e);
  }
  tnt.l = l

  go func() {
      for tnt.dead == false {
          conn, err := tnt.l.Accept()
          if err == nil && tnt.dead == false {
              go rpcs.ServeConn(conn)
          } else if err == nil {
              conn.Close()
          }
          if err != nil && tnt.dead == false {
              fmt.Printf("TnTServer(%v) accept: %v\n", me, err.Error())
              tnt.Kill()
          }
      }
  }()

  return tnt
}

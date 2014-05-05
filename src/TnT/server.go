package TnT

import (
  "fmt"
  "net"
  "net/rpc"
  "log"
  "sync"
  "os"
  "io/ioutil"
  "encoding/gob"
  "time"
)




type TnTServer struct {
  mu sync.Mutex
  l net.Listener
  me int
  dead bool
  servers []string
  root string
  Tree *FStree
}

type FStree struct {
    MyTree map[string] *FSnode
}

type FSnode struct {
    Name string
    Size int64
    LastModTime time.Time
    IsDir bool
    Depth int
    LogicalTime int
    Children map[string]bool
    VerVect map[int]int		//ask Zach to modify this
    SyncVect map[int]int	//ask Zach to modify this
    Parent string			//ask Zach to add this
    Exists bool				
}

func (tnt *TnTServer) GetVersion(args *GetVersionArgs, reply *GetVersionReply) error{
	reply.VerVect=make(map[int]int)
	reply.SyncVect=make(map[int]int)
	for k,v:=range tnt.Tree.MyTree[args.Path].VerVect{
		reply.VerVect[k]=v
	}
	for k,v:=range tnt.Tree.MyTree[args.Path].SyncVect{
		reply.SyncVect[k]=v
	}
	for k,v:=range tnt.Tree.MyTree[args.Path].Children{
		reply.Children[k]=v
	}
	
	return nil
}

func (tnt *TnTServer)PropagateUp(VersionVector map[int]int, SyncVector map[int]int, path string){
	//path:path of the parent
	//Propagate the changes in a version vector upwards
	tnt.mu.Lock()
	for k,v:=range tnt.Tree.MyTree[path].VerVect{
		if(v<VersionVector[k]){
			tnt.Tree.MyTree[path].VerVect[k]=v
		}
	}
	for k,v:=range tnt.Tree.MyTree[path].SyncVect{
		if(v>SyncVector[k]){
			tnt.Tree.MyTree[path].SyncVect[k]=v
		}
	}
	tnt.mu.Unlock()
	tnt.PropagateUp(tnt.Tree.MyTree[path].VerVect,tnt.Tree.MyTree[path].SyncVect,tnt.Tree.MyTree[path].Parent)
}

func (tnt *TnTServer) DeleteTree(dir string) {

    fst := tnt.Tree.MyTree

    // Delete all children; recursively delete if child is a directory
    for child, _ := range fst[dir].Children {
        if fst[child].IsDir {
            tnt.DeleteTree(child)
        } else {
            fst[child].Exists = false
            fst[child].LastModTime = time.Now()
            fst[child].VerVect[tnt.me] = fst[child].SyncVect[tnt.me] + 1
            fst[child].SyncVect[tnt.me] += 1
        }
    }
    // Set my own state
    fst[dir].Exists = false
    fst[dir].LastModTime = time.Now()  // note: my LastModTime is higher than any of my children
    fst[dir].VerVect[tnt.me] = fst.Tree[dir].SyncVect[tnt.me] + 1
    fst[dir].SyncVect[tnt.me] += 1
}

func (tnt *TnTServer) UpdateTree(dir string) {

    fst := tnt.Tree.MyTree

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
                if fst[child].LastModTime.Before(fi.ModTime()) {
                    fst[child].Exists = true
                    fst[child].LastModTime = fi.ModTime()
                    fst[child].VerVect[tnt.me] = fst[child].SyncVect[tnt.me] + 1
                    fst[child].SyncVect[tnt.me] += 1
                }
            } else {
                // File is new; so add a new entry in FStree
                fst[child] = new(FSnode)
                fst[child].Name = fi.Name()
                fst[child].Size = fi.Size()
                fst[child].IsDir = fi.IsDir()
                fst[child].LastModTime = fi.ModTime()
                fst[child].VerVect = make(map[int]int)
                fst[child].SyncVect = make(map[int]int)
                fst[child].Parent = dir
                fst[child].Exists = true

                // Initialize VecVect, SyncVect
                for i:=0; i<len(tnt.servers); i++ {
                    fst[child].VerVect[i] = 0
                    fst[child].SyncVect[i] = 0
                }
                fst[child].VerVect[tnt.me] = 1
                fst[child].SyncVect[tnt.me] = 1

                // Make an entry in parent directory
                fst[dir].Children[child] = true // also book-keeping: child really exists in file-system
            }

        } else if fi.IsDir() {
            child := dir + fi.Name() + string(filepath.Separator)

            if fst[child] != nil {
                // Directory is present in FStree; so "update" recursively
                fst[dir].Children[child] = true
                tnt.UpdateTree(child)
            } else {
                // Directory is new; so add a new entry in FStree recursively
                fst[child] = new(FSnode)
                fst[child].Name = fi.Name()
                fst[child].Size = fi.Size()
                fst[child].IsDir = fi.IsDir()
                fst[child].Children = make(map[string]bool)
                fst[child].LastModTime = fi.ModTime()

                fst[child].VerVect = make(map[int]int)
                fst[child].SyncVect = make(map[int]int)
                fst[child].Parent = dir
                fst[child].Exists = true

                // Initialize VecVect, SyncVect
                for i:=0; i<len(tnt.servers); i++ {
                    fst[child].VerVect[i] = 0
                    fst[child].SyncVect[i] = 0
                }
                fst[child].VerVect[tnt.me] = 1
                fst[child].SyncVect[tnt.me] = 1

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
func (tnt *TnTServer) SyncWrapper()(srv int,path string){
	//Update tree
	tnt.mu.Lock()
	tnt.UpdateTree(path)
	tnt.mu.Unlock()
	tnt.SyncNow(srv,path,false)
}

func (tnt *TnTServer)SyncNow(srv int,path string,onlySync bool){
	
	if(onlySync){
		parent:=tnt.Tree.MyTree[path].Parent
		setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect,tnt.Tree.MyTree[parent].SyncVect )
		return
	}
	
	//Sync Recursively
	args:=&GetVersionArgs{Path:path}
	var reply GetVersionReply
	for{
		ok:=call(tnt.Servers[srv],"TnTServer.GetVersion",args,&reply)
		if(ok){
			break
		}
	}
	mA_vs_sB := compareVersionVects(reply.VerVect, tnt.SyncVect)
    mB_vs_sA := compareVersionVects(tnt.VerVect, reply.SyncHist)
	if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
		onlySync=true 	
	    }else{
		//Check if the children are consistent
		for k,_:=range reply.Children{
			_,present:=tnt.Tree.MyTree[path].Children[k]
			if(!present){
				args1:=&GetVersionArgs{Path:k}
				var reply1 GetVersionReply
				for{
					ok:=call(tnt.Servers[srv],"TnTServer.GetVersion",args1,&reply1)
					if(ok){
						break
					}
					}
				tnt.mu.Lock()
				SyncSingle(srv,path,onlySync,reply1)
				tnt.mu.Unlock()
			}
		}
	
		for k,_:=range tnt.Tree.MyTree[path].Children{
			_,present:=reply.Children[k]
			if(!present){
				args1:=&GetVersionArgs{Path:k}
				var reply1 GetVersionReply
				for{
					ok:=call(tnt.Servers[srv],"TnTServer.GetVersion",args1,&reply1)
					if(ok){
						break
					}
					}
				tnt.mu.Lock()
				SyncSingle(srv,path,onlySync,reply1)
				tnt.mu.Unlock()
			}
		}
	}
	//Case of the leaf node
	if(len(tnt.Tree.MyTree[path].Children==0)){
		tnt.mu.Lock()
		SyncSingle(srv,path,onlySync,reply)
		tnt.mu.Unlock()
	}
	//Case of the non-leaf node
	
	for k,_:=tnt.Tree.MyTree[path].Children{
		SyncNow(srv,k,onlySync)
	}
}


func (tnt *TnTServer) SyncSingle(srv int,path string,onlySync bool,reply *GetVersionReply) {
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





	      // A : srv ; B : tnt.me
    	  mA_vs_sB := compareVersionVects(reply.VerVect, tnt.Tree.MyTree[path].SyncVect)
	      mB_vs_sA := compareVersionVects(tnt.Tree.MyTree[path[.VerVect, reply.SyncVect)

	      if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {

          // Do nothing, but update sync history (Can this do anything wrong?!)
    	      fmt.Println(tnt.me, "has all updates already from", srv)
        	  setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)

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

              }
          } else /* reply.Exists == false */ {
              fmt.Println(tnt.me, "is deleting local copy due to", srv)
              // delete local copy, set tnt.lastModTime = time.Now()
              if(tnt.Tree.MyTree[path].IsDir){
              		tnt.DeleteTree(path);
              		os.Remove(tnt.root + path)
              }else{
	              os.Remove(tnt.root + path)
	              }
              tnt.Tree.MyTree[path].lastModTime = time.Now()
              tnt.Tree.MyTree[path].Exists = false
          }

          // set modHist, syncHist
          setVersionVect(tnt.Tree.MyTree[path].ModVect, reply.ModVect)
          setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)

      } else {
          /*
          Four possible cases:
          (1) delete-delete conflict : just ignore
          (2) (a) delete-update conflict
              (b) update-delete conflict
          (3) update-update conflict
          */
          
          if reply.Exists == false && tnt.Tree.MyTree[path].Exists == false {
              // Delete-Delete conflict : update syncHist and ignore
              fmt.Println(tnt.me, "Delete-Delete conflict:", srv, "and", tnt.me, "deleted file independently : not really a conflict")
              setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)

          } else if reply.Exists == false && tnt.Exists == true {

              // Ask user to choose:
              fmt.Println("Delete-Update conflict:", srv, "has deleted, but", tnt.me, "has updated")
              choice := -1
              for choice != tnt.me && choice != srv {
                  fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                  fmt.Scanf("%d", &choice)
              }

              if choice == tnt.me {
                  // If my version is chosen, simply update syncHist
                  setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)
              } else {
                  // Delete local copy, set tnt.exists, tnt.lastModTime and update modHist and syncHist :
                  if(tnt.Tree.MyTree[path].IsDir){
              			tnt.DeleteTree(path);
              			os.Remove(tnt.root + path)
	              }else{
		              os.Remove(tnt.root + path)
	    	          }
                  tnt.Tree.MyTree[path].LastModTime = time.Now()
                  tnt.Tree.MyTree[path].Exists = false
                  setVersionVect(tnt.Tree.MyTree[path].VerVect, reply.VerVect)
                  setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)
              }

          } else if reply.Exists == true {

              /* Update-Delete or Update-Update conflict */

              if tnt.Tree.MyTree[path].Exists == false {
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
                  setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)
              } else {
                  // Fetch file, set tnt.lastModTime and update tnt.exists, modHist and syncHist :

                  // get file
                  tnt.CopyFileFromPeer(srv, path, path)
                  // set tnt.lastModTime
                  fi, err := os.Lstat(tnt.root + path)
                  if err != nil {
                      log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
                  } else {
                      tnt.Tree.MyTree[path].LastModTime = fi.ModTime()
                  }
                  // set exists, modHist, syncHist
                  tnt.Tree.MyTree[path].Exists = true
                  setVersionVect(tnt.Tree.MyTree[path].VerVect, reply.VerVect)
                  setMaxVersionVect(tnt.Tree.MyTree[path].SyncVect, reply.SyncVect)
              }
          }
      }
  
}
func (tnt *TnTServer) LogToFile(){
	f, err = os.OpenFile(tnt.Dump, os.O_WRONLY | os.O_CREATE, 0777)
    if err != nil {
        log.Println("Error opening file:", err)
    }

    encoder := gob.NewEncoder(f)
    encoder.Encode(tnt.Tree)
    f.Close()

}

func (tnt *TnTServer) GetFile(args *GetFileArgs, reply *GetFileReply) error {
  data, err := ioutil.ReadFile(tnt.root + args.FilePath)
  fi, err1 := os.Lstat(tnt.root + args.FilePath)

  if err != nil {
      log.Println(tnt.me, " : Error opening file:", err)
      reply.Err = err
  } else if err1 != nil {
      log.Println(tnt.me, " : Error opening file:", err1)
      reply.Err = err1
  } else {
      reply.Content = data
      reply.Perm = fi.Mode().Perm()
      reply.Err = nil
  }

  return nil
}

func (tnt *TnTServer) CopyFileFromPeer(srv int, path string, dest string) error {
//Handle the directory case  
  args := &GetFileArgs{FilePath:path}
  var reply GetFileReply

  ok := call(tnt.servers[srv], "TnTServer.GetFile", args, &reply)
  if ok {
      if reply.Err != nil {
          log.Println(tnt.me, ": Error opening file:", reply.Err)
      } else {
          err := ioutil.WriteFile(tnt.root + dest, reply.Content, reply.Perm)
          if err != nil {
              log.Println(tnt.me, ": Error writing file:", err)
          }
      }
  } else {
      log.Println(tnt.me, ": GetFile RPC failed")
  }

  return reply.Err
}

func (tnt *TnTServer) kill() {
  tnt.dead = true
  tnt.l.Close()
}

func StartServer(servers []string, me int, root string) *TnTServer {
  	gob.Register(GetFileArgs{})

  	tnt := new(TnTServer)
  	tnt.me = me
  	tnt.servers = servers
  	tnt.root = root
  	fmt.Println(root+"*~")
  	os.Remove(root+"*~")
  	tnt.dump := root+"FST_watch_new"
  
  	f, err := os.Open(tnt.dump)
	defer f.Close()
    if err != nil {
        fmt.Println(tnt.dump, "not found. Creating new tree...")
        fst := new(FStree)
        fst.Tree.MyTree = make(map[string]*FSnode)
        fst.Tree.MyTree[root] = new(FSnode)
        fst.Tree.MyTree[root].Name = root
        fst.Tree.MyTree[root].Children = make(map[string]bool)
        fst.Tree.MyTree[root].LastModTime = time.Now()
        tnt.Tree = fst
    } else {
        fmt.Println(tnt.dump, "found! Fetching tree...")
        var fst1 FStree
        decoder := gob.NewDecoder(f)
        decoder.Decode(&fst1)
        tnt.Tree = &fst1
    }   
    tnt.UpdateTree(root)
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
        tnt.kill()
      }
     }
  	}()

  	return tnt
}

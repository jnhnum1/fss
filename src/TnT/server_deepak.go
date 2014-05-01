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


const (
  EQUAL = 0
  LESSER = 1
  GREATER = 2
  INCOMPARABLE = 3
)
type GetVersionArgs struct{
	Path string
}
type GetVersionReply struct{
	VerVect map[int]int
	SyncVect map[int]int
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

type FStree struct {
    MyTree map[string] *FSnode
}

type FSnode struct {
    Name string
    Size int64
    ModTime time.Time
    IsDir bool
    Depth int
    Children map[string]bool
    VerVect map[int]int		//ask Zach to modify this
    SyncVect map[int]int	//ask Zach to modify this
    parent string			//ask Zach to add this
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
	
	return nil
}

func compareVector(hA map[int]int, hB map[int]int) int {
  /*
  (1) EQUAL - if all sequence numbers match
  (2) LESSER - if all sequence numbers in hA are <= that in hB (at least one is strictly smaller)
  (3) GREATER - if all sequence numbers in hA are >= that in hB (at least one is strictly greater)
  (4) INCOMPARABLE - otherwise
  */
  is_equal := true
  is_lesser := true
  is_greater := true

  for k, _ := range hA {
    if hA[k] < hB[k] {
      is_equal = false
      is_greater = false
    } else if hA[k] > hB[k] {
      is_equal = false
      is_lesser = false
    }
  }

  if is_equal {
      return EQUAL
  } else if is_lesser {
      return LESSER
  } else if is_greater {
      return GREATER
  }

  return INCOMPARABLE
}

func (tnt *TnTServer)PropagateUp(VersionVector map[int]int, SyncVector map[int]int, path string){
	//path:path of the parent
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
	tnt.PropagateUp(tnt.Tree.MyTree[path].VerVect,tnt.Tree.MyTree[path].SyncVect,tnt.Tree.MyTree[path].parent)
}

func (tnt *TnTServer)SyncNow(srv int,path string,onlySync bool){

	if(len(tnt.Tree.MyTree[path].Children==0)){
		//Call single file synchronization here
		//Also handle the special case of a directory
		//Propagate the version vector and sync vector up in single file code
		//Check for onlySync
		return
	}
	args:=&GetVersionArgs{Path:path}
	var reply GetVersionReply
	for{
		ok:=call(tnt.Servers[srv],"TnTServer.GetVersion",args,&reply)
		if(ok){
			break
		}
	}
	mA_vs_sB := compareVectors(reply.VerVect, tnt.SyncVect)
    mB_vs_sA := compareVersionVects(tnt.VerVect, reply.SyncHist)
	if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
		onlySync=true 	
	    }
	for k,_:=tnt.Tree.MyTree[path].Children{
		SyncNow(srv,k,onlySync)
	}
}

/*
//ToDo-Deepak
//This function is supposed to add a file/directory to watch
//Let us assume that it is already recursive
func (tnt *TnTServer) AddWatch() error{
}

//ToDo-Deepak
//This function is supposed to remove a file/directory from watch
func (tnt *TnTServer) RemoveWatch() error{
}

//ToDo-Deepak
//This function implements the two machine sync
//Semantics: I sync with the other machine, i.e. I want to ensure that I am as new as the other machine
func (tnt *TnTServer) Sync() error{
}

//ToDo-Deepak
//This function implements the check on all the files that we have after a crash
func (tnt *TnTServer) CheckAfterCrash() error{
}

//ToDo-Deepak
//This function is supposed to log the version vectors on the disk.
func (tnt *TnTServer) LogToDisk() error{
}
*/
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

    //setup the watch
  tnt.Tree = new(FStree)
  tnt.Tree.MyTree = make(map[string]*FSnode)
  tnt.Tree.MyTree[root] = new(FSnode)
  tnt.Tree.MyTree[root].Name = root
  tnt.Tree.MyTree[root].Depth = 0
  tnt.Tree.MyTree[root].Children = make(map[string]bool)
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

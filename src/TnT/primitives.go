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
  if(tnt.Tree.MyTree[path].IsDir){
  	args := &GetDirInfo{Path:path}
  	var reply GetDirReply
  	ok := call(tnt.servers[srv], "TnTServer.GetDir", args, &reply)
	if ok {
    	if reply.Err != nil {
        	log.Println(tnt.me, ": Error opening Directory:", reply.Err)
	      } else {
	      	
    	      err := os.MkDir(tnt.root + dest, reply.Perm)
        	  if err != nil {
            	  log.Println(tnt.me, ": Error writing file:", err)
	          }
    	  }
	  } else {
    	  log.Println(tnt.me, ": GetDir RPC failed")
	  }
  	
  }else{
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
	}

  return reply.Err
}

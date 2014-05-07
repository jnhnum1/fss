package TnT

import (
  "net"
  "log"
  "os"
  "io/ioutil"
  "encoding/gob"
  "time"
  "sync"
  "fmt"
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
    dump string
}

func (tnt *TnTServer) LogToFile(){

    f, err := os.OpenFile(tnt.dump, os.O_WRONLY | os.O_CREATE, 0777)
    if err != nil {
        log.Println("Error opening file:", err)
    }

    encoder := gob.NewEncoder(f)
    encoder.Encode(tnt.Tree)
    f.Close()
}


func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
}

func (tnt *TnTServer) ParseTree(path string, depth int) {
    fst := tnt.Tree.MyTree

    //spaces(depth)
    fmt.Println(fst[path].Name, ":", fst[path].LastModTime, fst[path].Exists, fst[path].VerVect, fst[path].SyncVect)

    if fst[path].IsDir {
        for child, _ := range fst[path].Children {
            spaces(depth)
            //fmt.Println("--", path, "-", child, "fst[path].Children[child]:", fst[path].Children[child])
            //fmt.Printf("%t : ", fst[path].Children[child])
            tnt.ParseTree(child, depth+1)
        }
    }
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

func (tnt *TnTServer) GetDir(args *GetDirArgs, reply *GetDirReply) error {

  fi, err1 := os.Lstat(tnt.root + args.Path)

  if err1 != nil {
      log.Println(tnt.me, " : Error opening file:", err1)
      reply.Err = err1
  } else {
      reply.Perm = fi.Mode().Perm()
      reply.Err = nil
  }

  return nil
}

func (tnt *TnTServer) CopyFileFromPeer(srv int, path string, dest string) error {
  //Handle the directory case  
  if(tnt.Tree.MyTree[path].IsDir){
  	args := &GetDirArgs{Path:path}
  	var reply GetDirReply
  	ok := call(tnt.servers[srv], "TnTServer.GetDir", args, &reply)
	if ok {
    	if reply.Err != nil {
        	log.Println(tnt.me, ": Error opening Directory:", reply.Err)
	      } else {
	      	
    	      err := os.Mkdir(tnt.root + dest, reply.Perm)
	      	
        	  if err != nil {
            	  	log.Println(tnt.me, ": Error writing file:", err)
	          }else{
			tnt.Tree.MyTree[path].Exists=true
		}
    	  }
	  } else {
    	  log.Println(tnt.me, ": GetDir RPC failed")
	  }
	  return reply.Err
  	
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
	  return reply.Err
	}


}

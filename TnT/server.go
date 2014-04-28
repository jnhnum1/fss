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
)

type TnTServer struct {
  mu sync.Mutex
  l net.Listener
  me int
  dead bool
  servers []string
  root string
}

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

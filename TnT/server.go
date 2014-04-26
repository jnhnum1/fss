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

func (tnt *TnTServer) GetFile(args *GetFileArgs, reply *GetFileReply) error {
  data, err := ioutil.ReadFile(tnt.root + args.FilePath)
  
  if err != nil {
    log.Println(tnt.me, " : Error opening file:", err)
    reply.Err = err
  } else {
    reply.Content = data
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
          err := ioutil.WriteFile(tnt.root + dest, reply.Content, 0644)
          if err != nil {
              log.Println(tnt.me, ": Error writing file:", err)
          }
      }
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

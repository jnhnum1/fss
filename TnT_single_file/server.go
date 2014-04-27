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

  fname string //the sole file to be in sync
  lastseq int
  mod_hist map[int]Version
  sync_hist map[int]Version
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

func (tnt *TnTServer) GetHistory(args *GetHistoryArgs, reply *GetHistoryReply) error {
  
  reply.ModHist = tnt.mod_hist
  reply.SyncHist = tnt.sync_hist

  tnt.lastseq += 1
  tnt.mod_hist[args.Me] = Version{seq:tnt.lastseq, timestamp: time.Now()}

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

func (tnt *TnTServer) SyncNow(srv int) {

  args := &GetHistoryArgs{}
  var reply GetHistoryReply

  ok := call(tnt.servers[srv], "TnTServer.GetHistory", args, &reply)
  if ok {
      

  } else {
      log.Println(tnt.me, ": GetHistory RPC failed")
  }
}

func (tnt *TnTServer) kill() {
  tnt.dead = true
  tnt.l.Close()
}

func StartServer(servers []string, me int, root string, fname string) *TnTServer {
  gob.Register(GetFileArgs{})

  tnt := new(TnTServer)
  tnt.me = me
  tnt.servers = servers
  tnt.root = root
  tnt.fname = fname
  tnt.lastseq = 0
  tnt.mod_hist = make(map[int]Version)
  tnt.sync_hist = make(map[int]Version)

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

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

type TnTServer struct {
  mu sync.Mutex
  l net.Listener
  me int
  dead bool
  servers []string
  root string

  fname string //the sole file to be in sync
  lastModTime time.Time
  modHist map[int]int
  syncHist map[int]int
}

func compareVersionVects(hA map[int]int, hB map[int]int) int {
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

func setVersionVect(hA map[int]int, hB map[int]int) {
  /* Sets hA = hB */
  for k, v := range hB {
    hA[k] = v
  }
}

func (tnt *TnTServer) GetFile(args *GetFileArgs, reply *GetFileReply) error {
  /*
  (1) Open file in (tnt.root+args.FilePath)
      (a) read file into bytes
      (b) read permissions of file
  (2) Put (file, permissions) into 'reply'
  */
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
  /*
  (1) Check for updates on local version of file: update modHist, syncHist if required
  (2) Send over modHist, syncHist through 'reply'
  */

  fi, err := os.Lstat(tnt.root + tnt.fname)
  if err != nil {
      log.Println(tnt.me, ": File does not exist:", err)
  } else {
      if tnt.lastModTime.Before(fi.ModTime()) {
          tnt.lastModTime = fi.ModTime()
          tnt.modHist[tnt.me] = tnt.syncHist[tnt.me] + 1
          tnt.syncHist[tnt.me] += 1
      }
  }

  reply.ModHist = tnt.modHist
  reply.SyncHist = tnt.syncHist

  return nil
}

func (tnt *TnTServer) CopyFileFromPeer(srv int, path string, dest string) error {
  /*
  (1) Call 'GetFile' RPC on server 'srv'
  (2) Write the file preserving permissions
  */
  
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
  /*
  (1) Check for updates on local version of file: update modHist, syncHist if required
  (2) Get modHist and syncHist from 'srv'
  (3) Decide between:
      (a) do nothing
      (b) fetch the file
      (c) conflict
  (4) If there is conflict, then ask the user for action.
      (a) If user asks to fetch, then fetch.
      (b) In either case set syncHist appropriately
  */

  fmt.Println("Machine", tnt.me, "syncing from machine", srv)

  fi, err := os.Lstat(tnt.root + tnt.fname)
  if err != nil {
      log.Println(tnt.me, ": File does not exist:", err)
  } else {
      if tnt.lastModTime.Before(fi.ModTime()) {
          tnt.lastModTime = fi.ModTime()
          tnt.modHist[tnt.me] = tnt.syncHist[tnt.me] + 1
          tnt.syncHist[tnt.me] += 1
      }
  }

  args := &GetHistoryArgs{}
  var reply GetHistoryReply

  ok := call(tnt.servers[srv], "TnTServer.GetHistory", args, &reply)

  if ok {

      fmt.Println("Printing Histories:")
      fmt.Println(tnt.me, "Mine :")
      fmt.Println("  modHist :", tnt.modHist)
      fmt.Println("  syncHist:", tnt.syncHist)
      fmt.Println("  lastModTime:", tnt.lastModTime)
      fmt.Println(srv, "Peer :")
      fmt.Println("  modHist :", reply.ModHist)
      fmt.Println("  syncHist:", reply.SyncHist)

      // A : srv ; B : tnt.me
      mA_vs_sB := compareVersionVects(reply.ModHist, tnt.syncHist)
      mB_vs_sA := compareVersionVects(tnt.modHist, reply.SyncHist)

      if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
          // do nothing
          fmt.Println(tnt.me, "has all updates already from", srv)
          // Update sync history. Can this do anything wrong?!
          for k, v := range reply.SyncHist {
              if tnt.syncHist[k] < v {
                  tnt.syncHist[k] = v
              }
          }
      } else if  mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
          // Fetch file, set tnt.lastModTime and update modHist and syncHist :
          fmt.Println(tnt.me, "is fetching file from", srv)
          // get file
          tnt.CopyFileFromPeer(srv, tnt.fname, tnt.fname)
          // set tnt.lastModTime
          fi, err := os.Lstat(tnt.root + tnt.fname)
          if err != nil {
              log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
          } else {
              tnt.lastModTime = fi.ModTime()
          }
          // set modHist, syncHist
          setVersionVect(tnt.modHist, reply.ModHist)
          for k, v := range reply.SyncHist {
              if tnt.syncHist[k] < v {
                  tnt.syncHist[k] = v
              }
          }
      } else {
          // report conflict : ask for resolution
          fmt.Println(tnt.me, "conflicts with", srv)

          fmt.Printf("Conflict on %d syncing from %d\n", tnt.me, srv)
          choice := -1
          for choice != tnt.me && choice != srv {
              fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
              fmt.Scanf("%d", &choice)
          }

          if choice == tnt.me {
              // If my version is chosen, simply update syncHist
              for k, v := range reply.SyncHist {
                  if tnt.syncHist[k] < v {
                      tnt.syncHist[k] = v
                  }
              }
          } else {
              // Fetch file, set tnt.lastModTime and update modHist and syncHist :

              // get file
              tnt.CopyFileFromPeer(srv, tnt.fname, tnt.fname)
              // set tnt.lastModTime
              fi, err := os.Lstat(tnt.root + tnt.fname)
              if err != nil {
                  log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
              } else {
                  tnt.lastModTime = fi.ModTime()
              }
              // set modHist, syncHist
              setVersionVect(tnt.modHist, reply.ModHist)
              for k, v := range reply.SyncHist {
                  if tnt.syncHist[k] < v {
                      tnt.syncHist[k] = v
                  }
              }
          }
      }
  } else {
      log.Println(tnt.me, ": GetHistory RPC failed")
  }
}

func (tnt *TnTServer) Kill() {
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

  fi, err := os.Lstat(root+fname)
  if err == nil {
      tnt.lastModTime = fi.ModTime()
  } else {
      tnt.lastModTime = time.Now()
  }

  tnt.modHist = make(map[int]int)
  tnt.syncHist = make(map[int]int)
  for i:=0; i<len(servers); i++ {
      tnt.modHist[i] = 0
      tnt.syncHist[i] = 0
  }

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

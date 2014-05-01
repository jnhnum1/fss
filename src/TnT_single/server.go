package TnT_single

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

  fname string // the name of the sole file to be in sync
  exists bool // true if a local copy of file exists
  lastModTime time.Time
  modHist map[int]int
  syncHist map[int]int
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


func (tnt *TnTServer) UpdateLocalState() {
  /*
    Check for updates on local version of file: update modHist, syncHist if required
    UpdateLocalState() is called at the beginning of GetHistory() RPC and in SyncNow()
  */
  fi, err := os.Lstat(tnt.root + tnt.fname)
  if err != nil {
      if tnt.exists {
          tnt.exists = false
          tnt.lastModTime = time.Now()
          tnt.modHist[tnt.me] = tnt.syncHist[tnt.me] + 1
          tnt.syncHist[tnt.me] += 1
      }
  } else {
      if tnt.lastModTime.Before(fi.ModTime()) {
          tnt.exists = true
          tnt.lastModTime = fi.ModTime()
          tnt.modHist[tnt.me] = tnt.syncHist[tnt.me] + 1
          tnt.syncHist[tnt.me] += 1
      }
  }
}

func (tnt *TnTServer) GetHistory(args *GetHistoryArgs, reply *GetHistoryReply) error {
  /*
  (1) Check for updates on local version of file: update modHist, syncHist if required
  (2) Send over exists, modHist, syncHist through 'reply'
  */

  tnt.UpdateLocalState()

  reply.Exists, reply.ModHist, reply.SyncHist = tnt.exists, tnt.modHist, tnt.syncHist

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
          log.Println("CopyFileFromPeer:", tnt.me, ": Error opening file:", reply.Err)
      } else {
          err := ioutil.WriteFile(tnt.root + dest, reply.Content, reply.Perm)
          if err != nil {
              log.Println("CopyFileFromPeer:", tnt.me, ": Error writing file:", err)
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
  (4) If there is conflict, 
      (a) Check if it's a delete-delete conflict. If yes, then update syncHist and ignore
      (b) If it is some other conflict, then ask the user for action.
      (c) Do appropriate action as specified by user.
      (d) In any case set syncHist appropriately
  */

  fmt.Println("Machine", tnt.me, "syncing from machine", srv)

  tnt.UpdateLocalState()

  args := &GetHistoryArgs{}
  var reply GetHistoryReply
  ok := call(tnt.servers[srv], "TnTServer.GetHistory", args, &reply)

  if ok {

      fmt.Println("Printing Histories:")
      fmt.Println(tnt.me, "Mine :")
      fmt.Println("  exists  :", tnt.exists)
      fmt.Println("  modHist :", tnt.modHist)
      fmt.Println("  syncHist:", tnt.syncHist)
      fmt.Println("  lastModTime:", tnt.lastModTime)
      fmt.Println(srv, "Peer :")
      fmt.Println("  exists  :", reply.Exists)
      fmt.Println("  modHist :", reply.ModHist)
      fmt.Println("  syncHist:", reply.SyncHist)

      // A : srv ; B : tnt.me
      mA_vs_sB := compareVersionVects(reply.ModHist, tnt.syncHist)
      mB_vs_sA := compareVersionVects(tnt.modHist, reply.SyncHist)

      if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {

          // Do nothing, but update sync history (Can this do anything wrong?!)
          fmt.Println(tnt.me, "has all updates already from", srv)
          setMaxVersionVect(tnt.syncHist, reply.SyncHist)

      } else if  mB_vs_sA == LESSER || mB_vs_sA == EQUAL {

          /*
          (1) If reply.Exists == true  : fetch file, set tnt.lastModTime
              If reply.Exists == false : delete local copy, set tnt.lastModTime = time.Now()
          (2) Update modHist and syncHist
          */
          if reply.Exists {
              fmt.Println(tnt.me, "is fetching file from", srv)
              // get file : it should exists on 'srv'
              tnt.CopyFileFromPeer(srv, tnt.fname, tnt.fname)
              // set tnt.lastModTime
              fi, err := os.Lstat(tnt.root + tnt.fname)
              if err != nil {
                  log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
              } else {
                  tnt.exists = true
                  tnt.lastModTime = fi.ModTime()
              }
          } else /* reply.Exists == false */ {
              fmt.Println(tnt.me, "is deleting local copy due to", srv)
              // delete local copy, set tnt.lastModTime = time.Now()
              os.Remove(tnt.root + tnt.fname)
              tnt.lastModTime = time.Now()
              tnt.exists = false
          }

          // set modHist, syncHist
          setVersionVect(tnt.modHist, reply.ModHist)
          setMaxVersionVect(tnt.syncHist, reply.SyncHist)

      } else {
          /*
          Four possible cases:
          (1) delete-delete conflict : just ignore
          (2) (a) delete-update conflict
              (b) update-delete conflict
          (3) update-update conflict
          */
          
          if reply.Exists == false && tnt.exists == false {
              // Delete-Delete conflict : update syncHist and ignore
              fmt.Println(tnt.me, "Delete-Delete conflict:", srv, "and", tnt.me, "deleted file independently : not really a conflict")
              setMaxVersionVect(tnt.syncHist, reply.SyncHist)

          } else if reply.Exists == false && tnt.exists == true {

              // Ask user to choose:
              fmt.Println("Delete-Update conflict:", srv, "has deleted, but", tnt.me, "has updated")
              choice := -1
              for choice != tnt.me && choice != srv {
                  fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                  fmt.Scanf("%d", &choice)
              }

              if choice == tnt.me {
                  // If my version is chosen, simply update syncHist
                  setMaxVersionVect(tnt.syncHist, reply.SyncHist)
              } else {
                  // Delete local copy, set tnt.exists, tnt.lastModTime and update modHist and syncHist :
                  os.Remove(tnt.root + tnt.fname)
                  tnt.lastModTime = time.Now()
                  tnt.exists = false
                  setVersionVect(tnt.modHist, reply.ModHist)
                  setMaxVersionVect(tnt.syncHist, reply.SyncHist)
              }

          } else if reply.Exists == true {

              /* Update-Delete or Update-Update conflict */

              if tnt.exists == false {
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
                  setMaxVersionVect(tnt.syncHist, reply.SyncHist)
              } else {
                  // Fetch file, set tnt.lastModTime and update tnt.exists, modHist and syncHist :

                  // get file
                  tnt.CopyFileFromPeer(srv, tnt.fname, tnt.fname)
                  // set tnt.lastModTime
                  fi, err := os.Lstat(tnt.root + tnt.fname)
                  if err != nil {
                      log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
                  } else {
                      tnt.lastModTime = fi.ModTime()
                  }
                  // set exists, modHist, syncHist
                  tnt.exists = true
                  setVersionVect(tnt.modHist, reply.ModHist)
                  setMaxVersionVect(tnt.syncHist, reply.SyncHist)
              }
          }
      }
  } else {
      log.Println(tnt.me, ": GetHistory RPC failed - try later")
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

  tnt.modHist = make(map[int]int)
  tnt.syncHist = make(map[int]int)
  for i:=0; i<len(servers); i++ {
      tnt.modHist[i] = 0
      tnt.syncHist[i] = 0
  }

  fi, err := os.Lstat(root+fname)
  if err == nil {
      tnt.exists = true
      tnt.modHist[tnt.me] = 1
      tnt.syncHist[tnt.me] = 1
      tnt.lastModTime = fi.ModTime()
  } else {
      tnt.exists = false
      tnt.lastModTime = time.Now()
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

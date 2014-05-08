package TnT_single_v2

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

  DO_NOTHING = 0
  UPDATE = 1
  DELETE = 2
)

type TnTServer struct {
  mu sync.Mutex
  l net.Listener
  me int
  dead bool
  servers []string
  root string

  fname string // the name of the sole file to be in sync
  Exists bool // true if a local copy of file Exists; won't be needed later, as you just need to look up in the FStree if the node Exists
  LastModTime time.Time
  LogicalTime int64
  Creator int
  CreationTime int64
  VerVect map[int]int64
  SyncVect map[int]int64
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
      Check for updates on local version of file: update VerVect, SyncVect if required
      UpdateLocalState() is called at the beginning of GetHistory() RPC and in SyncNow()
    */

    tnt.LogicalTime += 1

    tnt.SyncVect[tnt.me] = tnt.LogicalTime

    fi, err := os.Lstat(tnt.root + tnt.fname)
    if err != nil {
        if tnt.Exists {
            tnt.Exists = false // delete from FST
        }
    } else {
        if tnt.Exists == false {
            tnt.Exists = true
            tnt.Creator = tnt.me
            tnt.CreationTime = tnt.LogicalTime
            tnt.LastModTime = fi.ModTime()
            // tnt.VerVect = make(map[int]int64)
            for i:=0; i<len(tnt.servers); i++ {
                tnt.VerVect[i] = 0
            }
            tnt.VerVect[tnt.me] = tnt.LogicalTime
        } else if tnt.LastModTime.Before(fi.ModTime()) {
            tnt.LastModTime = fi.ModTime()
            tnt.VerVect[tnt.me] = tnt.LogicalTime
        }
    }
}

func (tnt *TnTServer) GetHistory(args *GetHistoryArgs, reply *GetHistoryReply) error {
  /*
  (1) Check for updates on local version of file: update VerVect, SyncVect if required
  (2) Send over Exists, VerVect, SyncVect through 'reply'
  */

  tnt.UpdateLocalState()

  reply.Exists, reply.Creator, reply.CreationTime, reply.VerVect, reply.SyncVect = tnt.Exists, tnt.Creator, tnt.CreationTime, tnt.VerVect, tnt.SyncVect

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
    (1) Check for updates on local version of file: update VerVect, SyncVect if required
    (2) Get VerVect and SyncVect from 'srv'
    (3) Decide between:
        (a) do nothing
        (b) fetch the file
        (c) conflict
    (4) If there is conflict, 
        (a) Check if it's a delete-delete conflict. If yes, then update SyncVect and ignore
        (b) If it is some other conflict, then ask the user for action.
        (c) Do appropriate action as specified by user.
        (d) In any case set SyncVect appropriately
    */

    fmt.Println("Machine", tnt.me, "syncing from machine", srv)

    tnt.UpdateLocalState()

    args := &GetHistoryArgs{}
    var reply GetHistoryReply
    ok := call(tnt.servers[srv], "TnTServer.GetHistory", args, &reply)

    if ok == false {
        log.Println(tnt.me, ": GetHistory RPC failed - try later")
    } else {

        fmt.Println("Printing Histories:")
        fmt.Println(tnt.me, "Mine :")
        fmt.Println("  Exists  :", tnt.Exists)
        fmt.Println("  VerVect :", tnt.VerVect)
        fmt.Println("  SyncVect:", tnt.SyncVect)
        fmt.Println("  Creator :", tnt.Creator)
        fmt.Println("  CreTime :", tnt.CreationTime)
        fmt.Println("  LastModTime:", tnt.LastModTime)
        fmt.Println(srv, "Peer :")
        fmt.Println("  Exists  :", reply.Exists)
        fmt.Println("  VerVect :", reply.VerVect)
        fmt.Println("  SyncVect:", reply.SyncVect)
        fmt.Println("  Creator :", reply.Creator)
        fmt.Println("  CreTime :", reply.CreationTime)

        // A : srv ; B : tnt.me
        action := DO_NOTHING
        choice := -1
        if reply.Exists == false && tnt.Exists == false {
            action = DO_NOTHING
        } else if reply.Exists == false && tnt.Exists == true {
            if reply.SyncVect[tnt.Creator] < tnt.CreationTime {
                action = DO_NOTHING
            } else if mB_vs_sA := compareVersionVects(tnt.VerVect, reply.SyncVect); mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
                action = DELETE
            } else {
                // Delete-Update conflict
                fmt.Println("Delete-Update conflict:", srv, "has deleted, but", tnt.me, "has updated")
                for choice != tnt.me && choice != srv {
                    fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                    fmt.Scanf("%d", &choice)
                }
                if choice == tnt.me {
                    action = DO_NOTHING
                } else {
                    action = DELETE
                }
            }
        } else if reply.Exists == true && tnt.Exists == false {
            mA_vs_sB := compareVersionVects(reply.VerVect, tnt.SyncVect)
            if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
                action = DO_NOTHING
            } else if tnt.SyncVect[reply.Creator] < reply.CreationTime {
                action = UPDATE
            } else {
                // Update-Delete conflict
                fmt.Println("Update-Delete conflict:", srv, "has updated, but", tnt.me, "has deleted")
                for choice != tnt.me && choice != srv {
                    fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                    fmt.Scanf("%d", &choice)
                }
                if choice == tnt.me {
                    action = DO_NOTHING
                } else {
                    action = UPDATE
                }
            }
        } else /* reply.Exists == true && tnt.Exists == true */ {
            mA_vs_sB := compareVersionVects(reply.VerVect, tnt.SyncVect)
            mB_vs_sA := compareVersionVects(tnt.VerVect, reply.SyncVect)
            if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
                action = DO_NOTHING
            } else if  mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
                action = UPDATE
            } else {
                // Update-Update conflict
                fmt.Println("Update-Update conflict:", srv, "and", tnt.me, "have updated independently")
                for choice != tnt.me && choice != srv {
                    fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
                    fmt.Scanf("%d", &choice)
                }
                if choice == tnt.me {
                    action = DO_NOTHING
                } else {
                    action = UPDATE
                }
            }
        }

        if action == DO_NOTHING {
            fmt.Println("ACTION:", tnt.me, "has nothing to do")
            setMaxVersionVect(tnt.SyncVect, reply.SyncVect)
        } else if action == DELETE {
            fmt.Println("ACTION:", tnt.me, "is deleting file due to", srv)
            os.Remove(tnt.root + tnt.fname)
            tnt.Exists = false
            // delete(tnt.VerVect)
            setMaxVersionVect(tnt.SyncVect, reply.SyncVect)
        } else if action == UPDATE {
            fmt.Println("ACTION:", tnt.me, "is getting file from", srv)
            // get file
            tnt.CopyFileFromPeer(srv, tnt.fname, tnt.fname)
            // set tnt.LastModTime
            fi, err := os.Lstat(tnt.root + tnt.fname)
            if err != nil {
                log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
            } else {
                tnt.LastModTime = fi.ModTime()
            }
            // set Exists, VerVect, SyncVect
            tnt.Exists = true
            tnt.Creator, tnt.CreationTime = reply.Creator, reply.CreationTime
            setVersionVect(tnt.VerVect, reply.VerVect)
            setMaxVersionVect(tnt.SyncVect, reply.SyncVect)
        }
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

  tnt.VerVect = make(map[int]int64)
  tnt.SyncVect = make(map[int]int64)
  for i:=0; i<len(servers); i++ {
      tnt.VerVect[i] = 0
      tnt.SyncVect[i] = 0
  }
  tnt.LogicalTime = 1

  fi, err := os.Lstat(root+fname)
  if err == nil {
      tnt.Exists = true
      tnt.Creator = tnt.me
      tnt.CreationTime = 1
      tnt.VerVect[tnt.me] = 1
      tnt.SyncVect[tnt.me] = 1
      tnt.LastModTime = fi.ModTime()
  } else {
      tnt.Exists = false
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

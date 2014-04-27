package TnT

import (
  "fmt"
  "net/rpc"
  "os"
)

type Version struct {
  seq int
  timestamp time.Time
}

type GetFileArgs struct {
  FilePath string
}

type GetFileReply struct {
  Content []byte
  Perm os.FileMode
  Err error
}

type GetHistoryArgs struct {
  Me int
}

type GetHistoryReply struct {
  ModHist map[int]Version
  SyncHist map[int]Version
}

// 'call' function from Labs :

func call(srv string, rpcname string,
          args interface{}, reply interface{}) bool {
  c, errx := rpc.Dial("unix", srv)
  if errx != nil {
    return false
  }
  defer c.Close()
    
  err := c.Call(rpcname, args, reply)
  if err == nil {
    return true
  }

  fmt.Println(err)
  return false
}

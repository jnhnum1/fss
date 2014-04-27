package TnT

import (
  "fmt"
  "net/rpc"
  "os"
)

type GetFileArgs struct {
  FilePath string
}

type GetFileReply struct {
  Content []byte
  Perm os.FileMode
  Err error
}

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

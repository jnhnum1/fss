package TnT

import (
  "fmt"
  "net/rpc"
  "os"
)

const(
  IN_MODIFY = 0x2
  IN_CREATE = 0x100
  IN_DELETE = 0x200
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

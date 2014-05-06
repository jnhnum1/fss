package TnT

import (
  "fmt"
  "net/rpc"
  "os"
)

const(
  IN_ATTRIB = 0x4
  IN_CLOSE = 0x8
  IN_CREATE = 0x100
  IN_CREATE_ISDIR = 0x40000100
  IN_DELETE = 0x200
  IN_DELETE_ISDIR = 0x40000200
  IN_IGNORED = 0x8000
  IN_MODIFY = 0x2
  IN_MOVE_FROM = 0x40
  IN_MOVE_TO = 0x80
  IN_OPEN = 0x20
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

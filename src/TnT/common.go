package TnT

import (
  "fmt"
  "net/rpc"
  "os"
)

const (
  EQUAL = 0
  LESSER = 1
  GREATER = 2
  INCOMPARABLE = 3
)

type GetVersionArgs struct{
  Path string
}

type GetVersionReply struct{
  Exists bool
  VerVect map[int]int64
  SyncVect map[int]int64
  Children map[string]bool
  IsDir map[string]bool //For children
}

type GetFileArgs struct {
  FilePath string
}

type GetFileReply struct {
  Content []byte
  Perm os.FileMode
  Err error
}


func compareVersionVects(hA map[int]int64, hB map[int]int64) int {
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

func setVersionVect(hA map[int]int64, hB map[int]int64) {
  /* For all k, sets hA[k] = hB[k] */
  for k, v := range hB {
    hA[k] = v
  }
}

func setMaxVersionVect(hA map[int]int64, hB map[int]int64) {
  /* For all k, sets hA[k] = max(hA[k], hB[k]) */
  for k, v := range hB {
    if hA[k] < v {
        hA[k] = v
    }
  }
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

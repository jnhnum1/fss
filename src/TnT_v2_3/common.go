package TnT_v2_3

import (
	"fmt"
	"net/rpc"
	"os"
	"path/filepath"
	"crypto/rand"
	"math/big"
	"strconv"

)

const (
	EQUAL = 0
	LESSER = 1
	GREATER = 2
	INCOMPARABLE = 3
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
  	IN_OPEN_ISDIR = 0x40000020
  	IN_CLOSE_ISDIR = 0x40000010
)

type NewData struct {
	TmpName string
	Path string
	IsDir bool
	Perm os.FileMode
}

type DelData struct {
	Path string
	IsDir bool
}

type GetVersionArgs struct{
	Path string
}

type GetVersionReply struct{
	Exists bool
	Creator int
	CreationTime int64
	VerVect map[int]int64
	SyncVect map[int]int64
	Children map[string]bool
	IsDir map[string]bool //For children : why do you need this? Can't we encode this in Children itself?
}

type GetFileArgs struct {
	FilePath string
}

type GetFileReply struct {
	Content []byte
	Perm os.FileMode
	Err error
}

type GetDirArgs struct{
	Path string
}

type GetDirReply struct{
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

func setMinVersionVect(hA map[int]int64, hB map[int]int64) {
	/* For all k, sets hA[k] = max(hA[k], hB[k]) */
	for k, v := range hB {
		if hA[k] > v {
			hA[k] = v
		}
	}
}

func MaxVersionVect(hA map[int]int64, hB map[int]int64) map[int]int64 {
	/* For all k, sets hA[k] = max(hA[k], hB[k]) */
	hC := make(map[int]int64)
	for k, v := range hB {
		if hA[k] < v {
			hC[k] = v
		} else {
			hC[k] = hA[k]
		}
	}
	return hC
}

func parent(path string) string {
	/*
	Gives the path of the parent. For example,
	(1) "./root/nest/tra/foo" will gives "./root/nest/tra/"
	(2) "./root/nest/tra/foo/" also gives "./root/nest/tra/"
	(3) If input does not contain a "/", then it will return ""
	*/
	if len(path) == 0 {
		return path
	}
	end := len(path) - 1
	if path[end] == filepath.Separator {
		end--
	}
	for ; end >= 0; end-- {
		if path[end] == filepath.Separator {
			break
		}
	}
	return path[0:end+1]
}

func rand_string(n int) string {
	max := big.NewInt(int64(1) << 62)
	rstr := ""
	for i:=0; i<n; i++ {
		bigx, _ := rand.Int(rand.Reader, max)
		x := bigx.Int64()
		rstr += strconv.FormatInt(x, 16)
	}
	return rstr
}

// 'call' function from Labs :

func call(srv string, rpcname string,
            args interface{}, reply interface{}) bool {
    fmt.Println("Calling:",srv,rpcname,args,reply)
	c, errx := rpc.DialHTTP("tcp", srv + ":1235")
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

package TnT_v2_1

import (
	"net"
	"log"
	"os"
	"io/ioutil"
	"encoding/gob"
	"time"
	"sync"
	"fmt"
)

type FSnode struct {
	Name string
	Size int64
	IsDir bool
	Children map[string]bool
	LastModTime time.Time
	Creator int
	CreationTime int64		//ask Zach to add this
	VerVect map[int]int64	//ask Zach to modify this
	SyncVect map[int]int64	//ask Zach to modify this
	Parent string		//ask Zach to add this
}

type FStree struct {
	LogicalTime int64
	MyTree map[string] *FSnode
}

type TnTServer struct {
	mu sync.Mutex
	l net.Listener
	me int
	dead bool
	servers []string
	root string
	Tree *FStree
	dump string
}

func (tnt *TnTServer) LogToFile(){

	f, err := os.OpenFile(tnt.dump, os.O_WRONLY | os.O_CREATE, 0777)
	if err != nil {
		log.Println("Error opening file:", err)
	}

	encoder := gob.NewEncoder(f)
	encoder.Encode(tnt.Tree)
	f.Close()
}

func (tnt *TnTServer) LiveAncestor(path string) string {

	fst := tnt.Tree.MyTree

	prt := parent(path)
	_, prt_present := fst[prt]
	for prt_present == false {
		prt = parent(prt)
		_, prt_present = fst[prt]
	}
	return prt
}

func spaces(depth int) {
	for i:=0; i<depth; i++ {
		fmt.Printf(" |")
	}
	fmt.Printf(" |---- ")
}

func (tnt *TnTServer) ParseTree(path string, depth int) {
	fst := tnt.Tree.MyTree

	if _, exists := fst[path]; exists {
		spaces(depth)
		fmt.Println(tnt.me, path, ":	", fst[path].LastModTime, "	", fst[path].VerVect, "	", fst[path].SyncVect, fst[path].Creator, fst[path].CreationTime)
		if fst[path].IsDir {
			for child, _ := range fst[path].Children {
				tnt.ParseTree(child, depth+1)
			}
		}
	}
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

func (tnt *TnTServer) GetDir(args *GetDirArgs, reply *GetDirReply) error {

  fi, err1 := os.Lstat(tnt.root + args.Path)

  if err1 != nil {
	  log.Println(tnt.me, " : Error opening file:", err1)
	  reply.Err = err1
  } else {
	  reply.Perm = fi.Mode().Perm()
	  reply.Err = nil
  }

  return nil
}

func (tnt *TnTServer) CopyFileFromPeer(srv int, path string, dest string, isDir bool) error {
	//Handle the directory case  
	if isDir {
		args := &GetDirArgs{Path:path}
		var reply GetDirReply
		ok := call(tnt.servers[srv], "TnTServer.GetDir", args, &reply)
		if ok {
			if reply.Err != nil {
				log.Println(tnt.me, ": Error opening Directory:", reply.Err)
			} else {
				err := os.Mkdir(tnt.root + dest, reply.Perm)
				if err != nil {
					log.Println(tnt.me, ": Error writing file:", err)
				}
			}
		} else {
			log.Println(tnt.me, ": GetDir RPC failed")
		}
		return reply.Err
	} else {
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
}

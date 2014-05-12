package TnT_v2_3

import (
	"net"
	"log"
	"os"
	"io/ioutil"
	"encoding/gob"
	"time"
	"sync"
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
	NewFiles []NewData
	DelFiles []DelData
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
	tmp string
	Test bool
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

func (tnt *TnTServer) CopyDirFromPeer(srv int, path string, dest string) (time.Time, error) {

	args := &GetDirArgs{Path:path}
	var reply GetDirReply

	rand_name := rand_string(2)
	var ts time.Time

	for {
		ok := call(tnt.servers[srv], "TnTServer.GetDir", args, &reply)
		if ok {
			break
		}
		time.Sleep(RPC_SLEEP_INTERVAL)
	}

	if reply.Err != nil {
		log.Println(tnt.me, ": Error opening Directory:", reply.Err)
	} else {
		err := os.Mkdir(tnt.tmp + rand_name, reply.Perm)
		if err != nil {
			log.Println(tnt.me, ": Error writing directory:", err)
		} else {
			tnt.Tree.NewFiles = append(tnt.Tree.NewFiles, NewData{TmpName:rand_name, Path:dest, IsDir:true, Perm:reply.Perm})
		}
		fi, err := os.Lstat(tnt.tmp + rand_name)
		ts = fi.ModTime()
	}

	return ts, reply.Err
}

func (tnt *TnTServer) CopyFileFromPeer(srv int, path string, dest string) (time.Time, error) {

	args := &GetFileArgs{FilePath:path}
	var reply GetFileReply

	rand_name := rand_string(2)
	var ts time.Time

	for {
		ok := call(tnt.servers[srv], "TnTServer.GetFile", args, &reply)
		if ok {
			break
		}
		time.Sleep(RPC_SLEEP_INTERVAL)
	}

	if reply.Err != nil {
		log.Println(tnt.me, ": Error opening file:", reply.Err)
	} else {
		err := ioutil.WriteFile(tnt.tmp + rand_name, reply.Content, reply.Perm)
		if err != nil {
			log.Println(tnt.me, ": Error writing file:", err)
		} else {
			tnt.Tree.NewFiles = append(tnt.Tree.NewFiles, NewData{TmpName:rand_name, Path: dest, IsDir:false, Perm: reply.Perm})
		}
		fi, err := os.Lstat(tnt.tmp + rand_name)
		ts = fi.ModTime()
	}

	return ts, reply.Err
}

func (tnt *TnTServer) DeleteDir(path string) {
	tnt.Tree.DelFiles = append(tnt.Tree.DelFiles, DelData{Path: path, IsDir: true})
}

func (tnt *TnTServer) DeleteFile(path string) {
	tnt.Tree.DelFiles = append(tnt.Tree.DelFiles, DelData{Path: path, IsDir: false})
}

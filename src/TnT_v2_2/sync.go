package TnT_v2_2

import (
	"fmt"
	"log"
	"os"
	"time"
	"strings"
)

func (tnt *TnTServer) Apply() {
	DelFiles := tnt.Tree.DelFiles
	NewFiles := tnt.Tree.NewFiles

	// Delete files
	for i:=0; i < len(DelFiles); i++ {
		delData := DelFiles[i]
		rname := rand_string(2)
		if delData.IsDir {
			os.Rename(tnt.root + delData.Path, tnt.tmp + rname)
			os.RemoveAll(tnt.tmp + rname)
		} else {
			os.Rename(tnt.root + delData.Path, tnt.tmp + rname)
			os.Remove(tnt.tmp + rname)
		}
	}
	tnt.Tree.DelFiles = DelFiles[0:0]

	// Update files
	for i:=0; i < len(NewFiles); i++ {
		newData := NewFiles[i]
		if newData.IsDir {
			err := os.Mkdir(tnt.root+newData.Path, newData.Perm)
			if err!=nil {
				log.Println(tnt.me, ": Error writing directory:",err)		
			}
		} else {
			err := os.Rename(tnt.tmp + newData.TmpName, tnt.root + newData.Path)
			if err!=nil {
				log.Println(tnt.me, ": Error writing file:",err)		
			}

		}
	}
	tnt.Tree.NewFiles = NewFiles[0:0]

	os.RemoveAll(tnt.tmp)
	os.Mkdir(tnt.tmp, 0777)
}

func (tnt *TnTServer) SyncWrapper(srv int, path string) {
	//Update tree and then sync
	tnt.mu.Lock()
	defer tnt.mu.Unlock()

	fst := tnt.Tree.MyTree

	if _, present := fst[path]; present == false {
		fmt.Println("Sync called on non-existent path:", path)
	} else {
		tnt.UpdateTreeWrapper(path)
		if tnt.Tree.MyTree[path].IsDir == true {
			tnt.SyncDir(srv, path)
		} else {
			tnt.SyncFile(srv, path)
		}
		tnt.LogToFile()
		tnt.ParseTree("./", 0)
		tnt.PrintTmp()
		var a int
		//fmt.Scanf("%d", &a)
		fmt.Println(a)
		tnt.Apply()
		fmt.Scanf("%d", &a)
		tnt.LogToFile()
	}
}

func (tnt *TnTServer) OnlySync(path string) {

	fst := tnt.Tree.MyTree

	parent := fst[path].Parent
	setMaxVersionVect(fst[path].SyncVect, fst[parent].SyncVect)
	for k, _ := range fst[path].Children {
		tnt.OnlySync(k)
	}
}

func (tnt *TnTServer) SyncDir(srv int, path string) (bool, map[int]int64, map[int]int64) {

	fst := tnt.Tree.MyTree // for ease of code

	args:=&GetVersionArgs{Path:path}
	var reply GetVersionReply
	for {
		ok:=call(tnt.servers[srv], "TnTServer.GetVersion", args, &reply)
		if ok {
			break
		}
		time.Sleep(RPC_SLEEP_INTERVAL)
	}

	_, exists := fst[path]
	action := DO_NOTHING
	//choice := -1 // since there is never a conflict on a directory now!
	if reply.Exists == false && exists == true {
		if reply.SyncVect[fst[path].Creator] < fst[path].CreationTime {
			action = SYNC_DOWN
		} else if mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect); mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
			action = DELETE
		} else {
			// Delete-Update conflict resolved as SYNC_DOWN
			action = SYNC_DOWN
		}
	} else if reply.Exists == true && exists == false {
		live_ancestor := tnt.LiveAncestor(path)
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[live_ancestor].SyncVect)
		//fmt.Println("UPDATE-DELETE:", path, live_ancestor, fst[live_ancestor].SyncVect, reply.Creator, reply.CreationTime, reply.VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = DO_NOTHING
		} else if fst[live_ancestor].SyncVect[reply.Creator] < reply.CreationTime {
			action = UPDATE
		} else {
			// Update-Delete conflict: resolved as UPDATE
			action = UPDATE
		}
	} else /* reply.Exists == true && exists == true */ {
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[path].SyncVect)
		//mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = SYNC_DOWN
		} else {
			action = UPDATE
		}
	}

	var verVect map[int]int64
	var syncVect map[int]int64

	if action == DO_NOTHING {
		fmt.Println("ACTION:", tnt.me, "has nothing to do for", path)
		live_ancestor := tnt.LiveAncestor(path) // should be the parent actually
		syncVect = MaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
		exists = false
		//setMaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
		//tnt.PropagateUp(fst[live_ancestor].VerVect, fst[live_ancestor].SyncVect, fst[live_ancestor].Parent)
	} else if action == DELETE {
		fmt.Println("ACTION:", tnt.me, "is deleting", path, "due to", srv)
		tnt.DeleteDir(path) //os.RemoveAll(tnt.root + path)
		syncVect = MaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		//tnt.PropagateUp(fst[path].VerVect,fst[path].SyncVect,fst[path].Parent)
		tnt.DeleteTree(path)
		exists = false
	} else if action == UPDATE {
		fmt.Println("ACTION:", tnt.me, "is updating", path, "from", srv)
		if exists == false {
			tnt.CopyDirFromPeer(srv, path, path)

			// Create new FSnode
			fst[path] = new(FSnode)
			fst[path].Name = strings.TrimPrefix(path, parent(path))
			//fst[path].Size = fi.Size()
			fst[path].IsDir = true
			//fst[path].LastModTime = fi.ModTime()
			fst[path].Children = make(map[string]bool)
			fst[path].VerVect = make(map[int]int64)
			fst[path].SyncVect = make(map[int]int64)
			for i:=0; i<len(tnt.servers); i++ {
				fst[path].VerVect[i] = 0
				fst[path].SyncVect[i] = 0
			}
			fst[path].Parent = parent(path)
			fst[parent(path)].Children[path] = true

			setVersionVect(fst[path].VerVect, reply.VerVect)
			//setMaxVersionVect(fst[path].SyncVect, reply.SyncVect) // done outside of 'if'
		}

		fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime

		newSyncVect := make(map[int]int64)
		for i:=0; i<len(tnt.servers); i++ {
			newSyncVect[i] = END_OF_WORLD
		}

		for k, _ := range fst[path].Children {
			var c_exists bool
			var c_verVect map[int]int64
			var c_syncVect map[int]int64
			if fst[k].IsDir == true {
				c_exists, c_verVect, c_syncVect = tnt.SyncDir(srv,k)
			} else {
				c_exists, c_verVect, c_syncVect = tnt.SyncFile(srv,k)
			}
			if c_exists == true {
				setMaxVersionVect(fst[path].VerVect, c_verVect)
				setMinVersionVect(newSyncVect, c_syncVect)
			} else {
				setMinVersionVect(newSyncVect, c_syncVect)
			}
		}
		for k, _ := range reply.Children {
			_, present := fst[path].Children[k]
			if present == false {
				var c_exists bool
				var c_verVect map[int]int64
				var c_syncVect map[int]int64
				if reply.IsDir[k] == true {
					c_exists, c_verVect, c_syncVect = tnt.SyncDir(srv,k)
				} else {
					c_exists, c_verVect, c_syncVect = tnt.SyncFile(srv,k)
				}
				if c_exists == true {
					setMaxVersionVect(fst[path].VerVect, c_verVect)
					setMinVersionVect(newSyncVect, c_syncVect)
				} else {
					setMinVersionVect(newSyncVect, c_syncVect)
				}
			}
		}
		fst[path].Creator, fst[path].CreationTime = reply.Creator, reply.CreationTime
		//setVersionVect(fst[path].VerVect, reply.VerVect)
		setVersionVect(fst[path].SyncVect, newSyncVect)
		setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
		exists = true
	} else /* action == SYNC_DOWN */ {
		fmt.Println("ACTION:", tnt.me, "is only syncing down for", path)
		setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		tnt.OnlySync(path)
		verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
		exists = true
	}
	return exists, verVect, syncVect
}

func (tnt *TnTServer) SyncFile(srv int, path string) (bool, map[int]int64, map[int]int64) {
	/*
	(1) Check for updates on local version of file: update VerVect, SyncVect if required
	(2) Get VerVect and SyncVect from 'srv'
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

	fst := tnt.Tree.MyTree

	//Sync Recursively
	args:=&GetVersionArgs{Path:path}
	var reply GetVersionReply
	for {
		ok:=call(tnt.servers[srv], "TnTServer.GetVersion", args, &reply)
		if ok {
			break
		}
		time.Sleep(RPC_SLEEP_INTERVAL)
	}

	// A : srv ; B : tnt.me
	action := DO_NOTHING
	choice := -1
	_, exists := fst[path]
	if reply.Exists == false && exists == false {
		action = DO_NOTHING // will never happen actually! :)
	} else if reply.Exists == false && exists == true {
		if reply.SyncVect[fst[path].Creator] < fst[path].CreationTime {
			action = DO_NOTHING
		} else if mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect); mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
			action = DELETE
		} else {
			// Delete-Update conflict: resolved as DO_NOTHING
			action = DO_NOTHING
			/*
			fmt.Println("Delete-Update conflict on", path, ":", srv, "has deleted, but", tnt.me, "has updated")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = DO_NOTHING
			} else {
				action = DELETE
			}
			*/
		}
	} else if reply.Exists == true && exists == false {
		live_ancestor := tnt.LiveAncestor(path)
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[live_ancestor].SyncVect)
		fmt.Println("UPDATE-DELETE:", path, live_ancestor, fst[live_ancestor].SyncVect, reply.Creator, reply.CreationTime, reply.VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = DO_NOTHING
		} else if fst[live_ancestor].SyncVect[reply.Creator] < reply.CreationTime {
			action = UPDATE
		} else {
			// Update-Delete conflict: resolved as UPDATE
			action = UPDATE
			/*
			fmt.Println("Update-Delete conflict on", path, ":", srv, "has updated, but", tnt.me, "has deleted")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = DO_NOTHING
			} else {
				action = UPDATE
			}
			*/
		}
	} else /* reply.Exists == true && exists == true */ {
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[path].SyncVect)
		mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = DO_NOTHING
		} else if  mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
			action = UPDATE
		} else {
			// Update-Update conflict
			fmt.Println("Update-Update conflict on", path, ":", srv, "and", tnt.me, "have independently updated")
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

	var verVect map[int]int64
	var syncVect map[int]int64

	if action == DO_NOTHING {
		fmt.Println("ACTION:", tnt.me, "has nothing to do for", path)
		if exists == true {
			setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
			verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
			//tnt.PropagateUp(fst[path].VerVect, fst[path].SyncVect, fst[path].Parent)
		} else {
			live_ancestor := tnt.LiveAncestor(path) // should be the parent actually
			syncVect = MaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
			//setMaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
			//tnt.PropagateUp(fst[live_ancestor].VerVect, fst[live_ancestor].SyncVect, fst[live_ancestor].Parent)
		}
	} else if action == DELETE {
		fmt.Println("ACTION:", tnt.me, "is deleting", path, "due to", srv)
		tnt.DeleteFile(path) //os.Remove(tnt.root + path)
		syncVect = MaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		//tnt.PropagateUp(fst[path].VerVect,fst[path].SyncVect,fst[path].Parent)
		tnt.DeleteTree(path)
		exists = false
	} else if action == UPDATE {
		fmt.Println("ACTION:", tnt.me, "is getting", path, "from", srv)
		// get file
		ts, _ := tnt.CopyFileFromPeer(srv, path, path)

		if exists == false {
			// Create new FSnode
			fst[path] = new(FSnode)
			fst[path].Name = strings.TrimPrefix(path, parent(path))
			//fst[path].Size = fi.Size()
			fst[path].IsDir = false
			fst[path].VerVect = make(map[int]int64)
			fst[path].SyncVect = make(map[int]int64)
			for i:=0; i<len(tnt.servers); i++ {
				fst[path].VerVect[i] = 0
				fst[path].SyncVect[i] = 0
			}
			fst[path].Parent = parent(path)
			fst[parent(path)].Children[path] = true
		}
		fst[path].LastModTime = ts
		fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime
		fst[path].Creator, fst[path].CreationTime = reply.Creator, reply.CreationTime
		setVersionVect(fst[path].VerVect, reply.VerVect)
		setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)

		verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
		exists = true
		//tnt.PropagateUp(fst[path].VerVect,fst[path].SyncVect,fst[path].Parent)
	}

	return exists, verVect, syncVect
}

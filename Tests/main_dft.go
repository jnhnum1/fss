package main

import{
	"fmt"
	"os"
	"path/filepath"
	"ioutil"
}


func DFT(dirname string, depth int ,str string)string {
    d, err := os.Open(dirname)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    defer d.Close()
    fi, err := d.Readdir(-1)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    for _, fi := range fi {
        if fi.Mode().IsRegular() {
           str=str+fi.Name()
        }
        if fi.IsDir() {
  			str=str+fi.Name()
            str=DFT(dirname+fi.Name()+string(filepath.Separator), depth+1,str)
        }else{
		data,_:=ioutil.ReadFile(dirname+fi.Name())
        	str=str+string(data)
        }
    }
    return str
}

func main(){
	fmt.Println(DFT("../roots/nest0",0,""))
	fmt.Println(DFT("../roots/nest1",0,""))
}

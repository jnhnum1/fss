package main

import "encoding/gob"
import "os"
import "log"

type FileContents struct {
  Filename string
  Contents string
}

func main() {
  // copy bytes from "from" file to "to" file via encoding of FileContents struct

  // from := os.Open("from")
  // to := os.OpenFile("to", os.O_WRONLY | os.O_CREATE, 0777)

  f, err := os.OpenFile("tempFile", os.O_WRONLY | os.O_CREATE, 0777)
  if err != nil {
    log.Println("Error opening file:", err)
  }

  encoder := gob.NewEncoder(f)
  encoder.Encode(FileContents{"foo", "foo's contents"})
  f.Close()

  f, err = os.Open("tempFile")
  var contents FileContents
  decoder := gob.NewDecoder(f)
  decoder.Decode(&contents)

  log.Println("received", contents)
}


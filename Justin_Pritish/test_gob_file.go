package main

//import "encoding/gob"
//import "os"
import "log"
import "io/ioutil"


func main() {

  d, err := ioutil.ReadFile("./watch_folder/a_meeting_by_the_river.mp3")
  
  if err != nil {
    log.Println("Error opening file:", err)
  }

  err = ioutil.WriteFile("./watch_folder/ambtr.mp3", d, 0644)
  if err != nil {
    log.Println("Error opening file:", err)
  }

  log.Println("copy dumped!")
}


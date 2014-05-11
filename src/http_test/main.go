package main

import "flag"
import "fmt"
import "log"
import "net"
import "net/http"
import "net/rpc"
import "time"

type Server struct {
}

type PingArgs struct {
  N int
}

type PingReply struct {
  N int
}


func (srv *Server) Ping(args *PingArgs, reply *PingReply) error {
  fmt.Println("received ping: ", args.N)
  *reply = PingReply{args.N}
  return nil
}

func main() {

  // this is just to parse command line options: --serve to run the RPC handler,
  // and --address A.B.C.D:portnumber to connect to somebody else.
  var server bool
  var address string
  flag.BoolVar(&server, "serve", false, "set this flag to run the RPC server")
  flag.StringVar(&address, "address", "", "specify the address to connect to")
  flag.Parse()
  if server {
    rpc.Register(&Server{})
    rpc.HandleHTTP()
    l, e := net.Listen("tcp", ":1235")
    if e != nil {
      log.Fatal("listen error:", e)
    } else {
      defer l.Close()
      fmt.Println("listening for requests...")
      http.Serve(l, nil)
    }
  }
  if address != "" {
    client, err := rpc.DialHTTP("tcp", address + ":1235")
    if err != nil {
      log.Fatal("dialing:", err)
    }
    for i := 0; i < 10; i++ {
      go func(i int) {
        var reply PingReply
        err = client.Call("Server.Ping", PingArgs{i}, &reply)
        if err != nil {
          log.Fatal("ping error:", err)
        }
        fmt.Println("got ping reply: ", reply.N)
      }(i)
      time.Sleep(time.Second)
    }
  }
}

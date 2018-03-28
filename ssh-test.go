package main

// 	"./ssh-client.go"
// import . "./ssh"
import (
	"log"

	"github.com/howels/infra-tools/ssh"
)

func main() {
	s := sshclient.NewSSHClient("sshtest", "sshtest", "localhost")
	out, err := s.Command("ls /")
	if err != nil {
		panic(err)
	}
	log.Print(string(out.Stdout))
	out2, err2 := s.Command("woami")
	if err2 != nil {
		log.Printf("Error: %+v %+v \n", err2, out2)
	}
	log.Print(string(out2.Stdout))
	s.Quit()
}

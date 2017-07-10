package sshclient

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

var client *ssh.Client
var session *ssh.Session
var user, host, pass string = "", "", ""

// SSHClient forms the external type
type SSHClient struct {
	user    string
	pass    string
	host    string
	client  *ssh.Client
	session *ssh.Session
}

// NewSSHClient simple constructor
func NewSSHClient(username string, password string, hostname string) *SSHClient {
	//simple constructor
	s := &SSHClient{
		user: username,
		pass: password,
		host: hostname,
	}
	return s

}

func main() {
	if len(os.Args) != 5 {
		log.Fatalf("Usage: %s <user> <password> <host:port> <command>", os.Args[0])
	}

	user = os.Args[1]
	pass = os.Args[2]
	host = os.Args[3]

	s := NewSSHClient(user, pass, host)

	out, err := s.Execute(os.Args[4])
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	client.Close()
}

// Execute runs an SSH command and returns the string response
func (s *SSHClient) Execute(cmd string) ([]byte, error) {
	//client, session, err := connectToHost(os.Args[1], os.Args[2])
	_, session, err := s.connectSSHHost(s.user, s.host, s.pass)
	if err != nil {
		panic(err)
	}
	result, err := session.CombinedOutput(cmd)
	if err != nil {
		// invalidate the sessions?
		// s.session = nil
		// s.client = nil
		panic(err)
	}
	return result, nil

}

// Quit closes the connection
func (s *SSHClient) Quit() {
	s.client.Close()
}

func (s *SSHClient) getSession() (*ssh.Client, *ssh.Session, error) {
	if s.session != nil && s.client != nil {
		return s.client, s.session, nil
	}
	client, session, err := s.connectSSHHost(user, host, pass)
	if err != nil {
		return nil, nil, err
	}
	return client, session, nil

}

func (s *SSHClient) connectSSHHost(user, host, pass string) (*ssh.Client, *ssh.Session, error) {

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	newclient, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		return nil, nil, err
	}
	s.client = newclient

	newsession, err := s.client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	s.session = newsession

	return s.client, s.session, nil
}

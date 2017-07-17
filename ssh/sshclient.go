package sshclient

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

var client *ssh.Client
var session *ssh.Session
var user, host, pass string = "", "", ""

//ShellConnection represents local, remote or fake unix shell interactions
type ShellConnection interface {
	Execute(*Command) (*Command, error)
	Command(string) (*CommandOutput, error)
}

// SSHClient forms the external type
type SSHClient struct {
	user    string
	pass    string
	host    string
	client  *ssh.Client
	session *ssh.Session
}

//Command holds the input and output data for a command
type Command struct {
	Command string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

//CommandOutput is the buffer contents of a completed command
type CommandOutput struct {
	Stdin  string
	Stdout string
	Stderr string
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

	out, err := s.Command(os.Args[4])
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out.Stdout))
	client.Close()
}

//Command takes string and inputs to stdin
func (s *SSHClient) Command(cmdString string) (*CommandOutput, error) {
	var stdoutb, stderrb bytes.Buffer
	stdoutWriter := bufio.NewWriter(&stdoutb)
	stderrWriter := bufio.NewWriter(&stderrb)
	cmd := &Command{Command: cmdString, Stderr: stderrWriter, Stdout: stdoutWriter}
	cmd, err := s.Execute(cmd)
	//log.Printf("SSH stdout: %v", stdoutb.String())
	result := &CommandOutput{
		Stdout: stdoutb.String(),
		Stderr: stderrb.String()}
	return result, err

}

// Execute runs an SSH command and returns a struct of stdin, stdout and stderr
func (s *SSHClient) Execute(cmd *Command) (*Command, error) {
	//client, session, err := connectToHost(os.Args[1], os.Args[2])
	_, session, err := s.connectSSHHost(s.user, s.host, s.pass)
	if err != nil {
		panic(err)
	}
	// result, err := session.CombinedOutput(cmd)
	// if err != nil {
	// 	// invalidate the sessions?
	// 	// s.session = nil
	// 	// s.client = nil
	// 	panic(err)
	// }
	err = s.prepareCommand(session, cmd)
	if err != nil {
		return cmd, err
	}
	err = session.Run(cmd.Command)
	return cmd, err
}

func (s *SSHClient) prepareCommand(session *ssh.Session, cmd *Command) error {
	for _, env := range cmd.Env {
		variable := strings.Split(env, "=")
		if len(variable) != 2 {
			continue
		}

		if err := session.Setenv(variable[0], variable[1]); err != nil {
			return err
		}
	}

	if cmd.Stdin != nil {
		stdin, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stdin for session: %v", err)
		}
		go io.Copy(stdin, cmd.Stdin)
	}

	if cmd.Stdout != nil {
		stdout, err := session.StdoutPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stdout for session: %v", err)
		}
		go io.Copy(cmd.Stdout, stdout)
	}

	if cmd.Stderr != nil {
		stderr, err := session.StderrPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stderr for session: %v", err)
		}
		go io.Copy(cmd.Stderr, stderr)
	}

	return nil
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

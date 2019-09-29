package esxi

import (
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/tmc/scp"
)

// connectToHost connect to ESXi host using SSH
func connectToHost(esxiSSHinfo SSHConnectionSettings) (*ssh.Client, *ssh.Session, error) {

	sshConfig := &ssh.ClientConfig{
		User: esxiSSHinfo.user,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				// Reply password to all questions
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = esxiSSHinfo.pass
				}

				return answers, nil
			}),
		},
	}

	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	esxiHostAndPort := fmt.Sprintf("%s:%s", esxiSSHinfo.host, esxiSSHinfo.port)

	attempt := 10
	for attempt > 0 {
		client, err := ssh.Dial("tcp", esxiHostAndPort, sshConfig)
		if err != nil {
			log.Printf("[runRemoteSshCommand] Retry connection: %d\n", attempt)
			attempt--
			time.Sleep(1 * time.Second)
		} else {

			session, err := client.NewSession()
			if err != nil {
				client.Close()
				return nil, nil, fmt.Errorf("Session Connection Error")
			}

			return client, session, nil

		}
	}
	return nil, nil, fmt.Errorf("Client Connection Error")
}

// runCommandOnHost runs a command on the remote host
func runCommandOnHost(esxiSSHinfo SSHConnectionSettings, remoteSSHCommand string, shortCmdDesc string) (string, error) {
	log.Println("[runRemoteSshCommand] :" + shortCmdDesc)

	client, session, err := connectToHost(esxiSSHinfo)
	if err != nil {
		log.Println("[runRemoteSshCommand] Failed err: " + err.Error())
		return "Failed to ssh to esxi host", err
	}

	stdoutRaw, err := session.CombinedOutput(remoteSSHCommand)
	stdout := strings.TrimSpace(string(stdoutRaw))
	log.Printf("[runRemoteSshCommand] cmd:/%s/\n stdout:/%s/\nstderr:/%s/\n", remoteSSHCommand, stdout, err)

	client.Close()
	return stdout, err
}

// copyFileToHost copies a file to the host via SCP
func copyFileToHost(esxiSSHinfo SSHConnectionSettings, localfileName string, remoteFileName string) error {
	log.Println("[copyFileViaScp] :" + localfileName)

	_, session, err := connectToHost(esxiSSHinfo)

	if err != nil {
		log.Println("[copyFileViaScp] Failed to connect to ESXi host via SSH: " + err.Error())

		return err
	}

	err = scp.CopyPath(localfileName, remoteFileName, session)

	if err != nil {
		log.Println("[copyFileViaScp] Failed to copy file to ESXi host: " + err.Error())

		return err
	}

	return nil
}

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"github.com/pkg/sftp"
	//"github.com/bitrise-io/go-utils/cmdex"
)

type ConfigsModel struct {
	hostname         string
	username         string
	password         string
	port             string
	uploadSourcePath string
	uploadTargetPath string
}

func createConfigsModelFromEnvs() ConfigsModel {

	ret := ConfigsModel{
		hostname:         os.Getenv("hostname"),
		username:         os.Getenv("username"),
		password:         os.Getenv("password"),
		uploadSourcePath: os.Getenv("upload_source_path"),
		uploadTargetPath: os.Getenv("upload_target_path"),
		port:             os.Getenv("port"),
	}
	return ret
}

func (configs ConfigsModel) print() {
	fmt.Println()
	log.Printf("Configs:")
	log.Printf(" - Hostname: %s \n", configs.hostname)
	log.Printf(" - Port: %s \n", configs.port)
	log.Printf(" - Username: *** \n")
	log.Printf(" - Password: *** \n")
	log.Printf(" - UploadSourcePath: %s \n", configs.uploadSourcePath)
	log.Printf(" - UploadTargetPath: %s \n", configs.uploadTargetPath)
}

func (configs ConfigsModel) validate() error {
	// required
	if configs.hostname == "" {
		return errors.New("No Hostname parameter specified!")
	}
	if configs.username == "" {
		return errors.New("No Username parameter specified!")
	}

	if configs.uploadSourcePath == "" {
		return errors.New("No Upload source path specified")
	}

	if configs.uploadTargetPath == "" {
		return errors.New("No Upload target path specified")
	}

	if configs.port == "" {
		return errors.New("No port specified!")
	}

	return nil
}

/*func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := cmdex.NewCommand("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}*/

func connectWithSSH(configs ConfigsModel) {
	// Connect to a remote host and request the sftp subsystem via the 'ssh'
	// command.  This assumes that passwordless login is correctly configured.
	log.Println("connecting ssh...")
	log.Println(configs.username + "@" + configs.hostname)
	cmd := exec.Command("ssh", configs.username+"@"+configs.hostname, "-s", "sftp", "-p", configs.port)

	// send errors from ssh to stderr
	cmd.Stderr = os.Stderr

	// get stdin and stdout
	wr, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	rd, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// start the process
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	//log.Println("printing password")
	//wr.Write([]byte(configs.password))
	defer cmd.Wait()

	log.Println("open sftp channel")
	// open the SFTP session
	client, err := sftp.NewClientPipe(rd, wr)
	if err != nil {
		log.Fatal(err)
	}

	// walk a directory
	log.Println("Walking")
	w := client.Walk(configs.uploadTargetPath)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		log.Println(w.Path())
	}
	log.Println("Walking finished")

	filename := filepath.Base(configs.uploadSourcePath)

	dat, err := ioutil.ReadFile(configs.uploadSourcePath)

	if err != nil {
		log.Fatalf("Error in opening source file: %s", err.Error())
	}

	// check if its already there
	fi, err := client.Lstat(filename)
	if err == nil {
		//log.Fatal(err)
		//sftp.Remove(filename)
		log.Printf("file already there !")
		log.Println(fi)
	}

	// leave your mark
	f, err := client.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write(dat); err != nil {
		log.Fatal(err)
	}

	// check it's there
	fi2, err := client.Lstat(filename)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(fi2)

	// close the connection
	client.Close()
}

func main() {
	configs := createConfigsModelFromEnvs()
	configs.print()
	if err := configs.validate(); err != nil {
		log.Fatalf("Issue with input: %s", err)
	}
	//connectWithSSH(configs)

	sshConfig := &ssh.ClientConfig{
		User: configs.username,
		Auth: []ssh.AuthMethod{
			ssh.Password(configs.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", configs.hostname+":"+configs.port, sshConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}

	defer connection.Close()

	// open an SFTP session over an existing ssh connection.
	sftp, err := sftp.NewClient(connection)
	if err != nil {
		log.Fatal(err)
	}
	defer sftp.Close()

	wd, err := sftp.Getwd()

	log.Println("wd: " + wd)

	filename := filepath.Base(configs.uploadSourcePath)

	dat, err := ioutil.ReadFile(configs.uploadSourcePath)

	if err != nil {
		log.Fatalf("Error reading file: %s\n", err.Error())
	}

	log.Printf("target file: %s\n", configs.uploadTargetPath+filename)

	// leave your mark
	f, err := sftp.Create(configs.uploadTargetPath + filename)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write(dat); err != nil {
		log.Fatal(err)
	}

}

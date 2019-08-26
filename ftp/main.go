package ftp

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var CONNECT_TIMEOUT = time.Duration(5) * time.Second

type Response struct {
	resp *ftp.Response
	conn *ftp.ServerConn
}

func (r *Response) Read(p []byte) (n int, err error) {
	return r.resp.Read(p)
}

func (r *Response) Close() error {
	if err := r.resp.Close(); err != nil {
		return err
	}

	if err := r.conn.Quit(); err != nil {
		return err
	}

	return nil
}

func Open(addr, user, password, path string) (io.ReadCloser, int64, error) {
	conn, err := ftp.DialTimeout(addr, CONNECT_TIMEOUT)
	if err != nil {
		return nil, 0, err
	}

	conn.DisableEPSV = true

	err = conn.Login(user, password)
	if err != nil {
		return nil, 0, err
	}

	size, err := conn.FileSize(path)
	if err != nil {
		return nil, 0, err
	}

	resp, err := conn.Retr(path)
	if err != nil {
		return nil, 0, err
	}

	return &Response{resp: resp, conn: conn}, size, nil

}

func SFTPOpen(addr, user, password, path string) (io.ReadCloser, int64, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout: CONNECT_TIMEOUT,
	}

	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, 0, err
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		return nil, 0, err
	}

	file, err := sftpClient.Open(path)
	if err != nil {
		return nil, 0, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}

	MB := 1 << (2 * 10) //1 Mebibyte
	bufReader := bufio.NewReaderSize(file, MB*10)
	bufReadCloser := ioutil.NopCloser(bufReader)

	return bufReadCloser, fileInfo.Size(), nil
}

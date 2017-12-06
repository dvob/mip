package ftp

import (
	"github.com/jlaffaye/ftp"
	"time"
)

var CONNECT_TIMEOUT = time.Duration(5) * time.Second

type Response struct {
	resp *ftp.Response
	conn *ftp.ServerConn
	Size int64
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

func Open(addr, user, password, path string) (*Response, error) {
	conn, err := ftp.DialTimeout(addr, CONNECT_TIMEOUT)
	if err != nil {
		return nil, err
	}

	conn.DisableEPSV = true

	err = conn.Login(user, password)
	if err != nil {
		return nil, err
	}

	size, err := conn.FileSize(path)
	if err != nil {
		return nil, err
	}

	resp, err := conn.Retr(path)
	if err != nil {
		return nil, err
	}

	return &Response{resp: resp, conn: conn, Size: size}, nil

}

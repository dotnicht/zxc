package deployer

import (
	"context"
	"fmt"
	"io"
	"net"
	"path"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

type Request struct {
	Host     string
	User     string
	Key      []byte
	Payload  io.Reader
	StopCmd  string
	StartCmd string
}

func Deploy(ctx context.Context, req Request) error {
	signer, err := gossh.ParsePrivateKey(req.Key)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	cfg := &gossh.ClientConfig{
		User:            req.User,
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	addr := req.Host + ":22"
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := gossh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return err
	}

	client := gossh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	if req.StopCmd != "" {
		if err := run(ctx, client, req.StopCmd); err != nil {
			return fmt.Errorf("stop: %w", err)
		}
	}

	remotePath := path.Join("/tmp", uuid.New().String()+".zip")

	if err := upload(ctx, client, req.Payload, remotePath); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer run(ctx, client, "rm -f "+remotePath)

	if err := run(ctx, client, fmt.Sprintf("unzip -o %s -d ~", remotePath)); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	if req.StartCmd != "" {
		if err := run(ctx, client, req.StartCmd); err != nil {
			return fmt.Errorf("start: %w", err)
		}
	}

	return nil
}

func Ping(ctx context.Context, host, user string, key []byte) error {
	signer, err := gossh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	cfg := &gossh.ClientConfig{
		User:            user,
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	addr := host + ":22"
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := gossh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return err
	}

	gossh.NewClient(sshConn, chans, reqs).Close()
	return nil
}

func run(ctx context.Context, client *gossh.Client, cmd string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("run %q: %w\noutput: %s", cmd, err, out)
	}
	return nil
}

func upload(ctx context.Context, client *gossh.Client, r io.Reader, remotePath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sc, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer sc.Close()

	f, err := sc.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create %s: %w", remotePath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

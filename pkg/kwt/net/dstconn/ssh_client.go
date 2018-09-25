package dstconn

import (
	"fmt"
	"net"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

type SSHClient struct {
	connOpts   SSHClientConnOpts
	client     *gossh.Client
	shutdownCh chan struct{}

	logTag string
	logger Logger
}

var _ Factory = &SSHClient{}

type SSHClientConnOpts struct {
	User             string
	Host             string
	PrivateKeyPEM    string
	HostPublicKeyAuf string // in authorized_hosts format
}

func NewSSHClient(connOpts SSHClientConnOpts, logger Logger) *SSHClient {
	return &SSHClient{
		connOpts:   connOpts,
		shutdownCh: make(chan struct{}),

		logTag: "SSHClient",
		logger: logger,
	}
}

func (c *SSHClient) Connect() error {
	signer, err := gossh.ParsePrivateKey([]byte(c.connOpts.PrivateKeyPEM))
	if err != nil {
		return fmt.Errorf("Parsing private key: %s", err)
	}

	hostPublicKey, err := ParsePublicKey(c.connOpts.HostPublicKeyAuf)
	if err != nil {
		return err
	}

	sshConfig := &gossh.ClientConfig{
		User:              c.connOpts.User,
		Auth:              []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback:   gossh.FixedHostKey(hostPublicKey),
		HostKeyAlgorithms: []string{gossh.KeyAlgoRSA},
	}

	client, err := gossh.Dial("tcp", c.connOpts.Host, sshConfig)
	if err != nil {
		return fmt.Errorf("Connecting to SSH server: %s", err)
	}

	c.client = client

	go c.keepAlive()

	return nil
}

func (c *SSHClient) Disconnect() error {
	close(c.shutdownCh)

	err := c.client.Close()
	if err != nil {
		c.logger.Debug(c.logTag, "Shutting down client: %s", err)
	}

	return nil
}

func (c *SSHClient) NewConn(ip net.IP, port int) (net.Conn, error) {
	addr := &net.TCPAddr{IP: ip, Port: port, Zone: ""}
	return c.client.DialTCP("tcp", nil, addr)
}

func (c *SSHClient) NewConnCopier(proxyDesc string) ConnCopier {
	return NewSSHConnCopier(proxyDesc, c.logger)
}

func (c *SSHClient) keepAlive() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b, data, err := c.client.SendRequest("keepalive@openssh.com", true, nil)
			c.logger.Debug(c.logTag, "Sending keepalive: %t %v %s", b, data, err)

		case <-c.shutdownCh:
			return
		}
	}
}
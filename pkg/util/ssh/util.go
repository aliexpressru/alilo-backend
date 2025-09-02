package ssh

import (
	"context"
	"fmt"
	"strings"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// var logger = zap.S()

// keyPass Пока пустой, не используем пароль для расшифровки ключа, может пригодиться если начнем
var keyPass string

func SendCommandForAll(ctx context.Context, commandType string) {
	hosts := []string{ // TODO подумать куда вынести список машин
	}
	for _, host := range hosts {
		go SendCommand(ctx, commandType, host)
	}
}

func readPubKey(ctx context.Context) ssh.AuthMethod {
	var key ssh.Signer

	var err error

	var b []byte

	if config.Get(ctx).SSHKey == "" {
		logger.Error(ctx, "EMPTY SSH KEY PROVIDED")

		return nil
	}

	b = []byte(config.Get(ctx).SSHKey)
	if !strings.Contains(string(b), "ENCRYPTED") {
		key, err = ssh.ParsePrivateKey(b)
		if err != nil {
			err = errors.Wrap(err, "Error with parsing key")
			logger.Errorf(ctx, "SSH key without password error: '%v'", err)

			return nil
		}
	} else {
		key, err = ssh.ParsePrivateKeyWithPassphrase(b, []byte(keyPass))
		if err != nil {
			err = errors.Wrap(err, "Error with parsing key")
			logger.Errorf(ctx, "SSH key with password error: '%v'", err)

			return nil
		}
	}

	return ssh.PublicKeys(key)
}

func SendCommand(ctx context.Context, commandType string, host string) {
	logger.Infof(ctx, "Get command:%v; for host:%v;", commandType, host)

	pubKey := readPubKey(ctx)
	if pubKey != nil {
		conf := &ssh.ClientConfig{
			User: "stopper",
			Auth: []ssh.AuthMethod{
				pubKey,
			},
			//nolint
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), //TODO переделать на статичный список ключей
		}

		client, err := ssh.Dial("tcp", fmt.Sprint(host, ":22"), conf)
		if err != nil {
			err = errors.Wrap(err, "SSH dial error")
			logger.Errorf(ctx, "Error while dial host: '%v' Error: '%v'", host, err)

			return
		}

		session, err := client.NewSession()
		if err != nil {
			err = errors.Wrap(err, "SSH dial error")
			logger.Errorf(ctx, "failed to create ssh session on host: '%v' Error: '%v'", host, err)

			return
		}

		defer func(session *ssh.Session) {
			errOnClose := session.Close()
			if errOnClose != nil {
				logger.Errorf(ctx, "Error while closing session: '%v'", errOnClose)
			}
		}(session)
		logger.Infof(ctx, "Get command type: '%v'", commandType)

		cmd := "sudo ufw default allow outgoing" // RESUME - NOT USED ANYMORE
		if commandType == "stop" {
			cmd = "sudo ufw default deny outgoing" // PAUSE - NOT USED ANYMORE
		}

		if commandType == "kill" {
			cmd = "sudo killall -s SIGKILL k6 | sudo killall -s SIGKILL k6yaml | sudo killall -s SIGKILL ghz" // STOP
		}

		err = session.Run(cmd)
		if err != nil {
			err = errors.Wrap(err, "SSH dial error")
			logger.Errorf(ctx, "failed to run command over SSH: '%v' Error: '%v'", host, err)
		}

		logger.Infof(ctx, "Sending to host: '%v' Command : '%v' ", host, cmd)
		logger.Infof(ctx, "commandType: '%v'", commandType)
	}
}

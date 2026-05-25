//go:build !windows

package daemon

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

var SocketPath = "/var/run/grubstation.sock"

// Run starts the daemon and waits for termination signals.
func (d *Daemon) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info().Interface("signal", sig).Msg("Received signal, stopping daemon")
		cancel()
	}()

	return d.run(ctx)
}

func (d *Daemon) listenUnixSocket(ctx context.Context, token string) {
	path := SocketPath
	_ = os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create unix socket")
		return
	}
	defer func() { _ = l.Close() }()
	defer func() { _ = os.Remove(path) }()

	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Debug().Err(err).Msg("Unix socket accept error")
				continue
			}
		}
		go d.handleUnixConnection(ctx, conn, token)
	}
}

func (d *Daemon) handleUnixConnection(ctx context.Context, conn net.Conn, token string) {
	defer func() { _ = conn.Close() }()
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		if cmd := scanner.Text(); cmd == "push" {
			log.Info().Msg("Push requested via local Unix socket")
			if err := d.TriggerUpdate(ctx); err != nil {
				log.Error().Err(err).Msg("Socket requested push failed")
				_, _ = fmt.Fprintf(conn, "ERROR: %v\n", err)
			} else {
				_, _ = fmt.Fprintf(conn, "OK\n")
			}
		}
	}
}

func RequestPushViaSocket(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", SocketPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if _, err = conn.Write([]byte("push\n")); err != nil {
		return err
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		if resp := scanner.Text(); resp == "OK" {
			return nil
		} else {
			return fmt.Errorf("daemon returned error: %s", resp)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("no response from daemon")
}

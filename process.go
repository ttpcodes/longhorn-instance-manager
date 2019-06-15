package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/longhorn/longhorn-engine-launcher/rpc"
)

func StartProcessLauncherCmd() cli.Command {
	return cli.Command{
		Name: "start-launcher",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "listen",
				Value: "localhost:8500",
			},
		},
		Action: func(c *cli.Context) {
			if err := startLauncher(c); err != nil {
				logrus.Fatalf("Error running start command: %v.", err)
			}
		},
	}
}

func startLauncher(c *cli.Context) error {
	listen := c.String("listen")

	l, err := NewProcessLauncher(listen)
	if err != nil {
		return err
	}

	if err := l.StartRPCServer(); err != nil {
		return err
	}
	logrus.Infof("Launcher listening to %v", listen)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logrus.Infof("Receive %v to exit", sig)
		l.Shutdown()
	}()

	return l.WaitForShutdown()
}

func ProcessCmd() cli.Command {
	return cli.Command{
		Name: "process",
		Subcommands: []cli.Command{
			ProcessCreateCmd(),
			ProcessDeleteCmd(),
			ProcessGetCmd(),
			ProcessListCmd(),
		},
	}
}

func ProcessCreateCmd() cli.Command {
	return cli.Command{
		Name: "start",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "name",
			},
			cli.StringFlag{
				Name: "binary",
			},
			cli.IntSliceFlag{
				Name: "reserved-ports",
			},
		},
		Action: func(c *cli.Context) {
			if err := startProcess(c); err != nil {
				logrus.Fatalf("Error running engine start command: %v.", err)
			}
		},
	}
}

func startProcess(c *cli.Context) error {
	if c.String("name") == "" || c.String("binary") == "" {
		return fmt.Errorf("missing required parameter")
	}

	url := c.GlobalString("url")
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("cannot connect to %v: %v", url, err)
	}
	defer conn.Close()

	client := rpc.NewLonghornProcessLauncherServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), FrontendTimeout)
	defer cancel()

	//NOTE: is there a better way?
	ports := []int32{}
	cPorts := c.IntSlice("reserved-ports")
	for i := 0; i < len(cPorts); i++ {
		ports[i] = int32(cPorts[i])
	}

	obj, err := client.ProcessCreate(ctx, &rpc.ProcessCreateRequest{
		Spec: &rpc.ProcessSpec{
			Name:          c.String("name"),
			Binary:        c.String("binary"),
			Args:          c.Args(),
			ReservedPorts: ports,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start engine: %v", err)
	}
	return printJSON(obj)
}

func ProcessDeleteCmd() cli.Command {
	return cli.Command{
		Name: "stop",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "name",
			},
		},
		Action: func(c *cli.Context) {
			if err := stopProcess(c); err != nil {
				logrus.Fatalf("Error running engine stop command: %v.", err)
			}
		},
	}
}

func stopProcess(c *cli.Context) error {
	if c.String("name") == "" {
		return fmt.Errorf("missing required parameter")
	}

	url := c.GlobalString("url")
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("cannot connect to %v: %v", url, err)
	}
	defer conn.Close()

	client := rpc.NewLonghornProcessLauncherServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), FrontendTimeout)
	defer cancel()

	obj, err := client.ProcessDelete(ctx, &rpc.ProcessDeleteRequest{
		Name: c.String("name"),
	})
	if err != nil {
		return fmt.Errorf("failed to stop engine: %v", err)
	}
	return printJSON(obj)
}

func ProcessGetCmd() cli.Command {
	return cli.Command{
		Name: "get",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "name",
			},
		},
		Action: func(c *cli.Context) {
			if err := getProcess(c); err != nil {
				logrus.Fatalf("Error running engine stop command: %v.", err)
			}
		},
	}
}

func getProcess(c *cli.Context) error {
	if c.String("name") == "" {
		return fmt.Errorf("missing required parameter")
	}

	url := c.GlobalString("url")
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("cannot connect to %v: %v", url, err)
	}
	defer conn.Close()

	client := rpc.NewLonghornProcessLauncherServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), FrontendTimeout)
	defer cancel()

	obj, err := client.ProcessGet(ctx, &rpc.ProcessGetRequest{
		Name: c.String("name"),
	})
	if err != nil {
		return fmt.Errorf("failed to get engine: %v", err)
	}
	return printJSON(obj)
}

func ProcessListCmd() cli.Command {
	return cli.Command{
		Name:      "list",
		ShortName: "ls",
		Action: func(c *cli.Context) {
			if err := listProcess(c); err != nil {
				logrus.Fatalf("Error running engine stop command: %v.", err)
			}
		},
	}
}

func listProcess(c *cli.Context) error {
	url := c.GlobalString("url")
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("cannot connect to %v: %v", url, err)
	}
	defer conn.Close()

	client := rpc.NewLonghornProcessLauncherServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), FrontendTimeout)
	defer cancel()

	obj, err := client.ProcessList(ctx, &rpc.ProcessListRequest{})
	if err != nil {
		return fmt.Errorf("failed to list engine: %v", err)
	}
	return printJSON(obj)
}

func printJSON(obj interface{}) error {
	output, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return err
	}

	fmt.Println(string(output))
	return nil
}
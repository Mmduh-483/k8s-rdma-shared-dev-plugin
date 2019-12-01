package resources

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	//"github.com/pkg/errors"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	registerapi "k8s.io/kubernetes/pkg/kubelet/apis/pluginregistration/v1"
)

var (
	root   *os.File
	tmpDir string
)

// Implementation of pluginapi.RegistrationServer for use in tests.
type fakeRegistrationServer struct {
	grpcServer      *grpc.Server
	sockDir         string
	pluginEndpoint  string
	failOnRegister  bool
	pluginWatchMode bool
}

func createFakeRegistrationServer(sockDir, endpoint string) *fakeRegistrationServer {
	return &fakeRegistrationServer{
		sockDir:        sockDir,
		pluginEndpoint: endpoint,
	}
}

func (s *fakeRegistrationServer) dial() (registerapi.RegistrationClient, *grpc.ClientConn, error) {
	sockPath := path.Join(s.sockDir, s.pluginEndpoint)
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c, err := grpc.DialContext(ctx, sockPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial socket %s, err: %v", sockPath, err)
	}

	return registerapi.NewRegistrationClient(c), c, nil
}

func (s *fakeRegistrationServer) getInfo(ctx context.Context, client registerapi.RegistrationClient) (*registerapi.PluginInfo, error) {
	infoResp, err := client.GetInfo(ctx, &registerapi.InfoRequest{})
	if err != nil {
		return infoResp, fmt.Errorf("failed to get plugin info using RPC GetInfo, err: %v", err)
	}

	return infoResp, nil
}

func (s *fakeRegistrationServer) notifyPlugin(client registerapi.RegistrationClient, registered bool, errStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := &registerapi.RegistrationStatus{
		PluginRegistered: registered,
		Error:            errStr,
	}

	if _, err := client.NotifyRegistrationStatus(ctx, status); err != nil {
		//return errors.Wrap(err, errStr)
		return fmt.Errorf("")
	}

	if errStr != "" {
		return fmt.Errorf("")
		//return errors.New(errStr)
	}

	return nil
}

func (s *fakeRegistrationServer) registerPlugin() error {
	client, conn, err := s.dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = s.getInfo(ctx, client)
	if err != nil {
		return err
	}

	if err = s.notifyPlugin(client, true, ""); err != nil {
		return err
	}

	if s.failOnRegister {
		return fmt.Errorf("fake registering error")
	}

	return nil
}

func (s *fakeRegistrationServer) Register(ctx context.Context, r *pluginapi.RegisterRequest) (api *pluginapi.Empty, err error) {
	s.pluginEndpoint = r.Endpoint
	api = &pluginapi.Empty{}
	if s.failOnRegister {
		err = fmt.Errorf("fake registering error")
	}
	return
}

func (s *fakeRegistrationServer) start() {
	l, err := net.Listen("unix", path.Join(s.sockDir, kubeEndPoint))
	if err != nil {
		panic(err)
	}
	s.grpcServer = grpc.NewServer()
	pluginapi.RegisterRegistrationServer(s.grpcServer, s)
	go s.grpcServer.Serve(l)
	s.waitForServer(5 * time.Second)
}

func (s *fakeRegistrationServer) waitForServer(timeout time.Duration) error {
	maxWaitTime := time.Now().Add(timeout)
	for {
		if time.Now().After(maxWaitTime) {
			return fmt.Errorf("waiting for the fake registration server timed out")
		}
		c, err := net.DialTimeout("unix", path.Join(s.sockDir, kubeEndPoint), time.Second)
		if err == nil {
			return c.Close()
		}
	}
}

func (s *fakeRegistrationServer) stop() {
	s.grpcServer.Stop()
	os.Remove(pluginapi.DevicePluginPath)
}

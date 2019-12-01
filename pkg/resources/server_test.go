package resources

import (
	"github.com/Mellanox/k8s-rdma-shared-dev-plugin/pkg/types"
	"github.com/Mellanox/k8s-rdma-shared-dev-plugin/pkg/utils"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	registerapi "k8s.io/kubernetes/pkg/kubelet/apis/pluginregistration/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("newResourceServer", func() {
	Context("newResourcesServer", func() {
		It("server with plugin watcher enabled", func() {
			fs := utils.FakeFilesystem{
				Dirs:     []string{"sys/class/net/ib0/"},
				Symlinks: map[string]string{"sys/class/net/ib0/device": "../../../0000:02:00.0"},
			}
			defer fs.Use()()
			conf := &types.UserConfig{ResourceName: "test_server", RdmaHcaMax: 100, Devices: []string{"ib0"}}
			obj := newResourceServer(conf, true, "rdma", "socket")
			rs := obj.(*resourceServer)
			Expect(rs.resourceName).To(Equal("rdma/test_server"))
			Expect(rs.endPoint).To(Equal("test_server.socket"))
			Expect(rs.watchMode).To(Equal(true))
			Expect(len(rs.devs)).To(Equal(100))
		})
		It("server with plugin watcher disabled", func() {
			fs := utils.FakeFilesystem{
				Dirs:     []string{"sys/class/net/ib0/"},
				Symlinks: map[string]string{"sys/class/net/ib0/device": "../../../0000:02:00.0"},
			}
			defer fs.Use()()
			conf := &types.UserConfig{ResourceName: "test_server", RdmaHcaMax: 100, Devices: []string{"ib0"}}
			obj := newResourceServer(conf, false, "rdma", "socket")
			rs := obj.(*resourceServer)
			Expect(rs.resourceName).To(Equal("rdma/test_server"))
			Expect(rs.endPoint).To(Equal("test_server.socket"))
			Expect(rs.watchMode).To(Equal(false))
			Expect(len(rs.devs)).To(Equal(100))
		})
	})
	Context("Start", func() {
		depSockDir := deprecatedSockDir
		actSockDir := activeSockDir
		var fs utils.FakeFilesystem
		var fsClean func()
		BeforeEach(func() {
			fs = utils.FakeFilesystem{}
			fsClean = fs.Use()
			deprecatedSockDir = fs.RootDir
			activeSockDir = fs.RootDir
		})
		AfterEach(func() {
			deprecatedSockDir = depSockDir
			activeSockDir = actSockDir
			fsClean()
		})
		It("start server with plugin watcher enabled", func() {
			conf := &types.UserConfig{ResourceName: "fake"}
			obj := newResourceServer(conf, true, "fake", "sock")
			rs := obj.(*resourceServer)
			registrationServer := createFakeRegistrationServer(fs.RootDir, "fake.sock")
			err := rs.Start()
			registrationServer.start()
			Expect(err).ToNot(HaveOccurred())
		})
		It("start server with plugin watcher disabled", func() {
			conf := &types.UserConfig{ResourceName: "fake"}
			obj := newResourceServer(conf, false, "fake", "sock")
			rs := obj.(*resourceServer)
			registrationServer := createFakeRegistrationServer(fs.RootDir, "fake.sock")
			registrationServer.start()
			err := rs.Start()
			Expect(err).ToNot(HaveOccurred())
		})
		It("start server with non-existing path", func() {
			conf := &types.UserConfig{ResourceName: "fake"}
			activeSockDir = "non existing path"
			obj := newResourceServer(conf, true, "fake", "sock")
			rs := obj.(*resourceServer)
			err := rs.Start()
			Expect(err).To(HaveOccurred())
		})
		It("failed to register plugin with watcher disabled", func() {
			conf := &types.UserConfig{ResourceName: "fake"}
			obj := newResourceServer(conf, false, "fake", "sock")
			rs := obj.(*resourceServer)
			registrationServer := createFakeRegistrationServer(fs.RootDir, "fake.sock")
			err := rs.Start()
			registrationServer.start()
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Stop", func() {
		It("stop server with plugin watcher enabled", func() {
			rs := resourceServer{watchMode: true, stopWatcher: make(chan bool), stop: make(chan interface{})}
			err := rs.Stop()
			Expect(err).ToNot(HaveOccurred())
		})
		It("stop non existing server", func() {
			rs := resourceServer{}
			err := rs.Stop()
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Context("Restart", func() {
		depSockDir := deprecatedSockDir
		actSockDir := activeSockDir
		var fs utils.FakeFilesystem
		var fsClean func()
		BeforeEach(func() {
			fs = utils.FakeFilesystem{}
			fsClean = fs.Use()
			deprecatedSockDir = fs.RootDir
			activeSockDir = fs.RootDir
		})
		AfterEach(func() {
			deprecatedSockDir = depSockDir
			activeSockDir = actSockDir
			fsClean()
		})
		It("Restart server with plugin", func() {
			conf := &types.UserConfig{ResourceName: "fake"}
			obj := newResourceServer(conf, true, "fake", "sock")
			rs := obj.(*resourceServer)
			registrationServer := createFakeRegistrationServer(fs.RootDir, "fake.sock")
			err := rs.Restart()
			registrationServer.start()
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Context("Watch", func() {
		depSockDir := deprecatedSockDir
		var fs utils.FakeFilesystem
		var fsClean func()
		BeforeEach(func() {
			fs = utils.FakeFilesystem{}
			fsClean = fs.Use()
			deprecatedSockDir = fs.RootDir
		})
		AfterEach(func() {
			deprecatedSockDir = depSockDir
			fsClean()
		})
		It("Watch for sockdir", func() {
			conf := &types.UserConfig{ResourceName: "fake"}
			obj := newResourceServer(conf, true, "fake", "sock")
			rs := obj.(*resourceServer)
			go rs.Watch()
		})
	})
	Context("GetInfo", func() {
		It("GetInfo of plugin", func() {
			rs := resourceServer{resourceName: "fake", endPoint: "fake.sock"}
			resp, err := rs.GetInfo(nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Type).To(Equal(registerapi.DevicePlugin))
			Expect(resp.Name).To(Equal("fake"))
			Expect(resp.Endpoint).To(Equal("/var/lib/kubelet/plugins_registry/fake.sock"))
		})
	})
	Context("GetInfo", func() {
		It("GetInfo of plugin", func() {
			rs := resourceServer{resourceName: "fake", endPoint: "fake.sock"}
			resp, err := rs.GetInfo(nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Type).To(Equal(registerapi.DevicePlugin))
			Expect(resp.Name).To(Equal("fake"))
			Expect(resp.Endpoint).To(Equal("/var/lib/kubelet/plugins_registry/fake.sock"))
		})
	})
	Context("Allocate", func() {
		It("Allocate resource", func() {
			rs := resourceServer{resourceName: "fake", endPoint: "fake.sock"}
			req := &pluginapi.AllocateRequest{ContainerRequests: []*pluginapi.ContainerAllocateRequest{nil, nil}}
			res, err := rs.Allocate(nil, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.ContainerResponses)).To(Equal(2))
		})
	})
})

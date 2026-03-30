package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Suite")
}

var _ = Describe("NewIOMMUFDDevicePlugin", func() {
	It("should create a plugin with the correct number of devices", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		Expect(dp.devs).To(HaveLen(maxDevices))
	})

	It("should set all devices to healthy", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		for _, dev := range dp.devs {
			Expect(dev.Health).To(Equal(pluginapi.Healthy))
		}
	})

	It("should assign unique device IDs", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		ids := make(map[string]struct{})
		for _, dev := range dp.devs {
			_, exists := ids[dev.ID]
			Expect(exists).To(BeFalse(), "duplicate device ID: %s", dev.ID)
			ids[dev.ID] = struct{}{}
		}
	})

	It("should set the correct resource name", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		Expect(dp.resourceName).To(Equal("devices.kubevirt.io/iommufd"))
	})

	It("should set the correct socket path", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		Expect(dp.socketPath).To(Equal(filepath.Join(devicePluginPath, "kubevirt-iommufd.sock")))
	})

	It("should store the socket directory", func() {
		dp := NewIOMMUFDDevicePlugin("/custom/socket/dir")
		Expect(dp.socketDir).To(Equal("/custom/socket/dir"))
	})
})

var _ = Describe("GetDevicePluginOptions", func() {
	It("should return options with PreStartRequired false", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		opts, err := dp.GetDevicePluginOptions(context.Background(), &pluginapi.Empty{})
		Expect(err).NotTo(HaveOccurred())
		Expect(opts.PreStartRequired).To(BeFalse())
	})
})

var _ = Describe("GetPreferredAllocation", func() {
	It("should return an empty response", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		resp, err := dp.GetPreferredAllocation(context.Background(), &pluginapi.PreferredAllocationRequest{})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp).NotTo(BeNil())
	})
})

var _ = Describe("PreStartContainer", func() {
	It("should return an empty response", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		resp, err := dp.PreStartContainer(context.Background(), &pluginapi.PreStartContainerRequest{})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp).NotTo(BeNil())
	})
})

var _ = Describe("GetInitialized", func() {
	It("should return false by default", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		Expect(dp.GetInitialized()).To(BeFalse())
	})

	It("should return true after setInitialized(true)", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		dp.setInitialized(true)
		Expect(dp.GetInitialized()).To(BeTrue())
	})

	It("should return false after setInitialized(false)", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		dp.setInitialized(true)
		dp.setInitialized(false)
		Expect(dp.GetInitialized()).To(BeFalse())
	})
})

var _ = Describe("cleanup", func() {
	It("should remove the socket file", func() {
		tmpDir := GinkgoT().TempDir()
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		dp.socketPath = filepath.Join(tmpDir, "test.sock")

		f, err := os.Create(dp.socketPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())

		Expect(dp.cleanup()).To(Succeed())
		_, err = os.Stat(dp.socketPath)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("should succeed when socket file does not exist", func() {
		dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
		dp.socketPath = "/tmp/nonexistent-socket-path.sock"
		Expect(dp.cleanup()).To(Succeed())
	})
})

var _ = Describe("Allocate", func() {
	Context("when IOMMUFD is not supported", func() {
		It("should return an empty container response for each request", func() {
			dp := NewIOMMUFDDevicePlugin("/tmp/test-sockets")
			req := &pluginapi.AllocateRequest{
				ContainerRequests: []*pluginapi.ContainerAllocateRequest{
					{DevicesIds: []string{"iommufd0"}},
					{DevicesIds: []string{"iommufd1"}},
				},
			}
			resp, err := dp.Allocate(context.Background(), req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ContainerResponses).To(HaveLen(2))
			for _, cr := range resp.ContainerResponses {
				Expect(cr.Devices).To(BeEmpty())
				Expect(cr.Mounts).To(BeEmpty())
			}
		})
	})
})

var _ = Describe("supportsIOMMUFD", func() {
	It("should return false when /dev/iommu does not exist", func() {
		Expect(supportsIOMMUFD()).To(BeFalse())
	})
})

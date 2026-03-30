package plugin

import (
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"
)

var _ = Describe("createIOMMUFDSocket", func() {
	It("should create a socket and pass an FD to a connecting client", func() {
		tmpDir := GinkgoT().TempDir()

		// Create a pipe to get a valid FD to pass
		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = w.Close() }()
		fd := int(r.Fd())

		socketPath, err := createIOMMUFDSocket(fd, tmpDir, "test-id")
		Expect(err).NotTo(HaveOccurred())
		Expect(socketPath).To(Equal(filepath.Join(tmpDir, "iommufd-test-id.sock")))

		// Connect and receive the FD
		conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = conn.Close() }()

		buf := make([]byte, 1)
		oob := make([]byte, 24)
		_, oobn, _, _, err := conn.ReadMsgUnix(buf, oob)
		Expect(err).NotTo(HaveOccurred())
		Expect(oobn).To(BeNumerically(">", 0))

		// Parse the received FD
		scms, err := unix.ParseSocketControlMessage(oob[:oobn])
		Expect(err).NotTo(HaveOccurred())
		Expect(scms).NotTo(BeEmpty())

		fds, err := unix.ParseUnixRights(&scms[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(fds).To(HaveLen(1))
		defer func() { _ = unix.Close(fds[0]) }()

		// Send ACK
		_, err = conn.Write([]byte{1})
		Expect(err).NotTo(HaveOccurred())

		// Wait for the goroutine to clean up the socket
		Eventually(func() bool {
			_, err := os.Stat(socketPath)
			return os.IsNotExist(err)
		}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
	})

	It("should create the socket directory if it does not exist", func() {
		tmpDir := GinkgoT().TempDir()
		socketDir := filepath.Join(tmpDir, "subdir")

		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = w.Close() }()

		socketPath, err := createIOMMUFDSocket(int(r.Fd()), socketDir, "test-id")
		Expect(err).NotTo(HaveOccurred())

		info, err := os.Stat(socketDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())

		// Connect to clean up the goroutine
		conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
		Expect(err).NotTo(HaveOccurred())
		_ = conn.Close()
	})
})

package plugin

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ensureDirWithRelabel", func() {
	It("should create the directory", func() {
		tmpDir := GinkgoT().TempDir()
		target := tmpDir + "/subdir"
		Expect(ensureDirWithRelabel(target)).To(Succeed())
		info, err := os.Stat(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
	})

	It("should succeed when directory already exists", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(ensureDirWithRelabel(tmpDir)).To(Succeed())
	})

	It("should create nested directories", func() {
		tmpDir := GinkgoT().TempDir()
		target := tmpDir + "/a/b/c"
		Expect(ensureDirWithRelabel(target)).To(Succeed())
		info, err := os.Stat(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
	})
})

var _ = Describe("relabelPath", func() {
	It("should not return an error on systems without SELinux", func() {
		tmpDir := GinkgoT().TempDir()
		tmpFile := tmpDir + "/testfile"
		f, err := os.Create(tmpFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())

		// On systems without SELinux, this should return nil (ENOTSUP is handled gracefully)
		Expect(relabelPath(tmpFile)).To(Succeed())
	})
})

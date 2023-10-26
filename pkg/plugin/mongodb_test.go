package plugin_test

import (
	"time"

	"github.com/meln5674/grafana-mongodb-community-plugin/pkg/plugin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConvertGoTimeFormatToMongo", func() {
	It("Should convert a truncated ruby date", func() {
		converted, err := plugin.ConvertGoTimeFormatToMongo(time.RubyDate[4:])
		Expect(err).ToNot(HaveOccurred())
		Expect(converted).To(Equal("%b %d %H:%M:%S %z %Y"))
	})
})

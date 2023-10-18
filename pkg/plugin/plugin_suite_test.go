package plugin_test

import (
	"testing"

	bsonprim "go.mongodb.org/mongo-driver/bson/primitive"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Suite")
}

func init() {
	// We can't do this in Before{Suite,Each} because of how DescribeTable works
	var err error
	testDecimal, err = bsonprim.ParseDecimal128("12345.6789")
	if err != nil {
		panic(err)
	}
}

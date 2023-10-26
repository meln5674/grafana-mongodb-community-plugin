package plugin_test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"go.mongodb.org/mongo-driver/bson"
	bsonprim "go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/meln5674/grafana-mongodb-community-plugin/pkg/plugin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	now                = time.Now()
	nowMillis          = now.Truncate(1 * time.Millisecond)
	nowSeconds         = uint32(now.Unix())
	nowFromSeconds     = time.Unix(int64(nowSeconds), 0)
	testDecimal        bsonprim.Decimal128 // = bsonprim.ParseDecimal128("12345.6789") // See BeforeSuite
	testDecimalAsFloat = float64(12345.6789)
)

var _ = Describe("QueryData", func() {
	It("Should return a response", func() {
		// This is where the tests for the datasource backend live.
		ds := plugin.MongoDBDatasource{}

		resp, err := ds.QueryData(
			context.Background(),
			&backend.QueryDataRequest{
				Queries: []backend.DataQuery{
					{RefID: "A"},
				},
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Responses).To(HaveLen(1), "QueryData must return a response")
	})
})

var _ = Describe("ToGrafanaValue", func() {
	DescribeTable("should convert",
		func(inValue interface{}, expectedOutValue interface{}, expectedType data.FieldType, valid bool) {
			actualOutValue, actualType, err := plugin.ToGrafanaValue(inValue)
			if !valid {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOutValue).To(Equal(expectedOutValue))
			Expect(actualType).To(Equal(expectedType))
		},
		Entry("an int32 to an int32", int32(1), int32(1), data.FieldTypeInt32, true),
		Entry("an int64 to an int64", int64(1), int64(1), data.FieldTypeInt64, true),
		Entry("a float64 to a float64", float64(1), float64(1), data.FieldTypeFloat64, true),
		Entry("a string to a string", "Hello, world!", "Hello, world!", data.FieldTypeString, true),
		Entry("a bool to a bool", true, true, data.FieldTypeBool, true),
		Entry("a document to a json.RawMessage",
			bson.D{
				bson.E{Key: "foo", Value: int32(1)},
				bson.E{Key: "bar", Value: float64(2.5)},
				bson.E{Key: "baz", Value: "Hello, world!"},
			},
			json.RawMessage(`{"foo":1,"bar":2.5,"baz":"Hello, world!"}`),
			data.FieldTypeJSON,
			true,
		),
		Entry("a map to a json.RawMessage",
			bson.M{
				"foo": int32(1),
			},
			json.RawMessage(`{"foo":1}`),
			data.FieldTypeJSON,
			true,
		),
		Entry("an array to a json.RawMessage", bson.A{int32(1), float64(2.5), "Hello, world!"}, json.RawMessage(`[1,2.5,"Hello, world!"]`), data.FieldTypeJSON, true),
		Entry("an ObjectID to a string", bsonprim.ObjectID([12]byte{0x43, 0x78, 0x42, 0x42, 0x64, 0x4d, 0x53, 0x32, 0x37, 0x4b, 0x57, 0x30}), "43784242644d5332374b5730", data.FieldTypeString, true),
		Entry("a DateTime to a Time",
			bsonprim.NewDateTimeFromTime(now),
			nowMillis,
			data.FieldTypeTime,
			true,
		),
		Entry("a Regex to a string",
			bsonprim.Regex{Pattern: ".*"},
			".*",
			data.FieldTypeString,
			true,
		),
		Entry("a JavaScript to a string",
			bsonprim.JavaScript(`console.Log("Hello, world!");`),
			`console.Log("Hello, world!");`,
			data.FieldTypeString,
			true,
		),
		Entry("a CodeWithScope to a string",
			bsonprim.CodeWithScope{Code: `console.Log("Hello, world!");`},
			`console.Log("Hello, world!");`,
			data.FieldTypeString,
			true,
		),
		Entry("a Timestamp to a time",
			bsonprim.Timestamp{T: nowSeconds, I: 0},
			nowFromSeconds,
			data.FieldTypeTime,
			true,
		),
		Entry("a Decimal to a float64",
			testDecimal,
			testDecimalAsFloat,
			data.FieldTypeFloat64,
			true,
		),
	)
})

package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/pkg/errors"

	"go.mongodb.org/mongo-driver/bson"
	bsonPrim "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongoOpts "go.mongodb.org/mongo-driver/mongo/options"
)

// Make sure MongoDBDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler, backend.StreamHandler interfaces. Plugin should not
// implement all these interfaces - only those which are required for a particular task.
// For example if plugin does not need streaming functionality then you are free to remove
// methods that implement backend.StreamHandler. Implementing instancemgmt.InstanceDisposer
// is useful to clean up resources used by previous datasource instance when a new datasource
// instance created upon datasource settings changed.
var (
	_ backend.QueryDataHandler      = (*MongoDBDatasource)(nil)
	_ backend.CheckHealthHandler    = (*MongoDBDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*MongoDBDatasource)(nil)
)

// NewMongoDBDatasource creates a new datasource instance.
func NewMongoDBDatasource(_ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &MongoDBDatasource{}, nil
}

// MongoDBDatasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type MongoDBDatasource struct {
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewMongoDBDatasource factory function.
func (d *MongoDBDatasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *MongoDBDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData called", "request", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryType = string

const (
	queryTypeTimeseries = "Timeseries"
	queryTypeTable      = "Table"
)

type queryModel struct {
	Database        string    `json:"database"`
	Collection      string    `json:"collection"`
	QueryType       queryType `json:"queryType"`
	TimestampField  string    `json:"timestampField"`
	TimestampFormat string    `json:"timestampFormat"`
	LabelFields     []string  `json:"labelFields"`
	ValueFields     []string  `json:"valueFields"`
	ValueFieldTypes []string  `json:"valueFieldTypes"`
	AutoTimeBound   bool      `json:"autoTimeBound"`
	AutoTimeSort    bool      `json:"autoTimeSort"`
	Aggregation     string    `json:"aggregation"`
}

type frameCountDocument struct {
	Labels map[string]interface{} `bson:"_id"`
	Count  int                    `bson:"count"`
}

type timestepDocument = map[string]interface{}

func (m *queryModel) numValues() int {
	if m.QueryType == queryTypeTimeseries {
		return len(m.ValueFields) + 1
	} else {
		return len(m.ValueFields)
	}
}

func (m *queryModel) getFieldTypes() ([]data.FieldType, error) {
	types := make([]data.FieldType, 0, m.numValues())
	if m.QueryType == queryTypeTimeseries {
		types = append(types, data.FieldTypeTime)
	}
	for _, typeStr := range m.ValueFieldTypes {
		type_, ok := data.FieldTypeFromItemTypeString(typeStr)
		if !ok {
			return nil, fmt.Errorf("Invalid Type: %s", typeStr)
		}
		types = append(types, type_)
	}
	return types, nil
}
func (m *queryModel) getPipeline(from time.Time, to time.Time) (mongo.Pipeline, error) {
	pipeline := mongo.Pipeline{}
	if m.QueryType == queryTypeTimeseries && m.AutoTimeBound {
		pipeline = append(pipeline, bson.D{bson.E{
			Key: "$match",
			Value: bson.D{bson.E{
				Key: m.TimestampField,
				Value: bson.D{
					bson.E{Key: "$gte", Value: bsonPrim.NewDateTimeFromTime(from)},
					bson.E{Key: "$lte", Value: bsonPrim.NewDateTimeFromTime(to)},
				},
			}},
		}})
	}

	userPipeline := mongo.Pipeline{}
	err := bson.UnmarshalExtJSON([]byte(m.Aggregation), false, &userPipeline)
	if err != nil {
		return mongo.Pipeline{}, errors.Wrap(err, "Failed to parse aggregation pipeline")
	}
	pipeline = append(pipeline, userPipeline...)
	if m.QueryType == queryTypeTimeseries && m.AutoTimeSort {
		pipeline = append(pipeline, bson.D{
			bson.E{
				Key:   "$sort",
				Value: bson.D{bson.E{Key: m.TimestampField, Value: 1}},
			},
		})
	}
	return pipeline, nil
}

func (m *queryModel) getLabels(doc map[string]interface{}) data.Labels {
	labels := make(map[string]string, len(m.LabelFields))
	if m.QueryType == queryTypeTimeseries {
		for _, labelKey := range m.LabelFields {
			labelValue, ok := doc[labelKey]
			if ok {
				labels[labelKey] = fmt.Sprintf("%v", labelValue)
			}
		}
	}
	return data.Labels(labels)
}

func (m *queryModel) getValues(doc map[string]interface{}) ([]interface{}, error) {
	values := make([]interface{}, 0, m.numValues())
	var err error
	if m.QueryType == queryTypeTimeseries {
		timestamp, ok := doc[m.TimestampField]
		if !ok {
			return nil, fmt.Errorf("All documents must have the Timestamp Field present")
		}
		var convertedTimestamp time.Time
		if m.TimestampFormat == "" {
			primTimestamp, isPrim := timestamp.(bsonPrim.DateTime)
			if !isPrim {
				return nil, fmt.Errorf("Timestamps must be bson DateTimes")
			}
			if isPrim {
				convertedTimestamp = primTimestamp.Time()
			}
		} else {
			stringTimestamp, isString := timestamp.(string)
			if !isString {
				return nil, fmt.Errorf("Timestamps must be strings when Timestamp Format is supplied")
			}
			convertedTimestamp, err = time.Parse(m.TimestampFormat, stringTimestamp)
			if err != nil {
				return nil, errors.Wrap(err, "Could not parse timestamp")
			}
		}
		values = append(values, convertedTimestamp)
	}
	for _, valueKey := range m.ValueFields {
		valueValue, ok := doc[valueKey]
		if !ok {
			values = append(values, nil)
		} else if asTime, isTime := valueValue.(bsonPrim.DateTime); isTime {
			values = append(values, asTime.Time())
		} else {
			values = append(values, valueValue)
		}
	}
	return values, nil
}

type jsonData struct {
	URL            string `json:"url"`
	TLS            bool   `json:"tls"`
	TLSCertificate string `json:"tlsCertificate"`
	TLSCA          string `json:"tlsCa"`
	TLSInsecure    bool   `json:"tlsInsecure"`
	TLSServerName  string `json:"tlsServerName"`
}

func connect(ctx context.Context, pCtx backend.PluginContext) (client *mongo.Client, err error, internalErr error) {
	data := jsonData{}
	err = json.Unmarshal([]byte(pCtx.DataSourceInstanceSettings.JSONData), &data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to parse data source settings")
	}
	opts := mongoOpts.Client()

	mongoURL, err := url.Parse(data.URL)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Invalid Datasource URL %s", data.URL)), nil
	}

	opts = opts.ApplyURI(mongoURL.String())
	username, hasUsername := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["username"]
	password, hasPassword := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["password"]
	tlsCertificateKey, hasTLSCertificateKey := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["tlsCertificateKey"]

	if hasUsername && username != "" {
		if hasPassword && password != "" {
			mongoURL.User = url.UserPassword(username, password)
		} else {
			mongoURL.User = url.User(username)
		}

		credential := options.Credential{
			Username: username,
			Password: password,
		}
		opts = opts.SetAuth(credential)
	}

	if data.TLS {
		tlsConfig := &tls.Config{}
		if data.TLSCA != "" {
			tlsConfig.RootCAs = x509.NewCertPool()
			if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(data.TLSCA)) {
				return nil, fmt.Errorf("failed to add tlsCA"), nil
			}
		}
		if data.TLSInsecure {
			tlsConfig.InsecureSkipVerify = true
		}
		if (data.TLSCertificate != "") != hasTLSCertificateKey {
			return nil, fmt.Errorf("Must provide both tlsCertificate and tlsCertificateKey, or neither"), nil
		}
		if data.TLSCertificate != "" && hasTLSCertificateKey {
			clientCert, err := tls.X509KeyPair([]byte(data.TLSCertificate), []byte(tlsCertificateKey))
			if err != nil {
				return nil, errors.Wrap(err, "Failed to parse TLS Certificate-Key Pair"), nil
			}
			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}
		if data.TLSServerName != "" {
			tlsConfig.ServerName = data.TLSServerName
		}
		opts = opts.SetTLSConfig(tlsConfig)
	}

	mongoClient, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "Error while connecting to MongoDB"), nil
	}

	return mongoClient, nil, nil
}

func (m *queryModel) getLabelsID(labels data.Labels) string {
	// TODO: Might not work, need to find a fast but stable way to identify a set of labels
	// labelsID := fmt.Sprintf("%#v", map[string]string(labels))
	if len(m.LabelFields) == 0 {
		return ""
	}
	labelsID := fmt.Sprintf("%s=%s", m.LabelFields[0], labels[m.LabelFields[0]])
	for _, label := range m.LabelFields[1:] {
		labelsID += fmt.Sprintf(",%s=%s", label, labels[label])
	}
	return labelsID
}

func (m *queryModel) getFrameFieldNames(labelsID string) []string {
	fieldNames := make([]string, 0, m.numValues())
	if m.QueryType == queryTypeTimeseries {
		fieldNames = append(fieldNames, m.TimestampField)
	}
	fieldNames = append(fieldNames, m.ValueFields...)
	return fieldNames
}

func (m *queryModel) parseQueryResultDocument(frames map[string]*data.Frame, doc timestepDocument, fieldTypes []data.FieldType) (err error) {
	defer func() {
		if panic_ := recover(); panic_ != nil {
			switch panic_.(type) {
			case error:
				err = panic_.(error)
			default:
				err = fmt.Errorf("%v", panic_)
			}
		}
	}()
	labels := m.getLabels(doc)
	labelsID := m.getLabelsID(labels)
	frame, ok := frames[labelsID]
	if !ok {
		log.DefaultLogger.Debug("Creating frame for unique label combination", "doc", doc, "labels", labels, "labelsID", labelsID)
		frame = data.NewFrameOfFieldTypes(labelsID, 0, fieldTypes...)
		frame.SetFieldNames(m.getFrameFieldNames(labelsID)...)
		frames[labelsID] = frame
	}
	row, err := m.getValues(doc)
	if err != nil {
		return errors.Wrap(err, "Failed to extract value columns")
	}
	frame.AppendRow(row...)

	return nil
}

func (d *MongoDBDatasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Info("query called", "context", pCtx, "query", query)
	response := backend.DataResponse{}

	// Unmarshal the JSON into our queryModel and parse values into usable representations
	var qm queryModel

	var err error
	err = json.Unmarshal(query.JSON, &qm)
	if err != nil {
		response.Error = errors.Wrap(err, "Invalid query JSON")
		return response
	}

	fieldTypes, err := qm.getFieldTypes()
	if err != nil {
		response.Error = errors.Wrap(err, "Could not determine field types")
		return response
	}

	numUserTypes := len(fieldTypes)
	if qm.QueryType == queryTypeTimeseries {
		numUserTypes--
	}
	if qm.numValues() != len(fieldTypes) {
		response.Error = fmt.Errorf(
			"Value Fields and Value Field Types must be the same length (%d vs %d)",
			qm.numValues(),
			numUserTypes,
		)
		return response
	}

	pipeline, err := qm.getPipeline(query.TimeRange.From, query.TimeRange.To)
	if err != nil {
		response.Error = errors.Wrap(err, "Failed to produce final pipeline")
		return response
	}

	mongoClient, err, internalErr := connect(ctx, pCtx)
	if internalErr != nil {
		response.Error = errors.Wrap(internalErr, "Internal failure while connecting to mongo")
		return response
	}
	if err != nil {
		response.Error = errors.Wrap(err, "Failed to connect to mongo")
		return response
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database(qm.Database).Collection(qm.Collection)

	frames := map[string]*data.Frame{}

	log.DefaultLogger.Info("Querying MongoDB", "context", pCtx, "query", query, "pipeline", pipeline)
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		response.Error = errors.Wrap(err, "Failed to send query to mongo")
		return response
	}
	for cursor.Next(ctx) {
		doc := timestepDocument{}
		err = cursor.Decode(&doc)
		if err != nil {
			response.Error = errors.Wrap(err, "Failed to parse document")
			return response
		}
		err = qm.parseQueryResultDocument(frames, doc, fieldTypes)
		if err != nil {
			response.Error = fmt.Errorf("Bad document: %s, %v", err, doc)
			return response
		}
	}
	if cursor.Err() != nil {
		response.Error = errors.Wrap(cursor.Err(), "Failed to get next result document")
		return response
	}

	// add the frames to the response.
	response.Frames = make([]*data.Frame, 0, len(frames))
	for _, frame := range frames {
		response.Frames = append(response.Frames, frame)
	}

	log.DefaultLogger.Info("query finished", "context", pCtx, "query", query, "response", response)
	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *MongoDBDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("CheckHealth called", "request", req)
	result := backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "MongoDB is Responding",
	}
	mongoClient, err, internalErr := connect(ctx, req.PluginContext)
	if internalErr != nil {
		return nil, errors.Wrap(err, "Failed to connect to mongo")
	}
	if err != nil {
		result.Status = backend.HealthStatusError
		result.Message = err.Error()
		return &result, nil
	}
	defer mongoClient.Disconnect(ctx)
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		result.Status = backend.HealthStatusError
		result.Message = "Ping failed: " + err.Error()
		return &result, nil
	}

	return &result, nil
}

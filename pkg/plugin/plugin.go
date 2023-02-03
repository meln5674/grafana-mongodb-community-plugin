package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	sprig "github.com/go-task/slim-sprig"
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
	defaultQueryType    = queryTypeTable
)

type queryModel struct {
	Database             string    `json:"database"`
	Collection           string    `json:"collection"`
	QueryType            queryType `json:"queryType"`
	TimestampField       string    `json:"timestampField"`
	TimestampFormat      string    `json:"timestampFormat"`
	LabelFields          []string  `json:"labelFields"`
	LegendFormat         string    `json:"legendFormat"`
	ValueFields          []string  `json:"valueFields"`
	ValueFieldTypes      []string  `json:"valueFieldTypes"`
	AutoTimeBound        bool      `json:"autoTimeBound"`
	AutoTimeSort         bool      `json:"autoTimeSort"`
	Aggregation          string    `json:"aggregation"`
	SchemaInference      bool      `json:"schemaInference"`
	SchemaInferenceDepth int       `json:"schemaInferenceDepth"`
}

func (m *queryModel) resolve(fields []field) (resolvedQueryModel, error) {
	var err error

	queryType := m.QueryType
	if queryType == "" {
		queryType = defaultQueryType
	}
	switch queryType {
	case queryTypeTable:
		return &tableQueryModel{
			fields: fields,
		}, nil
	case queryTypeTimeseries:
		var legendTemplate *template.Template
		if m.LegendFormat != "" {
			legendTemplate, err = template.New("legend").Funcs(sprig.TxtFuncMap()).Parse(m.LegendFormat)
			if err != nil {
				return nil, err
			}
		}
		return &timeseriesQueryModel{
			fields:               fields,
			timestampFieldName:   m.TimestampField,
			timestampFieldFormat: m.TimestampFormat,
			labelFieldNames:      m.LabelFields,
			legendTemplate:       legendTemplate,
		}, nil
	default:
		return nil, fmt.Errorf("Query type must be one of: %s, %s", queryTypeTable, queryTypeTimeseries)
	}
}

type field struct {
	Name string
	Type data.FieldType
}

func (f *field) get(doc timestepDocument) interface{} {
	return doc[f.Name]
}

func toGrafanaValue(value interface{}) (interface{}, data.FieldType, error) {
	// Only handles types explicitly referenced as being returned from bson.Unmarshal
	// https://pkg.go.dev/go.mongodb.org/mongo-driver@v1.11.1/bson#hdr-Native_Go_Types
	// notably, this does not deal with pointer types, like *float64

	// 19
	if value == nil {
		return nil, data.FieldTypeUnknown, nil
	}
	switch v := value.(type) {
	// 1-5
	case int32, int64, float64, string, bool:
		return value, data.FieldTypeFor(value), nil
	// 6-7
	case bsonPrim.A, bsonPrim.D, bsonPrim.M, map[string]interface{}, []interface{}:
		// map[string]interface{} and []interface{} aren't documented,
		// but can be observed to be returned
		bytes, err := bson.MarshalExtJSON(value, false, false)
		if err != nil {
			return nil, data.FieldTypeUnknown, err
		}
		return json.RawMessage(bytes), data.FieldTypeJSON, err
	// 8
	case bsonPrim.ObjectID:
		bytes := [12]byte(v)
		return hex.EncodeToString(bytes[:]), data.FieldTypeString, nil
	// 9
	case bsonPrim.DateTime:
		return v.Time(), data.FieldTypeTime, nil
	// 10
	case bsonPrim.Binary:
		return hex.EncodeToString(v.Data), data.FieldTypeString, nil
	// 11
	case bsonPrim.Regex:
		return fmt.Sprintf("%s", v.Pattern), data.FieldTypeString, nil
	// 12
	case bsonPrim.JavaScript:
		return string(v), data.FieldTypeString, nil
	// 13
	case bsonPrim.CodeWithScope:
		return string(v.Code), data.FieldTypeString, nil
	// 14
	case bsonPrim.Timestamp:
		return time.Unix(int64(v.T), 0), data.FieldTypeTime, nil
	// 15
	case bsonPrim.Decimal128:
		f, err := strconv.ParseFloat(v.String(), 64)
		return f, data.FieldTypeFloat64, err
	// 16-17
	case bsonPrim.MinKey, bsonPrim.MaxKey:
		return fmt.Sprintf("%#v", v), data.FieldTypeString, nil
	// 18
	case bsonPrim.Undefined:
		return nil, data.FieldTypeUnknown, nil
	// 19: See above
	// 20
	case bsonPrim.DBPointer:
		return fmt.Sprintf("%#v", v), data.FieldTypeString, nil
	// 21
	case bsonPrim.Symbol:
		return string(v), data.FieldTypeString, nil
	}
	return nil, data.FieldTypeUnknown, fmt.Errorf("Got value with a type not expected to be generated by BSON: %#v (%s)", value, reflect.ValueOf(value).Type())
}

func convertValue(value interface{}, nullable bool) (interface{}, data.FieldType, error) {
	converted, type_, err := toGrafanaValue(value)
	if err != nil {
		return nil, type_, err
	}
	if converted == nil {
		return nil, type_, nil
	}
	if !nullable {
		return converted, type_, nil
	}

	// Adding e.g. a float64 to a frame of *float64 is not handled seamlessly,
	// we have do it manually
	// We can't just do valueValueValue.Addr().Interface(), as scalar's aren't addressable
	convertedValue := reflect.ValueOf(converted)
	convertedPtr := reflect.New(convertedValue.Type())
	convertedPtr.Elem().Set(convertedValue)
	return convertedPtr.Interface(), type_.NullableType(), nil
}

type resolvedQueryModel interface {
	makeFrame(id string, labels data.Labels) (*data.Frame, error)
	getLabels(doc timestepDocument) (labels data.Labels, labelsID string)
	getValues(doc timestepDocument) ([]interface{}, error)
}

type tableQueryModel struct {
	fields []field
}

func (m *tableQueryModel) makeFrame(id string, labels data.Labels) (*data.Frame, error) {
	names := make([]string, len(m.fields))
	types := make([]data.FieldType, len(m.fields))

	for ix, field := range m.fields {
		names[ix] = field.Name
		types[ix] = field.Type
	}
	frame := data.NewFrameOfFieldTypes(id, 0, types...)
	frame.SetFieldNames(names...)

	return frame, nil
}

func (m *tableQueryModel) getLabels(doc timestepDocument) (data.Labels, string) {
	return make(data.Labels), ""
}

func (m *tableQueryModel) getValues(doc timestepDocument) ([]interface{}, error) {
	var err error
	var actualType data.FieldType
	values := make([]interface{}, len(m.fields))
	for ix, field := range m.fields {
		name := field.Name
		type_ := field.Type
		value, ok := doc[name]
		if !ok || value == nil {
			if !type_.Nullable() {
				return nil, fmt.Errorf("Field %s was null or absent, but is not nullable. If using schema inference, please increase the depth to the first document missing this field, or manually specify the schema", name)
			}
			values[ix] = nil
			continue
		}

		values[ix], actualType, err = convertValue(value, type_.Nullable())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to convert value for %s", name))
		}

		if values[ix] != nil && actualType != type_ {
			return nil, fmt.Errorf("Type mismatch for field %s: expected %s, got %s (%#v)", name, type_, actualType, values[ix])
		}
	}
	return values, nil
}

var _ = resolvedQueryModel(&tableQueryModel{})

type timeseriesQueryModel struct {
	timestampFieldName   string
	timestampFieldFormat string
	labelFieldNames      []string
	legendTemplate       *template.Template
	fields               []field
}

var _ = resolvedQueryModel(&timeseriesQueryModel{})

func (m *timeseriesQueryModel) makeFrame(id string, labels data.Labels) (*data.Frame, error) {
	names := make([]string, 1+len(m.fields))
	types := make([]data.FieldType, 1+len(m.fields))
	names[0] = m.timestampFieldName
	types[0] = data.FieldTypeTime
	valueNames := names[1:]
	valueTypes := types[1:]
	for ix, field := range m.fields {
		valueNames[ix] = field.Name
		valueTypes[ix] = field.Type
	}
	frame := data.NewFrameOfFieldTypes(id, 0, types...)
	frame.SetFieldNames(names...)
	for _, field := range frame.Fields {
		field.Labels = labels
		displayName, err := m.getDisplayName(field.Name, field.Labels)
		if err != nil {
			return nil, err
		}
		if displayName == "" {
			continue
		}
		field.Config = &data.FieldConfig{
			DisplayNameFromDS: displayName,
		}
	}

	return frame, nil
}

func (m *timeseriesQueryModel) getLabels(doc timestepDocument) (data.Labels, string) {
	// TODO: Might not work, need to find a fast but stable way to identify a set of labels
	// labelsID := fmt.Sprintf("%#v", map[string]string(labels))

	labels := make(data.Labels, len(m.labelFieldNames))

	labelsID := strings.Builder{}

	for ix, key := range m.labelFieldNames {
		value, ok := doc[key]
		if !ok {
			continue
		}
		labels[key] = fmt.Sprintf("%v", value)

		if ix != 0 {
			labelsID.WriteString(",")
		}
		labelsID.WriteString(fmt.Sprintf("%s=%v", key, value))

	}

	return labels, labelsID.String()
}

func (m *timeseriesQueryModel) getDisplayName(valueField string, labels data.Labels) (string, error) {
	if m.legendTemplate == nil {
		return "", nil
	}

	builder := strings.Builder{}
	err := m.legendTemplate.Execute(&builder, map[string]interface{}{
		"Value":  valueField,
		"Labels": labels,
	})
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func (m *timeseriesQueryModel) convertTimestamp(timestamp interface{}) (time.Time, error) {
	if m.timestampFieldFormat == "" {
		primTimestamp, isPrim := timestamp.(bsonPrim.DateTime)
		if !isPrim {
			return time.Time{}, fmt.Errorf("Timestamps must be bson DateTimes")
		}
		if isPrim {
			return primTimestamp.Time(), nil
		}
	}
	stringTimestamp, isString := timestamp.(string)
	if !isString {
		return time.Time{}, fmt.Errorf("Timestamps must be strings when Timestamp Format is supplied")
	}
	convertedTimestamp, err := time.Parse(m.timestampFieldFormat, stringTimestamp)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "Could not parse timestamp")
	}
	return convertedTimestamp, nil
}

func (m *timeseriesQueryModel) getValues(doc timestepDocument) ([]interface{}, error) {
	var err error
	values := make([]interface{}, 1+len(m.fields))

	timestamp, ok := doc[m.timestampFieldName]
	if !ok {
		return nil, fmt.Errorf("All documents must have the Timestamp Field present")
	}
	values[0], err = m.convertTimestamp(timestamp)
	if err != nil {
		return nil, err
	}

	valueValues := values[1:]
	var actualType data.FieldType
	for ix, field := range m.fields {
		name := field.Name
		value := doc[name]
		type_ := field.Type
		valueValues[ix], actualType, err = convertValue(value, type_.Nullable())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to convert value for %s", name))
		}
		if actualType != type_ {
			return nil, fmt.Errorf("Type mismatch for field %s: expected %s, got %s", name, type_, actualType)
		}
	}

	return values, nil
}

type frameCountDocument struct {
	Labels map[string]interface{} `bson:"_id"`
	Count  int                    `bson:"count"`
}

type timestepDocument = map[string]interface{}

func (m *queryModel) getFields() ([]field, error) {
	if len(m.ValueFields) != len(m.ValueFields) {
		return nil, fmt.Errorf(
			"Value Fields and Value Field Types must be the same length (%d vs %d)",
			len(m.ValueFields),
			len(m.ValueFields),
		)
	}
	fields := make([]field, len(m.ValueFields))
	var ok bool
	for ix, typeStr := range m.ValueFieldTypes {
		fields[ix].Name = m.ValueFields[ix]
		fields[ix].Type, ok = data.FieldTypeFromItemTypeString(typeStr)
		if !ok {
			return nil, fmt.Errorf("Invalid Type: %s", typeStr)
		}
	}
	return fields, nil
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

type jsonData struct {
	URL            string `json:"url"`
	TLS            bool   `json:"tls"`
	TLSCertificate string `json:"tlsCertificate"`
	TLSCA          string `json:"tlsCa"`
	TLSInsecure    bool   `json:"tlsInsecure"`
	TLSServerName  string `json:"tlsServerName"`
}

type secureJsonData struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	TLSCertificateKey string `json:"tlsCertificateKey"`
}

type datasource struct {
	jsonData
	secureJsonData
}

func (d *datasource) getAuth() (*options.Credential, error) {
	if d.Username == "" {
		return nil, nil
	}
	/*if d.Password != "" {
		mongoURL.User = url.UserPassword(d.Username, d.Password)
	} else {
		mongoURL.User = url.User(d.Username)
	}*/

	return &options.Credential{
		Username: d.Username,
		Password: d.Password,
	}, nil
}

func (d *datasource) getTLS() (*tls.Config, error) {
	if !d.TLS {
		return nil, nil
	}
	tlsConfig := &tls.Config{}
	if d.TLSCA != "" {
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(d.TLSCA)) {
			return nil, fmt.Errorf("failed to add tlsCA")
		}
	}
	if d.TLSInsecure {
		tlsConfig.InsecureSkipVerify = true
	}
	if (d.TLSCertificate != "") != (d.TLSCertificateKey != "") {
		return nil, fmt.Errorf("Must provide both tlsCertificate and tlsCertificateKey, or neither")
	}
	if d.TLSCertificate != "" && d.TLSCertificateKey != "" {
		clientCert, err := tls.X509KeyPair([]byte(d.TLSCertificate), []byte(d.TLSCertificateKey))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse TLS Certificate-Key Pair")
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	if d.TLSServerName != "" {
		tlsConfig.ServerName = d.TLSServerName
	}
	return tlsConfig, nil
}

func connect(ctx context.Context, pCtx backend.PluginContext) (client *mongo.Client, err error, internalErr error) {
	data := datasource{}
	err = json.Unmarshal([]byte(pCtx.DataSourceInstanceSettings.JSONData), &data.jsonData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to parse data source settings")
	}
	secureJsonData, err := json.Marshal(pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to remarshal secure data source settings")
	}
	err = json.Unmarshal(secureJsonData, &data.secureJsonData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to parse data source settings")
	}
	opts := mongoOpts.Client()

	mongoURL, err := url.Parse(data.URL)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Invalid Datasource URL %s", data.URL)), nil
	}

	opts = opts.ApplyURI(mongoURL.String())
	credential, err := data.getAuth()
	if err != nil {
		return nil, err, nil
	}
	if credential != nil {
		opts.SetAuth(*credential)
	}
	tlsConfig, err := data.getTLS()
	if err != nil {
		return nil, err, nil
	}
	if tlsConfig != nil {
		opts.SetTLSConfig(tlsConfig)
	}

	mongoClient, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "Error while connecting to MongoDB"), nil
	}

	return mongoClient, nil, nil
}

type resultParser struct {
	frames map[string]*data.Frame
	model  resolvedQueryModel
}

func (p *resultParser) parseQueryResultDocument(doc timestepDocument) (err error) {
	defer func() {
		if panic_ := recover(); panic_ != nil {
			buf := make([]byte, 1<<16)
			buflen := runtime.Stack(buf, false)
			log.DefaultLogger.Error("Panic while parsing document", "document", doc, "error", panic_, "trace", string(buf[:buflen]))
			switch panic_.(type) {
			case error:
				err = panic_.(error)
			default:
				err = fmt.Errorf("%v", panic_)
			}
		}
	}()
	labels, labelsID := p.model.getLabels(doc)
	frame, ok := p.frames[labelsID]
	if !ok {
		log.DefaultLogger.Debug("Creating frame for unique label combination", "doc", doc, "labels", labels, "labelsID", labelsID)
		frame, err = p.model.makeFrame(labelsID, labels)
		if err != nil {
			return err
		}
		p.frames[labelsID] = frame
	}
	row, err := p.model.getValues(doc)
	if err != nil {
		return errors.Wrap(err, "Failed to extract value columns")
	}
	log.DefaultLogger.Debug("Parsed row", "row", row, "id", labelsID)
	frame.AppendRow(row...)

	return nil
}

type bufferingCursor struct {
	*mongo.Cursor
	buffer []timestepDocument
}

func (c *bufferingCursor) Next(ctx context.Context) (doc timestepDocument, more bool, err error) {
	more = c.Cursor.Next(ctx)
	if !more {
		err = c.Cursor.Err()
		return
	}

	doc = make(timestepDocument)
	err = c.Cursor.Decode(&doc)
	if err != nil {
		more = false
		return
	}

	c.buffer = append(c.buffer, doc)
	more = true
	return
}

type bufferedCursor struct {
	*mongo.Cursor
	buffer []timestepDocument
}

func (c *bufferedCursor) Next(ctx context.Context) (doc timestepDocument, more bool, decodeErr bool, err error) {
	if len(c.buffer) != 0 {
		doc = c.buffer[0]
		c.buffer = c.buffer[1:]
		more = true
		return
	}

	more = c.Cursor.Next(ctx)
	if !more {
		err = c.Cursor.Err()
		return
	}

	doc = make(timestepDocument)
	err = c.Decode(&doc)
	if err != nil {
		decodeErr = true
		more = false
	}
	return
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

	log.DefaultLogger.Debug("Query Model Parsed", "queryModel", qm)

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

	log.DefaultLogger.Info("Querying MongoDB", "context", pCtx, "query", query, "pipeline", pipeline)
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		response.Error = errors.Wrap(err, "Failed to send query to mongo")
		return response
	}

	buffered := bufferedCursor{
		Cursor: cursor,
	}

	var fields []field

	if qm.SchemaInference {
		buffering := bufferingCursor{
			Cursor: cursor,
			buffer: make([]timestepDocument, 0, qm.SchemaInferenceDepth),
		}

		ignored := make(map[string]struct{}, 1+len(qm.LabelFields))
		if qm.QueryType == queryTypeTimeseries {
			ignored[qm.TimestampField] = struct{}{}
		}
		for _, name := range qm.LabelFields {
			ignored[name] = struct{}{}
		}

		state := NewSchemaInference(ignored)

		doc, more, err := buffering.Next(ctx)
		for len(buffering.buffer) < qm.SchemaInferenceDepth && more {
			err = state.updateDoc(doc)
			if err != nil {
				break
			}

			doc, more, err = buffering.Next(ctx)
		}
		if err != nil {
			response.Error = errors.Wrap(err, "Schema Inference Failed")
			return response
		}
		fields = state.finish()
		log.DefaultLogger.Debug(
			"Inferred schema",
			"requestedDocs", qm.SchemaInferenceDepth,
			"bufferedDocs", len(buffering.buffer),
			"fields", fields,
		)
		buffered.buffer = buffering.buffer
	} else {
		fields, err = qm.getFields()
		if err != nil {
			response.Error = err
			return response
		}
	}

	resolvedModel, err := qm.resolve(fields)
	if err != nil {
		response.Error = err
		return response
	}

	log.DefaultLogger.Debug(
		"Resolved query model",
		"model", resolvedModel,
	)

	parser := resultParser{
		frames: map[string]*data.Frame{},
		model:  resolvedModel,
	}

	docCount := 0
	doc, more, decodeErr, err := buffered.Next(ctx)
	for more {
		err = parser.parseQueryResultDocument(doc)
		if err != nil {
			response.Error = fmt.Errorf("Failed to convert document number %d: %s, %v", docCount, err, doc)
			return response
		}
		doc, more, decodeErr, err = buffered.Next(ctx)
		docCount++
	}
	if err != nil {
		if decodeErr {
			response.Error = errors.Wrap(err, fmt.Sprintf("Failed to decode document number %d", docCount))
			return response
		} else {
			response.Error = errors.Wrap(err, fmt.Sprintf("Failed to fetch result document number %d", docCount+1))
			return response
		}
	}
	log.DefaultLogger.Info(fmt.Sprintf("Processed %d documents", docCount))

	// add the frames to the response.
	response.Frames = make([]*data.Frame, 0, len(parser.frames))
	for _, frame := range parser.frames {
		response.Frames = append(response.Frames, frame)
	}

	log.DefaultLogger.Debug("query finished", "context", pCtx, "query", query, "response", response)
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

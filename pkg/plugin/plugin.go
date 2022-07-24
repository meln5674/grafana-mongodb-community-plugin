package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"go.mongodb.org/mongo-driver/bson"
	bsonPrim "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

type queryModel struct {
	Database        string   `json:"database"`
	Collection      string   `json:"collection"`
	TimestampField  string   `json:"timestampField"`
	LabelFields     []string `json:"labelFields"`
	ValueFields     []string `json:"valueFields"`
	ValueFieldTypes []string `json:"valueFieldTypes"`

	Aggregation string `json:"aggregation"`
}

type frameCountDocument struct {
	Labels map[string]interface{} `bson:"_id"`
	Count  int                    `bson:"count"`
}

type timestepDocument = map[string]interface{}

func (m *queryModel) getFieldTypes() ([]data.FieldType, error) {
	types := make([]data.FieldType, 0, len(m.ValueFieldTypes)+1)
	types = append(types, data.FieldTypeTime)
	for _, typeStr := range m.ValueFieldTypes {
		type_, ok := data.FieldTypeFromItemTypeString(typeStr)
		if !ok {
			return nil, fmt.Errorf("Invalid Type: %s", typeStr)
		}
		types = append(types, type_)
	}
	return types, nil
}

func (m *queryModel) getPipeline() (mongo.Pipeline, error) {
	pipeline := mongo.Pipeline{}
	err := bson.UnmarshalExtJSON([]byte(m.Aggregation), true, &pipeline)
	return pipeline, err
}

func (m *queryModel) getLabels(doc map[string]interface{}) data.Labels {
	labels := make(map[string]string, len(m.LabelFields))
	for _, labelKey := range m.LabelFields {
		labelValue, ok := doc[labelKey]
		if ok {
			labels[labelKey] = fmt.Sprintf("%v", labelValue)
		}
	}
	return data.Labels(labels)
}

func (m *queryModel) getValues(doc map[string]interface{}) ([]interface{}, error) {
	values := make([]interface{}, 0, len(m.ValueFields)+1)
	timestamp, ok := doc[m.TimestampField]
	if !ok {
		return nil, fmt.Errorf("All documents must have the Timestamp Field present")
	}
	primTimestamp, isPrim := timestamp.(bsonPrim.DateTime)
	if !(isPrim) {
		return nil, fmt.Errorf("Timestamps must be bson DateTimes")
	}
	var convertedTimestamp time.Time
	if isPrim {
		convertedTimestamp = primTimestamp.Time()
	}
	values = append(values, convertedTimestamp)
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

func (m *queryModel) getCountPipelineTail() (mongo.Pipeline, error) {
	id := bson.D{}
	for _, field := range m.LabelFields {
		id = append(id, bsonPrim.E{Key: field, Value: "$" + field})
	}
	stage := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: id},
			{Key: "count", Value: bson.D{
				{Key: "$sum", Value: 1},
			}},
		}},
	}

	return mongo.Pipeline{stage}, nil
}

type jsonData struct {
	URL string `json:"url"`
}

func connect(ctx context.Context, pCtx backend.PluginContext) (client *mongo.Client, errMsg string, err error, internalErr error) {
	data := jsonData{}
	err = json.Unmarshal([]byte(pCtx.DataSourceInstanceSettings.JSONData), &data)
	if err != nil {
		return nil, "", nil, err
	}
	mongoURL, err := url.Parse(data.URL)
	if err != nil {
		return nil, "Invalid URL: ", err, nil
	}

	username, hasUsername := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["username"]
	password, hasPassword := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["username"]

	if hasUsername {
		if hasPassword {
			mongoURL.User = url.UserPassword(username, password)
		} else {
			mongoURL.User = url.User(username)
		}
	}

	mongoClient, err := mongo.Connect(ctx, mongoOpts.Client().ApplyURI(mongoURL.String()))
	if err != nil {
		return nil, "Error while connecting to MongoDB: ", err, nil
	}

	return mongoClient, "", nil, nil
}

func (d *MongoDBDatasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Info("query called", "context", pCtx, "query", query)
	response := backend.DataResponse{}

	// Unmarshal the JSON into our queryModel and parse values into usable representations
	var qm queryModel

	var err error
	err = json.Unmarshal(query.JSON, &qm)
	if err != nil {
		response.Error = err
		return response
	}

	fieldTypes, err := qm.getFieldTypes()
	if err != nil {
		response.Error = err
		return response
	}

	if len(qm.ValueFields)+1 != len(fieldTypes) {
		response.Error = fmt.Errorf("Value Fields and Value Field Types must be the same length (%d vs %d)", len(qm.ValueFields), len(fieldTypes)-1)
		return response
	}

	pipeline, err := qm.getPipeline()
	if err != nil {
		response.Error = err
		return response
	}

	/*
		// Make a modified version of the requested aggregation pipeline which counts the number of documents
		// by each unique combination of the label fields
		countPipelineTail, err := qm.getCountPipelineTail()
		if err != nil {
			response.Error = err
			return response
		}

		countPipeline := make(mongo.Pipeline, 0, len(pipeline)+len(countPipelineTail))
		countPipeline = append(countPipeline, pipeline...)
		countPipeline = append(countPipeline, countPipelineTail...)
	*/

	mongoClient, _, err, internalErr := connect(ctx, pCtx)
	if internalErr != nil {
		response.Error = internalErr
		return response
	}
	if err != nil {
		response.Error = err
		return response
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database(qm.Database).Collection(qm.Collection)

	frames := map[string]*data.Frame{}

	log.DefaultLogger.Info("Querying MongoDB", "context", pCtx, "query", query, "pipeline", pipeline)
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		response.Error = err
		return response
	}

	for cursor.Next(ctx) {
		doc := timestepDocument{}
		err = cursor.Decode(&doc)
		if err != nil {
			response.Error = err
			return response
		}
		labels := qm.getLabels(doc)
		labelsID := fmt.Sprintf("%#v", labels)
		// TODO: Might not work, need to find a fast but stable way to identify a set of labels
		frame, ok := frames[labelsID]
		if !ok {
			log.DefaultLogger.Debug("Creating frame for unique label combination", "context", pCtx, "query", query, "doc", doc, "labels", labels, "labelsID", labelsID)
			frame = data.NewFrameOfFieldTypes(labelsID, 0, fieldTypes...)
			frames[labelsID] = frame
		}
		row, err := qm.getValues(doc)
		if err != nil {
			response.Error = fmt.Errorf("Bad documment: %s, %v", err, doc)
			return response
		}
		log.DefaultLogger.Debug("Got row", "row", row, "doc", doc)
		frame.AppendRow(row...)
	}
	if cursor.Err() != nil {
		response.Error = cursor.Err()
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
	mongoClient, errMsg, err, internalErr := connect(ctx, req.PluginContext)
	if internalErr != nil {
		return nil, err
	}
	if err != nil {
		result.Status = backend.HealthStatusError
		result.Message = errMsg + err.Error()
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

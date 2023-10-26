package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/pkg/errors"

	"go.mongodb.org/mongo-driver/mongo"
	mongoOpts "go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBDateSpecifierReplacements is a mapping from date specifiers used by
// Golang's time.Parse to those used by MongoDB's $dateFromString stage.
// If the "to" field is nil, then that indicates MongoDB has no equivalent, and
// translating any string containing it should produce an error.
var MongoDBDateSpecifierReplacements = []struct {
	from string
	to   *string
}{
	{"%", str("%%")},
	{"2006", str("%Y")},
	{"06", nil},
	{"January", str("%B")},
	{"Jan", str("%b")},
	{"15", str("%H")},
	{"01", str("%m")},
	{"1", nil},
	{"Monday", nil},
	{"Mon", nil},
	{"__2", str("%j")},
	{"002", str("%j")},
	{"02", str("%d")},
	{"_2", str("%d")},
	{"2", nil},
	{"03", nil},
	{"3", nil},
	{"04", str("%M")},
	{"4", nil},
	{"05", str("%S")},
	{"5", nil},
	{"PM", nil},
	{"-0700", str("%z")},
	{"-07", str("%z")},
}

func str(s string) *string {
	return &s
}

func ConvertGoTimeFormatToMongo(goFormat string) (mongoFormat string, err error) {
	mongoFormatBuilder := strings.Builder{}
	unsupported := make([]string, 0)
	ix := 0
	runes := []rune(goFormat)
	for ix < len(runes) {
		found := false
		for _, repl := range MongoDBDateSpecifierReplacements {
			if len(repl.from) > len(runes[ix:]) || string(runes[ix:ix+len(repl.from)]) != repl.from {
				continue
			}
			found = true
			ix += len(repl.from)
			if repl.to == nil {
				unsupported = append(unsupported, repl.from)
				continue
			}
			mongoFormatBuilder.WriteString(*repl.to)
		}
		if found {
			continue
		}
		mongoFormatBuilder.WriteRune(runes[ix])
		ix++
	}

	if len(unsupported) != 0 {
		errMsg := strings.Builder{}
		errMsg.WriteString("MongoDB does not have an equivalent for the following date specifiers: ")
		first := false
		for _, spec := range unsupported {
			if first {
				first = false
			} else {
				errMsg.WriteString(", ")
			}
			errMsg.WriteString(spec)
		}
		err = fmt.Errorf(errMsg.String())
	}
	return mongoFormatBuilder.String(), nil
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

	err = data.applyAuth(mongoURL)
	if err != nil {
		return nil, err, nil
	}
	opts = opts.ApplyURI(mongoURL.String())

	tlsConfig, err := data.getTLS()
	if err != nil {
		return nil, err, nil
	}
	if tlsConfig != nil {
		opts.SetTLSConfig(tlsConfig)
	}
	log.DefaultLogger.Debug("Connecting with options", "opts", opts)

	mongoClient, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "Error while connecting to MongoDB"), nil
	}

	return mongoClient, nil, nil
}

func (d *MongoDBDatasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Info("query called", "context", pCtx, "query", query)
	response := backend.DataResponse{}

	// Unmarshal the JSON into our QueryModel and parse values into usable representations
	var qm QueryModel

	var err error
	err = json.Unmarshal(query.JSON, &qm)
	if err != nil {
		response.Error = errors.Wrap(err, "Invalid query JSON")
		return response
	}

	log.DefaultLogger.Debug("Query Model Parsed", "QueryModel", qm)

	pipeline, err := qm.getPipeline(query.TimeRange.From, query.TimeRange.To)
	if err != nil {
		response.Error = errors.Wrap(err, "Failed to produce final pipeline")
		return response
	}

	log.DefaultLogger.Debug("Effective pipeline", "pipeline", pipeline)

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
			for _, name := range qm.LabelFields {
				ignored[name] = struct{}{}
			}
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
			"ignored", ignored,
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

func (d *MongoDBDatasource) ping(ctx context.Context, req *backend.CheckHealthRequest) error {
	mongoClient, err, internalErr := connect(ctx, req.PluginContext)
	if internalErr != nil {
		return errors.Wrap(err, "Failed to connect to mongo")
	}
	if err != nil {
		return err
	}
	defer mongoClient.Disconnect(ctx)
	return mongoClient.Ping(ctx, nil)
}

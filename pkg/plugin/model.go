package plugin

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	sprig "github.com/go-task/slim-sprig"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	bsonPrim "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type queryType = string

const (
	queryTypeTimeseries = "Timeseries"
	queryTypeTable      = "Table"
	defaultQueryType    = queryTypeTable
)

type QueryModel struct {
	Database             string    `json:"database"`
	Collection           string    `json:"collection"`
	QueryType            queryType `json:"queryType"`
	TimestampField       string    `json:"timestampField,omitempty"`
	TimestampFormat      string    `json:"timestampFormat,omitempty"`
	LabelFields          []string  `json:"labelFields,omitempty"`
	LegendFormat         string    `json:"legendFormat,omitempty"`
	ValueFields          []string  `json:"valueFields"`
	ValueFieldTypes      []string  `json:"valueFieldTypes,omitempty"`
	AutoTimeBound        bool      `json:"autoTimeBound"`
	AutoTimeBoundAtStart bool      `json:"autoTimeBoundAtStart"`
	AutoTimeSort         bool      `json:"autoTimeSort"`
	Aggregation          string    `json:"aggregation"`
	SchemaInference      bool      `json:"schemaInference"`
	SchemaInferenceDepth int       `json:"schemaInferenceDepth,omitempty"`
}

func (m *QueryModel) resolve(fields []field) (resolvedQueryModel, error) {
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
			return nil, fmt.Errorf("Type mismatch for field %s: expected %s, got %s (%#v, %#v)", name, type_, actualType, value, values[ix])
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
		value, ok := doc[name]
		if !ok || value == nil {
			if !type_.Nullable() {
				return nil, fmt.Errorf("Field %s was null or absent, but is not nullable. If using schema inference, please increase the depth to the first document missing this field, or manually specify the schema", name)
			}
			valueValues[ix] = nil
			continue
		}

		valueValues[ix], actualType, err = convertValue(value, type_.Nullable())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to convert value for %s (%#v)", name, value))
		}
		if actualType != type_ {
			return nil, fmt.Errorf("Type mismatch for field %s: expected %s, got %s (%#v, %#v)", name, type_, actualType, value, valueValues[ix])
		}
	}

	return values, nil
}

func (m *QueryModel) getFields() ([]field, error) {
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

func (m *QueryModel) getTimeBoundPipelineStage(from time.Time, to time.Time) (bson.D, error) {
	fromTime := bsonPrim.NewDateTimeFromTime(from)
	toTime := bsonPrim.NewDateTimeFromTime(to)
	var match bson.D
	if m.TimestampFormat == "" {
		match = bson.D{bson.E{
			Key: m.TimestampField,
			Value: bson.D{
				bson.E{Key: "$gte", Value: fromTime},
				bson.E{Key: "$lte", Value: toTime},
			},
		}}
	} else {
		convertedFormat, err := ConvertGoTimeFormatToMongo(m.TimestampFormat)
		if err != nil {
			return nil, err
		}
		parsedString := bson.D{bson.E{
			Key: "$dateFromString",
			Value: bson.D{
				bson.E{Key: "dateString", Value: "$" + m.TimestampField},
				bson.E{Key: "format", Value: convertedFormat},
			},
		}}
		match = bson.D{bson.E{
			Key: "$expr",
			Value: bson.D{bson.E{
				Key: "$and",
				Value: bson.A{
					bson.D{bson.E{
						Key:   "$gte",
						Value: bson.A{parsedString, fromTime},
					}},
					bson.D{bson.E{
						Key:   "$lte",
						Value: bsonPrim.A{parsedString, toTime},
					}},
				},
			}},
		}}
	}
	return bson.D{bson.E{
		Key:   "$match",
		Value: match,
	}}, nil
}

func (m *QueryModel) getPipeline(from time.Time, to time.Time) (mongo.Pipeline, error) {
	pipeline := mongo.Pipeline{}

	if m.QueryType == queryTypeTimeseries && m.AutoTimeBound && m.AutoTimeBoundAtStart {
		timeBoundStage, err := m.getTimeBoundPipelineStage(from, to)
		if err != nil {
			return nil, err
		}
		pipeline = append(pipeline, timeBoundStage)
	}

	userPipeline := mongo.Pipeline{}
	err := bson.UnmarshalExtJSON([]byte(m.Aggregation), false, &userPipeline)
	if err != nil {
		return mongo.Pipeline{}, errors.Wrap(err, "Failed to parse aggregation pipeline")
	}
	pipeline = append(pipeline, userPipeline...)

	if m.QueryType == queryTypeTimeseries && m.AutoTimeBound && !m.AutoTimeBoundAtStart {
		timeBoundStage, err := m.getTimeBoundPipelineStage(from, to)
		if err != nil {
			return nil, err
		}
		pipeline = append(pipeline, timeBoundStage)
	}
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

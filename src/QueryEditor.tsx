import { defaults, zip } from 'lodash';

import React, { ChangeEvent, PureComponent, SyntheticEvent } from 'react';
import { 
  Input,
  FieldSet,
  InlineField,
  InlineFormLabel,
  InlineFieldRow,
  InlineSwitch,
  CodeEditor,
  Select,
  Button,
} from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from './datasource';
import { defaultQuery, MongoDBDataSourceOptions, MongoDBQuery, MongoDBQueryType } from './types';

type Props = QueryEditorProps<DataSource, MongoDBQuery, MongoDBDataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
  readonly labelWidth = 25;
  readonly longWidth = 50;

  readonly queryTypeOptions = [
    {
        label: "Timeseries",
        value: MongoDBQueryType.Timeseries,
        description: "Return time-indexed series of values, distinguished by a set of labels"
    },
    {
        label: "Table",
        value: MongoDBQueryType.Table,
        description: "Return arbitrary rows for a table or further processing"
    }
  ];

  readonly defaultQueryType: MongoDBQueryType = MongoDBQueryType.Timeseries;

  onDatabaseChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, database: event.target.value });
  };

  onCollectionChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, collection: event.target.value });
    // executes the query
    onRunQuery();
  };
  onQueryTypeChange = (
        query: Props['query'],
        onChange: Props['onChange'],
        onRunQuery: Props['onRunQuery'],
  ) => (newValue: SelectableValue) => {
    onChange({ ...query, queryType: newValue.value });
    onRunQuery();
  };

  onTimestampFieldChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, timestampField: event.target.value });
    // executes the query
    onRunQuery();
  };

  onTimestampFormatChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, timestampFormat: event.target.value });
    // executes the query
    onRunQuery();
  };

  
  onLabelFieldChange = (index: number) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    let newLabelFields = Array.from(query.labelFields)
    newLabelFields.splice(index, 1, event.target.value);
    onChange({ ...query, labelFields: newLabelFields });
    // executes the query
    onRunQuery();
  };

  onLabelFieldAppend = () => {
    const { onChange, query, onRunQuery } = this.props;
    let newLabelFields = Array.from(query.labelFields)
    newLabelFields.splice(query.labelFields.length, 0, "");
    onChange({ ...query, labelFields: newLabelFields });
    // executes the query
    onRunQuery();
  };

  onLabelFieldRemove = (index: number) => () => {
    const { onChange, query, onRunQuery } = this.props;
    let newLabelFields = Array.from(query.labelFields)
    newLabelFields.splice(index, 1);
    onChange({ ...query, labelFields: newLabelFields });
    // executes the query
    onRunQuery();
  };

  onLegendFormatChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, legendFormat: event.target.value });
    // executes the query
    onRunQuery();
  };

  onSchemaInferenceChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, schemaInference: event.target.checked });
    // executes the query
    onRunQuery();
  };
  onSchemaInferenceDepthChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, schemaInferenceDepth: parseInt(event.target.value, 10) });
    // executes the query
    onRunQuery();
  };

  onValueFieldChange = (index: number) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    let newValueFields = Array.from(query.valueFields);
    newValueFields.splice(index, 1, event.target.value);
    onChange({ ...query, valueFields: newValueFields });
    // executes the query
    onRunQuery();
  };

  onValueFieldTypeChange = (index: number) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    let newValueFieldTypes = Array.from(query.valueFieldTypes);
    newValueFieldTypes.splice(index, 1, event.target.value);
    onChange({ ...query, valueFieldTypes: newValueFieldTypes });
    // executes the query
    onRunQuery();
  };

  onValueFieldAppend = () => {
    const { onChange, query, onRunQuery } = this.props;
    let newValueFields = Array.from(query.valueFields);
    let newValueFieldTypes = Array.from(query.valueFieldTypes);
    newValueFields.splice(query.valueFields.length, 0, "");
    newValueFieldTypes.splice(query.valueFieldTypes.length, 0, "");
    onChange({ ...query, valueFields: newValueFields, valueFieldTypes: newValueFieldTypes });
    // executes the query
    onRunQuery();
  };

  onValueFieldRemove = (index: number) => () => {
    const { onChange, query, onRunQuery } = this.props;
    let newValueFields = Array.from(query.valueFields);
    let newValueFieldTypes = Array.from(query.valueFieldTypes);
    newValueFields.splice(index, 1);
    newValueFieldTypes.splice(index, 1);
    onChange({ ...query, valueFields: newValueFields, valueFieldTypes: newValueFieldTypes });
    // executes the query
    onRunQuery();
  };



  onAutoTimeBoundChange = (event: SyntheticEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, autoTimeBound: event.currentTarget.checked });
    // executes the query
    onRunQuery();
  };

  onAutoTimeSortChange = (event: SyntheticEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, autoTimeSort: event.currentTarget.checked });
    // executes the query
    onRunQuery();
  };

  onAggregationChange = (newAggregation: string) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, aggregation: newAggregation });
    // executes the query
    onRunQuery();
  };


  render() {
    const query = defaults(this.props.query, defaultQuery);
    const { onChange, onRunQuery } = this.props;

    return (
      <>
        <FieldSet>
          <InlineFieldRow>
            <InlineField labelWidth={this.labelWidth} label="Database.Collection">
              <Input
                width={this.longWidth}
                name="database"
                type="text"
                placeholder="my_database"
                value={query.database || ''}
                onChange={this.onDatabaseChange}
              ></Input>
            </InlineField>
            <InlineField label=".">
              <Input
                width={this.longWidth}
                name="collection"
                type="text"
                placeholder="my_collection"
                value={query.collection || ''}
                onChange={this.onCollectionChange}
              ></Input>
            </InlineField>
          </InlineFieldRow>
          <InlineField
              labelWidth={this.labelWidth}
              tooltip="Type of query to execute"
              label="QueryType"
              >
            <Select
              options={this.queryTypeOptions}
              value={this.queryTypeOptions.find((queryType) => queryType.value === query.queryType) ?? this.queryTypeOptions[0]}
              onChange={this.onQueryTypeChange(query, onChange, onRunQuery)}
                width={this.longWidth}
            ></Select>
          </InlineField>

          { (query.queryType || this.defaultQueryType) === MongoDBQueryType.Timeseries ? (
            <>
              <InlineField
                  labelWidth={this.labelWidth}
                  label="Timestamp Field"
                  tooltip="Field to expect in every document containing the timestamp"
                  >
                <Input
                  width={this.longWidth}
                  value={query.timestampField || ''}
                  onChange={this.onTimestampFieldChange}
                  type="text"
                  placeholder="timestamp"
                  name="timestampField"
                ></Input>
              </InlineField>
              <InlineField
                  labelWidth={this.labelWidth}
                  label="Timestamp Format"
                  tooltip="If blank, assume timestamps are native BSON dates. Otherwise, parse the timestamp as a string in the format described here: https://pkg.go.dev/time#Parse"
                  >
                <Input
                  width={this.longWidth}
                  value={query.timestampFormat || ''}
                  onChange={this.onTimestampFormatChange}
                  type="text"
                  placeholder="<BSON $date>"
                  name="timestampField"
                ></Input>
              </InlineField>
              <InlineFormLabel
                  width={this.labelWidth}
                  tooltip="Each unique combination of these fields defines a separate time series. Nested fields are not supported, please project to a flat document"
              >
                Label Fields
              </InlineFormLabel>
              <div>
                  {query.labelFields.map((field, index) => (
                      <InlineFieldRow key={index}>
                          <Input
                            width={this.longWidth}
                            onChange={this.onLabelFieldChange(index)}
                            value={field}
                            placeholder="name"
                          ></Input>
                          <Button onClick={this.onLabelFieldRemove(index)}>-</Button>
                      </InlineFieldRow>
                  ))}
                  <Button onClick={this.onLabelFieldAppend}>+</Button>
              </div>
              <InlineField
                    labelWidth={this.labelWidth}
                    label="Legend Format"
                    tooltip={"Series name override. Replacements are:\n{{.Value}}: Value field name.\n{{.Labels.field_name}}: Value of the label with name 'field_name'\n{{.Labels}}: key=value,... for all labels\nSee https://pkg.go.dev/text/template for full syntax.\nFunctions from https://masterminds.github.io/sprig/ are provided"}
              >
                <Input
                  value={query.legendFormat || ""}
                  onChange={this.onLegendFormatChange}
                />
              </InlineField>
            </>
          ) : false }
          { (query.queryType || this.defaultQueryType) === MongoDBQueryType.Timeseries ? (
            <>
              <InlineField
                  label="Automatic Time-Bound"
                  labelWidth={this.labelWidth}
                  tooltip="Add a stage at the beginning to $match documents where Timestamp Field is within the current dashboard time range"
                  >
                <InlineSwitch
                  value={query.autoTimeBound || false}
                  onChange={this.onAutoTimeBoundChange}
                ></InlineSwitch>
              </InlineField>
              <InlineField
                  label="Automatic Time-Sort"
                  labelWidth={this.labelWidth}
                  tooltip="Add a stage at the end to $sort documents ascending by Timestamp Field"
                  >
                <InlineSwitch
                  value={query.autoTimeSort || false}
                  onChange={this.onAutoTimeSortChange}
                ></InlineSwitch>
              </InlineField>
            </>
          ) : false }

          <div className="gf-form">
            <InlineFormLabel
              width={this.labelWidth}
              tooltip="If enabled, Grafana will attempt to figure out the types of your data based on the first few documents. Otherwise, you will need to specify the names and datatypes of each field"
            >
              Infer Schema
            </InlineFormLabel>
            <InlineSwitch
              value={query.schemaInference || false}
              onChange={this.onSchemaInferenceChange}
            />
          </div>

          { query.schemaInference ?
            <>
              <InlineField
                    labelWidth={this.labelWidth}
                    label="Schema Inference Depth"
                    tooltip="How many documents to consider for inference before assuming no new fields will be present. If all documents have the same fields, you can set this to 1"
              >
                <Input
                    value={`${query.schemaInferenceDepth}`}
                    onChange={this.onSchemaInferenceDepthChange}
                    type="number"
                />
              </InlineField>
            </>
            :
            <>
              <InlineFormLabel
                width={this.labelWidth}
                tooltip="These fields contain measurements or other recorded values. You must also specify the data types (float64, uint64, string, etc) for each field. Prefix with a star if a field may not appear in every document for a given series. See https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/data#FieldType for a list of valid types. Nested fields are not supported, please project to a flat document"
              >Value Fields</InlineFormLabel>
              {zip(query.valueFields, query.valueFieldTypes).map((field, index) => (
                  <InlineFieldRow key={index}>
                      <Input
                        onChange={this.onValueFieldChange(index)}
                        width={this.longWidth}
                        value={field[0]}
                        placeholder="name"
                      ></Input>
                      <InlineField label=":">
                          <Input
                            onChange={this.onValueFieldTypeChange(index)}
                            width={this.longWidth}
                            value={field[1]}
                            placeholder="type"
                          ></Input>
                      </InlineField>
                      <Button onClick={this.onValueFieldRemove(index)}>-</Button>
                  </InlineFieldRow>
              ))}
              <Button onClick={this.onValueFieldAppend}>+</Button>
            </>
          }

        </FieldSet>
        <InlineFormLabel
          width={this.labelWidth}
          tooltip="Argument to db.collection.aggregate(...), a JSON array of pipeline stage objects. Helper functions like new Date() or ObjectId() are not supported, consult the MongoDB manual at https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/ to see how to represent these functions in pure JSON"
        >
          Aggregation
        </InlineFormLabel>
        <div 
          style={{ resize: "vertical" }}
        >
          <CodeEditor
            height="300px"
            showLineNumbers={true}
            language="json"
            value={query.aggregation || ''}
            onBlur={this.onAggregationChange}
          ></CodeEditor>
        </div>
      </>
    );
  }
}

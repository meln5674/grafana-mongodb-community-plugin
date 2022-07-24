import { defaults } from 'lodash';

import React, { ChangeEvent, PureComponent, SyntheticEvent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { defaultQuery, MongoDBDataSourceOptions, MongoDBQuery } from './types';
const { FormField, Switch } = LegacyForms;


type Props = QueryEditorProps<DataSource, MongoDBQuery, MongoDBDataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
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

  onTimestampFieldChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, timestampField: event.target.value });
    // executes the query
    onRunQuery();
  };


  onLabelFieldsChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, labelFields: event.target.value.split(",") });
    // executes the query
    onRunQuery();
  };

  onValueFieldsChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, valueFields: event.target.value.split(",") });
    // executes the query
    onRunQuery();
  };

  onValueFieldTypesChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, valueFieldTypes: event.target.value.split(",") });
    // executes the query
    onRunQuery();
  };


  onAggregationChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, aggregation: event.target.value });
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


  render() {
    const query = defaults(this.props.query, defaultQuery);

    return (
      <div className="gf-form-group">
      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={query.database || ''}
          onChange={this.onDatabaseChange}
          label="Database"
        />
      </div>

      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={query.collection || ''}
          onChange={this.onCollectionChange}
          label="Collection"
        />

      </div>
      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={query.timestampField || ''}
          onChange={this.onTimestampFieldChange}
          label="Timestamp Field"
          tooltip="Field to expect in every document containing a unix millis timestamp or ISO timestamp"
        />
      </div>
      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={(query.labelFields || []).join(",")}
          onChange={this.onLabelFieldsChange}
          label="Label Fields"
          tooltip="Comma separated list of fields containg labels to distinguish different series. Nested fields are not supported, please project to a flat document"
        />
      </div>

      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={(query.valueFields || []).join(",")}
          onChange={this.onValueFieldsChange}
          label="Value Fields"
          tooltip="Comma separated list of fields containing measurements or other recorded values. Nested fields are not supported, please project to a flat document"
        />
      </div>

      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={(query.valueFieldTypes || []).join(",")}
          onChange={this.onValueFieldTypesChange}
          label="Value Field Types"
          tooltip="Comma separated list of the data types (float64, uint64, string, etc) of the values listed in the Value Fields. Prefix with a star if a field may not appear in every document for a given series."
        />
      </div>

      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={query.aggregation || ''}
          onChange={this.onAggregationChange}
          label="Aggregation"
          tooltip="Argument to db.collection.aggregate(...), a JSON array of pipeline stage objects"
        />
      </div>

      <div className="gf-form">
        <Switch
          checked={query.autoTimeBound|| false}
          onChange={this.onAutoTimeBoundChange}
          label="Automatic Time-Bound"
        />
      </div>
      <div className="gf-form">
        <Switch
          checked={query.autoTimeSort|| false}
          onChange={this.onAutoTimeSortChange}
          label="Automatic Time-Sort"
        />
      </div>

      </div>
    );
  }
}

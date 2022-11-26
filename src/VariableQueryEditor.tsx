import { MongoDBVariableQuery, defaultVariableQuery } from './types';
import { defaults } from 'lodash';
import React, { ChangeEvent, PureComponent } from 'react';
import { 
  Input,
  InlineField,
  InlineFormLabel,
  InlineFieldRow,
  CodeEditor,
} from '@grafana/ui';


interface VariableQueryProps {
  query: MongoDBVariableQuery;
  onChange: (query: MongoDBVariableQuery) => void;
}

export class VariableQueryEditor extends PureComponent<VariableQueryProps> {
  readonly labelWidth = 25;
  readonly longWidth = 50;

  onDatabaseChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, database: event.target.value });
  };

  onCollectionChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, collection: event.target.value });
  };

  onFieldNameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, fieldName: event.target.value });
  };

  onFieldTypeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, fieldType: event.target.value });
  };

  onAggregationChange = (newAggregation: string) => {
    const { onChange, query } = this.props;
    onChange({ ...query, aggregation: newAggregation });
  };

  render() {
    const query = defaults(this.props.query, defaultVariableQuery);

    return (
      <>
        <InlineFieldRow>
          <InlineField
              labelWidth={this.labelWidth}
              label="Database.Collection"
              >
            <Input
              width={this.longWidth}
              name="database"
              type="text"
              placeholder="my_database"
              onChange={this.onDatabaseChange}
              value={query.database}
            ></Input>
          </InlineField>
          <InlineField
              label="."
              >
            <Input
              width={this.longWidth}
              name="collection"
              type="text"
              placeholder="my_collection"
              onChange={this.onCollectionChange}
              value={query.collection}
            ></Input>
          </InlineField>
        </InlineFieldRow>
  
        <InlineFieldRow>
           <InlineField
               label="Field"
               labelWidth={this.labelWidth}
               >
             <Input
               width={this.longWidth}
               placeholder="name"
               onChange={this.onFieldNameChange}
               value={query.fieldName}
             ></Input>
           </InlineField>
           <InlineField
              label=":"
              >
            <Input
              width={this.longWidth}
              placeholder="type"
              onChange={this.onFieldTypeChange}
              value={query.fieldType}
            ></Input>
          </InlineField>
        </InlineFieldRow>
  
        <InlineFormLabel
          width={this.labelWidth}
          tooltip="Argument to db.collection.aggregate(...), a JSON array of pipeline stage objects. Helper functions like new Date() or ObjectId() are not supported, consult the MongoDB manual at https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/ to see how to represent these functions in pure JSON"
        >
          Aggregation
        </InlineFormLabel>
        <CodeEditor
          height="200px"
          showLineNumbers={true}
          language="json"
          onBlur={this.onAggregationChange}
          value={query.aggregation}
        ></CodeEditor>
      </>
    );
  }
};

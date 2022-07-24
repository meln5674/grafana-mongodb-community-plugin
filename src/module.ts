import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './ConfigEditor';
import { QueryEditor } from './QueryEditor';
import { MongoDBQuery, MongoDBDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, MongoDBQuery, MongoDBDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
